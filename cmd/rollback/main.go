package main

import (
	"context"
	"errors"
	"os"
	
	"github.com/urfave/cli/v3"
	log "github.com/diniamo/glog"

	"github.com/diniamo/nq/internal/external"
	"github.com/diniamo/nq/internal/profiles"
	"github.com/diniamo/nq/internal/message"
)


func run(ctx context.Context, cmd *cli.Command) error {
	var err error
	
	p := profiles.NewProfiles(profiles.SystemProfiles, "system")
	
	err = p.Populate()
	if err != nil {
		return errors.New("Failed to get system profiles: " + err.Error())
	}

	if cmd.Bool("list") {
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
	
	to := profiles.Profile(cmd.Int("to"))
	// HACK: is 0 a valid generation? I don't know how to check.
	if to == 0 {
		to = -1
	}
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
	cmd := cli.Command{
		Name: "rollback",
		Usage: "a convenience program for rolling back on NixOS",
		ArgsUsage: "<profile (default: previous)>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "list",
				Usage: "list available profiles instead of rolling back",
				Aliases: []string{"l"},
				HideDefault: true,
			},
			&cli.IntFlag {
				Name: "to",
				Usage: "the profile to roll back to - may be a negative, in which it's relative to the current profile",
				Aliases: []string{"t", "profile", "p"},
				DefaultText: "previous",
			},
		},
		Action: run,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
