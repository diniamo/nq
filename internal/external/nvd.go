package external

import (
	"os"
	"os/exec"

	"github.com/diniamo/nq/internal/log"
)

func Nvd(from, to string)  {
	nvd := exec.Command("nvd", "diff", from, to)
	nvd.Stdout = os.Stdout
	nvd.Stderr = os.Stderr
		
	err := nvd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			log.Error("nvd: non-zero exit code")
		} else {
			log.Errorf("Failed to run nvd: %s", err)
		}
	}
}
