package main

import (
	"bytes"
	"context"
	"encoding/gob"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/diniamo/nq/internal/external"
	"github.com/diniamo/nq/internal/log"
	"github.com/diniamo/nq/internal/trap"

	"github.com/adrg/xdg"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)


type Profile struct {
	Flake string
	Configuration string
	Remote string
}

type Data struct {
	DefaultProfile string
	Profiles map[string]Profile
}


//go:embed repl.nix
var replNix string


func loadData(path string) (ret Data) {
	file, err := os.Open(path)
	
	if err == nil {
		defer file.Close()

		decoder := gob.NewDecoder(file)
		decoder.Decode(&ret)
	}

	if ret.Profiles == nil {
		ret.Profiles = make(map[string]Profile)
	}

	return ret
}

func saveData(path string, data Data) {
	file, err := os.Create(path)
	
	if err == nil {
		defer file.Close()
		
		encoder := gob.NewEncoder(file)
		err = encoder.Encode(data)
		if err != nil {
			log.Errorf("Failed to encode/write the save data: %s", err)
		}
	} else {
		log.Errorf("Failed to create/open the save file: %s", err)
	}
}


func run(ctx context.Context, cmd *cli.Command) error {
	dataPath := path.Join(xdg.DataHome, "rebuild-profiles.gob")
	data := loadData(dataPath)
			
	save := false


	var profile string
	if passedProfile := cmd.StringArg("profile"); passedProfile != "" {
		profile = passedProfile
		
		if data.DefaultProfile == "" || cmd.Bool("save-default") {
			data.DefaultProfile = profile
			save = true
		}
	} else {
		if data.DefaultProfile != "" {
			profile = data.DefaultProfile
		} else {
			return errors.New("Missing profile")
		}
	}

	
	profileData := data.Profiles[profile]
	updateProfile := false

	flake := cmd.String("flake")
	if flake != "" {
		profileData.Flake = flake
		updateProfile = true
	} else {
		flake = profileData.Flake
	}

	configuration := cmd.String("configuration")
	if configuration != "" {
		profileData.Configuration = configuration
		updateProfile = true
	} else {
		configuration = profileData.Configuration
	}
	
	remote := cmd.String("remote")
	if remote != "" {
		profileData.Remote = remote
		updateProfile = true
	} else {
		remote = profileData.Remote
	}
	
	if profileData.Flake == "" {
		return errors.New("Missing flake")
	}
	if profileData.Configuration == "" {
		return errors.New("Missing configuration")
	}

	if updateProfile {
		data.Profiles[profile] = profileData
		save = true
	}


	if save { saveData(dataPath, data) }


	if cmd.Bool("repl") {
		replacer := strings.NewReplacer(
			"@flake@", flake,
			"@configuration@", configuration,

			"@blue@", "\033[34;1m",
			"@reset@", "\033[0m",
			"@bold@", "\033[1m",
			"@attention@", "\033[35;1m",
		)
		replReplaced := replacer.Replace(replNix)

		nix := exec.Command("nix", "repl", "--impure", "--expr", replReplaced)
		nix.Stdin = os.Stdin
		nix.Stdout = os.Stdout
		nix.Stderr = os.Stderr
		
		err := nix.Run()
		if err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				return errors.New("nix: non-zero exit code")
			} else {
				return err
			}
		}

		return nil
	}


	log.Messagef("Building %s#%s", flake, configuration)

	
	flakeRef := fmt.Sprintf(
		"%s#nixosConfigurations.%s.config.system.build.toplevel",
		profileData.Flake, profileData.Configuration,
	)
	
	var nixOut bytes.Buffer
	
	nom := exec.Command(
		"nom", "build",
		"--no-link", "--print-out-paths",
		flakeRef,
	)
	nom.Stderr = os.Stderr
	nom.Stdout = &nixOut

	err := nom.Run()

	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			trap.Exit(1)
		}

		log.Warnf("Failed to run nom: %s", err)

		nix := exec.Command(
			"nix", "build",
			"--no-link", "--print-out-paths",
			flakeRef,
		)
		nix.Stderr = os.Stderr
		nix.Stdout = &nixOut

		err = nix.Run()
		if err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				return errors.New("nix: non-zero exit code")
			} else {
				return err
			}
		}
	}
	
	outPath := strings.TrimRight(nixOut.String(), "\n")


	if profileData.Remote == "" {
		log.Message("Comparing changes")

		external.Nvd("/run/current-system", outPath)

		log.Message("Activating configuration")

		external.ActivateLocal(outPath)
	} else {
		fmt.Printf("(%s) Password: ", profileData.Remote)
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}
		fmt.Println()

		scriptFile, err := os.CreateTemp("", "tmp.")
		if err != nil {
			return err
		}
		scriptPath := scriptFile.Name()

		trap.TrapExit(func() {
			scriptFile.Close()
			err = os.Remove(scriptPath)
			if err != nil {
				log.Errorf("%v\n%s could not be removed, which is a major security risk. Remove it as soon as possible!", err, scriptPath)
			}
		})

		singleQuote := []byte{'\''}
		escapedPassword := bytes.ReplaceAll(password, singleQuote, []byte{'\\', '\''})

		_, err = scriptFile.WriteString("#!/bin/sh\nprintf '%s' '")
		if err != nil {
			return err
		}
		_, err = scriptFile.Write(escapedPassword)
		if err != nil {
			return err
		}
		_, err = scriptFile.Write(singleQuote)
		if err != nil {
			return err
		}

		scriptFile.Close()
		err = os.Chmod(scriptPath, 0500)
		if err != nil {
			return err
		}

		log.Messagef("Copying configuration to %s", profileData.Remote)

		sshEnv := append(
			os.Environ(),
			"SSH_ASKPASS=" + scriptPath, "SSH_ASKPASS_REQUIRE=force",
		)

		nix := exec.Command(
			"nix", "copy",
			"--to", "ssh-ng://%s" + profileData.Remote,
			"--no-check-sigs",
			outPath,
		)

		nix.Env = sshEnv

		nix.Stdout = os.Stdout
		nix.Stderr = os.Stderr

		err = nix.Run()
		if err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				return errors.New("nix: non-zero exit-code")
			} else {
				return err
			}
		}

		log.Messagef("Activating configuration on %s", profileData.Remote)

		ssh := exec.Command(
			"ssh", profileData.Remote,
			fmt.Sprintf(
				"sudo --prompt= --stdin -- /bin/sh -c '%s'",
				external.ActivationCommand(outPath),
			),
		)
		
		ssh.Env = sshEnv
		
		sshIn, err := ssh.StdinPipe()
		if err != nil {
			return err
		}
		ssh.Stdout = os.Stdout
		ssh.Stderr = os.Stderr

		err = ssh.Start()
		if err != nil {
			return err
		}

		sshIn.Write(password)
		sshIn.Write([]byte{'\n'})

		err = ssh.Wait()
		if err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				return errors.New("ssh/switch-to-configuration/nix-env: non-zero exit code")
			} else {
				return err
			}
		}
	}

	return nil
}

func main() {
	cmd := cli.Command{
		Name: "rebuild",
		Usage: "a convenience program for rebuilding on NixOS",
		Action: run,
		ArgsUsage: "<profile>",
	}

	cmd.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name: "save-default",
			Usage: "use the selected profile by default on subsequent runs",
			Aliases: []string{"s"},
			HideDefault: true,
		},
		&cli.BoolFlag{
			Name: "repl",
			Usage: "start a repl with the configuration of the profile loaded instead of rebuilding (remote is ignored)",
			Aliases: []string{"r"},
			HideDefault: true,
		},
			
		&cli.StringFlag{
			Name: "flake",
			Usage: "the path of the flake to use",
			Aliases: []string{"f"},
		},
		&cli.StringFlag{
			Name: "configuration",
			Usage: "the NixOS configuration to build",
			Aliases: []string{"c"},
		},
		&cli.StringFlag{
			Name: "remote",
			Usage: "the remote to deploy the built configuration to",
			Aliases: []string{"r"},
		},
	}
	cmd.Arguments = []cli.Argument{
		&cli.StringArg{
			Name: "profile",
			UsageText: "the profile to act on",
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
		trap.Exit(1)
	}

	// Call exit hook
	trap.Exit(0)
}
