package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/adrg/xdg"
	"github.com/fatih/color"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)


var exitHook func()


type Profile struct {
	Flake string
	Configuration string
	Remote string
}

type Data struct {
	DefaultProfile string
	Profiles map[string]Profile
}

type RunError struct {
	message string
}
func (e *RunError) Error() string {
	return e.message
}


var messagePrefixColor = color.New(color.FgGreen)
var warnColor = color.New(color.FgYellow)
var errorColor = color.New(color.FgRed)

func msg(message any) {
	messagePrefixColor.Fprint(os.Stderr, "> ")
	fmt.Fprintln(os.Stderr, message)
}

func msgf(format string, a ...any) {
	messagePrefixColor.Fprint(os.Stderr, "> ")
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Fprintln(os.Stderr)
}

func warn(message any) {
	warnColor.Fprintln(os.Stderr, message)
}

func warnf(format string, a ...any) {
	warnColor.Fprintf(os.Stderr, format, a...)
	fmt.Fprintln(os.Stderr)
}

func err(message any) {
	errorColor.Fprintln(os.Stderr, message)
}

func errf(format string, a ...any) {
	errorColor.Fprintf(os.Stderr, format, a...)
	fmt.Fprintln(os.Stderr)
}


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
			warn(err)
		}
	} else {
		warn(err)
	}
}

func exit(code int) {
	if exitHook != nil {
		exitHook()
	}

	os.Exit(code)
}

func trapExit(callback func()) {
	exitHook = callback

	sig := make(chan os.Signal, 1)
	go func() {
        <-sig
		callback()
    }()
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
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
			return &RunError{"Missing profile"}
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
		return &RunError{"Missing flake"}
	}
	if profileData.Configuration == "" {
		return &RunError{"Missing configuration"}
	}

	if updateProfile {
		data.Profiles[profile] = profileData
		save = true
	}


	if save { saveData(dataPath, data) }


	msg("Building Nixos configuration...")

	
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
			exit(1)
		}

		warn(err)

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
				return &RunError{}
			} else {
				return err
			}
		}
	}
	
	outPath := strings.TrimRight(nixOut.String(), "\n")


	activationCommand := fmt.Sprintf(
		"nix-env --profile /nix/var/nix/profiles/system --set %s && %s/bin/switch-to-configuration switch",
		outPath, outPath,
	)

	if profileData.Remote == "" {
		msg("Comparing changes...")

		nvd := exec.Command("nvd", "diff", "/run/current-system", outPath)
		nvd.Stdout = os.Stdout
		nvd.Stderr = os.Stderr
		
		err = nvd.Run()
		if err != nil {
			warnf("Error executing nvd: %v", err)
		}

		msg("Activating configuration...")

		activate := exec.Command("sudo", "--", "/bin/sh", "-c", activationCommand)
		activate.Stdin = os.Stdin
		activate.Stdout = os.Stdout
		activate.Stderr = os.Stderr
			
		err = activate.Run()
		if err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				return &RunError{}
			} else {
				return err
			}
		}
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

		trapExit(func() {
			scriptFile.Close()
			err = os.Remove(scriptPath)
			if err != nil {
				errf("%v\n%s could not be removed, which is a major security risk. Remove it as soon as possible!", err, scriptPath)
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

		msgf("Copying configuration to %s...", profileData.Remote)

		sshEnv := append(
			os.Environ(),
			fmt.Sprintf("SSH_ASKPASS=%s", scriptPath), "SSH_ASKPASS_REQUIRE=force",
		)

		nix := exec.Command(
			"nix", "copy",
			"--to", fmt.Sprintf("ssh-ng://%s", profileData.Remote),
			"--no-check-sigs",
			outPath,
		)

		nix.Env = sshEnv

		nix.Stdout = os.Stdout
		nix.Stderr = os.Stderr

		err = nix.Run()
		if err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				return &RunError{}
			} else {
				return err
			}
		}

		msgf("Activating configuration on %s...", profileData.Remote)

		ssh := exec.Command(
			"ssh", profileData.Remote,
			fmt.Sprintf("sudo --prompt= --stdin -- /bin/sh -c '%s'", activationCommand),
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
				return &RunError{}
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
			Usage: "whether to use the selected profile by default on subsequent runs",
			Aliases: []string{"s"},
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
		color.New(color.FgRed, color.Bold).Fprintln(os.Stderr, err)
		exit(1)
	}

	exit(0)
}
