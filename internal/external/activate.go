package external

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/diniamo/nq/internal/profiles"
)

func ActivateSwitchCommand(path string) string {
	return fmt.Sprintf(
		"%s/bin/switch-to-configuration switch && nix-env --profile /nix/var/nix/profiles/system --set %s",
		path, path,
	)
}

func ActivateSwitch(path string) error {
	activate := exec.Command("sudo", "--", "/bin/sh", "-c", ActivateSwitchCommand(path))
	activate.Stdin = os.Stdin
	activate.Stdout = os.Stdout
	activate.Stderr = os.Stderr

	err := activate.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return errors.New("switch-to-configuration/nix-env: non-zero exit code")
		} else {
			return err
		}
	}
	
	return nil
}

func ActivateRollback(profiles *profiles.Profiles, profile profiles.Profile) error {
	command := fmt.Sprintf(
		"%s/bin/switch-to-configuration switch && nix profile rollback --profile %s/%s --to %d",
		profiles.ProfilePath(profile), profiles.Directory, profiles.Name, profile,
	)
	
	activate := exec.Command("sudo", "--", "/bin/sh", "-c", command)
	activate.Stdin = os.Stdin
	activate.Stdout = os.Stdout
	activate.Stderr = os.Stderr

	err := activate.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return errors.New("switch-to-configuration/nix: non-zero exit code")
		} else {
			return err
		}
	}

	return nil
}
