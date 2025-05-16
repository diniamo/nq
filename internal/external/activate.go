package external

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func ActivationCommand(path string) string {
	return fmt.Sprintf(
		"nix-env --profile /nix/var/nix/profiles/system --set %s && %s/bin/switch-to-configuration switch",
		path, path,
	)
}

func ActivateLocal(path string) error {
	command := ActivationCommand(path)

	activate := exec.Command("sudo", "--", "/bin/sh", "-c", command)
	activate.Stdin = os.Stdin
	activate.Stdout = os.Stdout
	activate.Stderr = os.Stderr

	err := activate.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return errors.New("nix-env/switch-to-configuration: non-zero exit code")
		} else {
			return err
		}
	}
	
	return nil
}
