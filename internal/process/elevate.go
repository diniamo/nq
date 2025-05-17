package process

import (
	"os"
)

func IsElevated() bool {
	return os.Geteuid() == 0
}

/*
func SelfElevate() {
	if IsElevated() {
		return
	}

	cmd := exec.Command("sudo", "--")
	cmd.Args = append(cmd.Args, os.Args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err := cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			log.Fatalf("Failed to run sudo for elevating self: %s", err)
			os.Exit(1)
		}
	}

	os.Exit(0)
}
*/
