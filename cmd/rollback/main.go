package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	log "github.com/diniamo/glog"

	"github.com/diniamo/nq/internal/external"
	"github.com/diniamo/nq/internal/message"
	"github.com/diniamo/nq/internal/profiles"
)


type Args struct {
	list bool
	to int
}


const usage = `Convenience program for rolling back on NixOS.

Usage: rollback [option...]

Options:
  --help, -h  show this text and exit
  --list, -l  list available profiles instead of rolling back
  --to, -t int  the profile to roll back to
                may be negative, in which case it's considered relative to the current profile
                (default: -1)
`


func run(args *Args) error {
	var err error

	p := profiles.NewProfiles(profiles.SystemProfiles, "system")

	err = p.Populate()
	if err != nil {
		return errors.New("Failed to get system profiles: " + err.Error())
	}

	if args.list {
		p.Sort()
		err = p.Print()
		if err != nil {
			return err
		}

		return nil
	}

	p.ReverseSort()

	cur, err := p.Current()
	if err != nil {
		return err
	}

	to := profiles.Profile(args.to)
	if to < 0 {
		to, err = p.Previous(cur, -int(to))
		if err != nil {
			return err
		}
	}

	message.Stepf("%d -> %d", cur, to)

	curPath, err := p.OutPath(cur)
	if err != nil {
		return err
	}

	newPath, err := p.OutPath(to)
	if err != nil {
		return err
	}

	message.Stepf("Comparing changes (%d -> %d)", cur, to)

	external.Diff(curPath, newPath)

	message.Stepf("Switching to and activating %d", to)

	err = external.ActivateRollback(&p, to)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	args := Args{to: -1}

	for i := 1; i < len(os.Args); {
		arg := os.Args[i]

		switch arg {
		case "-h", "--help":
			fmt.Print(usage)
			return
		case "-l", "--list":
			args.list = true
		case "-t", "--to":
			if i + 1 == len(os.Args) {
				log.Fatalf("Missing value for %s", arg)
			}
			value := os.Args[i + 1]

			var err error
			args.to, err = strconv.Atoi(value)
			if err != nil {
				log.Fatalf("Invalid value for %s: %s", arg, value)
			}
		default:
			log.Fatalf("Invalid argument: %s", arg)
		}
	}

	err := run(&args)
	if err != nil {
		log.Fatal(err)
	}
}
