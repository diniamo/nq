package external

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func ActivationCommand(path string) string {
	return fmt.Sprintf(
		"%s/bin/switch-to-configuration switch && nix-env --profile /nix/var/nix/profiles/system --set %s",
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
			return errors.New("switch-to-configuration/nix-env: non-zero exit code")
		} else {
			return err
		}
	}
	
	return nil
}
