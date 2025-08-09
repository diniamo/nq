package main

import (
	"bytes"
	"encoding/gob"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/adrg/xdg"
	"golang.org/x/term"
	log "github.com/diniamo/glog"

	"github.com/diniamo/nq/internal/external"
	"github.com/diniamo/nq/internal/process"
	"github.com/diniamo/nq/internal/message"
)


type Args struct {
	profile string
	saveDefault bool
	repl bool
	flake string
	configuration string
	targetHost string
	extra []string
}

type Profile struct {
	Flake string
	Configuration string
	TargetHost string
}

type Data struct {
	DefaultProfile string
	Profiles map[string]Profile
}


const usage = `Convenience program for rebuilding on NixOS.

Usage: rebuild [option...] [profile]

Options:
  --help, -h                  show this text and exit
  --save-default, -s          use the passed profile by default on subsequence runs
  --repl, -r                  start a repl with the configuration instead of rebuilding (remote is ignored)
  --flake, -f path            path of the flake
  --configuration, -c string  NixOS configuration to build
  --target-host, -t string    remote to deploy the built configuration on
  ...                         all non-recognized options are passed to the Nix build command
                              these aren't saved between runs

Profile: an arbitrary name to save passed values to
`

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


func run(args *Args) error {
	dataPath := path.Join(xdg.DataHome, "rebuild-profiles.gob")
	data := loadData(dataPath)
	save := false


	var profileName string
	if args.profile != "" {
		profileName = args.profile

		if data.DefaultProfile == "" || args.saveDefault {
			data.DefaultProfile = profileName
			save = true
		}
	} else if data.DefaultProfile != "" {
		profileName = data.DefaultProfile
	}


	var profile Profile
	if profileName != "" {
		profile = data.Profiles[profileName]
	}
	updateProfile := false

	if args.flake != "" {
		profile.Flake = args.flake
		updateProfile = true
	}
	if args.configuration != "" {
		profile.Configuration = args.configuration
		updateProfile = true
	}
	if args.targetHost != "" {
		profile.TargetHost = args.targetHost
		updateProfile = true
	}

	if profile.Flake == "" {
		return errors.New("Missing flake")
	}
	if profile.Configuration == "" {
		return errors.New("Missing configuration")
	}

	if updateProfile && profileName != "" {
		data.Profiles[profileName] = profile
		save = true
	}
	if save {
		saveData(dataPath, data)
	}


	if args.repl {
		replacer := strings.NewReplacer(
			"@flake@", profile.Flake,
			"@configuration@", profile.Configuration,

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


	message.Stepf("Building %s#%s", profile.Flake, profile.Configuration)


	flakeRef := fmt.Sprintf(
		"%s#nixosConfigurations.%s.config.system.build.toplevel",
		profile.Flake, profile.Configuration,
	)

	nixArgs := []string{"build", flakeRef, "--no-link", "--print-out-paths"}
	if args.extra != nil {
		nixArgs = append(nixArgs, args.extra...)
	}

	nom := exec.Command("nom", nixArgs...)
	nom.Stderr = os.Stderr

	var nixOut bytes.Buffer
	nom.Stdout = &nixOut

	err := nom.Run()

	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return errors.New("nom: non-zero exit code")
		}

		log.Warnf("Failed to run nom: %s", err)

		nix := exec.Command("nix", nixArgs...)
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

	outPath := nixOut.String()
	// Trim newline
	outPath = outPath[:len(outPath)-1]


	if profile.TargetHost == "" {
		message.Step("Comparing changes")

		external.Diff("/run/current-system", outPath)

		message.Step("Activating configuration")

		external.ActivateSwitch(outPath)
	} else {
		fmt.Printf("(%s) Password: ", profile.TargetHost)
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

		process.TrapExit(func() {
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

		message.Stepf("Copying configuration to %s", profile.TargetHost)

		sshEnv := append(
			os.Environ(),
			"SSH_ASKPASS=" + scriptPath, "SSH_ASKPASS_REQUIRE=force",
		)

		nix := exec.Command(
			"nix", "copy",
			"--to", "ssh-ng://" + profile.TargetHost,
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

		message.Stepf("Activating configuration on %s", profile.TargetHost)

		ssh := exec.Command(
			"ssh", profile.TargetHost,
			fmt.Sprintf(
				"sudo --prompt= --stdin -- /bin/sh -c '%s'",
				external.ActivateSwitchCommand(outPath),
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

func safeValue(index int, option string) string {
	if index >= len(os.Args) {
		log.Fatalf("Missing value for %s", option)
	}

	return os.Args[index]
}

func main() {
	args := Args{}

	for i := 1; i < len(os.Args); {
		arg := os.Args[i]

		if arg[0] != '-' {
			args.profile = arg
			goto next
		}

		switch arg {
		case "-h", "--help":
			fmt.Print(usage)
			return
		case "-s", "--save-default":
			args.saveDefault = true
		case "-r", "--repl":
			args.repl = true
		case "-f", "--flake":
			args.flake = safeValue(i + 1, arg)
			i += 1
		case "-c", "--configuration":
			args.configuration = safeValue(i + 1, arg)
			i += 1
		case "-t", "--target-host":
			args.targetHost = safeValue(i + 1, arg)
			i += 1
		default:
			if args.extra == nil {
				args.extra = []string{arg}
			} else {
				args.extra = append(args.extra, arg)
			}
		}

	next:
		i += 1
	}

	err := run(&args)
	if err != nil {
		log.Fatal(err)
		process.Exit(1)
	}

	// Call exit hook
	process.Exit(0)
}
