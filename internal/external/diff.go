package external

import (
	"os"
	"os/exec"

	"github.com/diniamo/nq/internal/log"
)

func Diff(from, to string)  {
	command := exec.Command("dix", from, to)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
		
	err := command.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			log.Error("dix: non-zero exit code")
		} else {
			log.Errorf("Failed to run dix: %s", err)
		}
	}
}
