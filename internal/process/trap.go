package process

import (
	"os"
	"os/signal"
	"syscall"
)

var exitHook func()

func TrapExit(callback func()) {
	exitHook = callback

	sig := make(chan os.Signal, 1)
	go func() {
        <-sig
		callback()
    }()
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
}

func Exit(code int) {
	if exitHook != nil {
		exitHook()
	}

	os.Exit(code)
}
