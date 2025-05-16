package external

import (
	"os"
	"os/exec"
)

func Nvd(from, to string) (err error) {
	nvd := exec.Command("nvd", "diff", from, to)
	nvd.Stdout = os.Stdout
	nvd.Stderr = os.Stderr
		
	err = nvd.Run()
	return
}
