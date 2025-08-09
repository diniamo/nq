package external

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func ActivateSwitchCommand(path string) string {
	return fmt.Sprintf(
		"nix-env --profile /nix/var/nix/profiles/system --set %s && %s/bin/switch-to-configuration switch",
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
