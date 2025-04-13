package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/adrg/xdg"
	"github.com/akamensky/argparse"
	"golang.org/x/term"
)


const fgRed = "\033[31m"
const fgYellow = "\033[33m"
const fgGreen = "\033[32m"

const bold = "\033[1m"

const reset = "\033[0m"


var exitHook func()


type Profile struct {
	Flake string
	Hostname string
	Remote string
}

type Data struct {
	DefaultProfile string
	Profiles map[string]Profile
}


func message(message any) {
	fmt.Printf("%s>%s %v\n", fgGreen, reset, message)
}

func messagef(format string, a ...any) {
	message(fmt.Sprintf(format, a...))
}

func warn(message any) {
	fmt.Fprintf(os.Stderr, "%s%v%s\n", fgYellow, message, reset)
}

func warnf(format string, a ...any) {
	warn(fmt.Sprintf(format, a...))
}

func error(message any) {
	fmt.Fprintf(os.Stderr, "%s%v%s\n", fgRed, message, reset)
}

func errorf(format string, a ...any) {
	error(fmt.Sprintf(format, a...))
}

func fatal(message any) {
	fmt.Fprintf(os.Stderr, "%s%s%v%s\n", fgRed, bold, message, reset)
	exit(1)
}

func fatalf(format string, a ...any) {
	fatal(fmt.Sprintf(format, a...))
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

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
        <-sig
		callback()
    }()
}

func main() {
	parser := argparse.NewParser("rebuild", "A convenience program for rebuilding on NixOS")

	profile := parser.StringPositional(&argparse.Options{Help: "The profile to act on"})
	saveDefault := parser.Flag("s", "save-default", &argparse.Options{Help: "Whether to use the current profile by default on subsequent runs"})
	
	flake := parser.String("f", "flake", &argparse.Options{Help: "The path of the flake to use"})
	hostname := parser.String("n", "hostname", &argparse.Options{Help: "The hostname to build"})
	remote := parser.String("r", "remote", &argparse.Options{Help: "The remote to deploy the built configuration on"})

	parser.Parse(os.Args)


	dataPath := path.Join(xdg.DataHome, "rebuild-profiles.gob")
	data := loadData(dataPath)
		
	updateDefault := false
	updateProfile := false



	var resolvedProfile string
	if *profile != "" {
		resolvedProfile = *profile
		updateDefault = data.DefaultProfile == "" || *saveDefault
	} else if data.DefaultProfile != "" {
		resolvedProfile = data.DefaultProfile
	} else {
		fatal("Missing profile")
	}


	profileData := data.Profiles[resolvedProfile]

	if *flake != "" {
		profileData.Flake = *flake
		updateProfile = true
	}
	if *hostname != "" {
		profileData.Hostname = *hostname
		updateProfile = true
	}
	if *remote != "" {
		profileData.Remote = *remote
		updateProfile = true
	}

	if profileData.Flake == "" {
		fatal("Missing flake")
	}
	if profileData.Hostname == "" {
		fatal("Missing hostname")
	}


	if updateDefault { data.DefaultProfile = resolvedProfile }
	if updateProfile { data.Profiles[resolvedProfile] = profileData }
	if updateDefault || updateProfile { saveData(dataPath, data) }


	message("Building Nixos configuration...")

	
	flakeRef := fmt.Sprintf(
		"%s#nixosConfigurations.%s.config.system.build.toplevel",
		profileData.Flake, profileData.Hostname,
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
				exit(1)
			} else {
				fatal(err)
			}
		}
	}
	
	outPath := strings.TrimRight(nixOut.String(), "\n")


	activationCommand := fmt.Sprintf(
		"nix-env --profile /nix/var/nix/profiles/system --set %s && %s/bin/switch-to-configuration switch",
		outPath, outPath,
	)

	if profileData.Remote == "" {
		message("Comparing changes...")

		nvd := exec.Command("nvd", "diff", "/run/current-system", outPath)
		nvd.Stdout = os.Stdout
		nvd.Stderr = os.Stderr
		
		err = nvd.Run()
		if err != nil {
			warnf("Error executing nvd: %v", err)
		}

		message("Activating configuration...")

		activate := exec.Command("sudo", "--", "/bin/sh", "-c", activationCommand)
		activate.Stdin = os.Stdin
		activate.Stdout = os.Stdout
		activate.Stderr = os.Stderr
			
		err = activate.Run()
		if err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				exit(1)
			} else {
				fatal(err)
			}
		}
	} else {
		fmt.Printf("(%s) Password: ", profileData.Remote)
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			fatal(err)
		}
		fmt.Println()

		scriptFile, err := os.CreateTemp("", "tmp.")
		if err != nil {
			fatal(err)
		}
		scriptPath := scriptFile.Name()

		trapExit(func() {
			scriptFile.Close()
			err = os.Remove(scriptPath)
			if err != nil {
				errorf("%v\n%s could not be removed, which is a major security risk. Remove it as soon as possible!", err, scriptPath)
			}
		})

		singleQuote := []byte{'\''}
		escapedPassword := bytes.ReplaceAll(password, singleQuote, []byte{'\\', '\''})

		_, err = scriptFile.WriteString("#!/bin/sh\nprintf '%s' '")
		if err != nil {
			fatal(err)
		}
		_, err = scriptFile.Write(escapedPassword)
		if err != nil {
			fatal(err)
		}
		_, err = scriptFile.Write(singleQuote)
		if err != nil {
			fatal(err)
		}

		scriptFile.Close()
		err = os.Chmod(scriptPath, 0500)
		if err != nil {
			fatal(err)
		}

		messagef("Copying configuration to %s...", profileData.Remote)

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
				exit(1)
			} else {
				fatal(err)
			}
		}

		messagef("Activating configuration on %s...", profileData.Remote)

		ssh := exec.Command(
			"ssh", profileData.Remote,
			fmt.Sprintf("sudo --prompt= --stdin -- /bin/sh -c '%s'", activationCommand),
		)
		
		ssh.Env = sshEnv
		
		sshIn, err := ssh.StdinPipe()
		if err != nil {
			fatal(err)
		}
		ssh.Stdout = os.Stdout
		ssh.Stderr = os.Stderr

		err = ssh.Start()
		if err != nil {
			fatal(err)
		}

		sshIn.Write(password)
		sshIn.Write([]byte{'\n'})

		err = ssh.Wait()
		if err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				exit(1)
			} else {
				fatal(err)
			}
		}
	}

	// Call exitHook
	exit(0)
}
