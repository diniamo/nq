package message

import (
	"fmt"
	"os"

	log "github.com/diniamo/glog"
)

func Step(message any) {
	log.SuccessColor.Fprint(os.Stderr, "> ")
	fmt.Fprintln(os.Stderr, message)
}

func Stepf(format string, a ...any) {
	log.SuccessColor.Fprint(os.Stderr, "> ")
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Fprintln(os.Stderr)
}
