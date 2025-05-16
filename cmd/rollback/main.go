package main

import (
	"context"
	"os"

	"github.com/diniamo/swich/internal/external"
	"github.com/diniamo/swich/internal/generations"
	"github.com/diniamo/swich/internal/log"

	"github.com/urfave/cli/v3"
)


func run(ctx context.Context, cmd *cli.Command) error {
	if cmd.Bool("list") {
		generations.Print()
		return nil
	}

	var err error

	cur, err := generations.Current()
	if err != nil {
		return err
	}
	
	to := generations.Generation(cmd.Int("to"))
	// HACK: is 0 a valid generation? I don't know how to check.
	if to == 0 {
		to = -1
	}
	if to < 0 {
		to, err = generations.Previous(cur, -int(to))
		if err != nil {
			return err
		}
	}

	log.Messagef("%d -> %d", cur, to)

	curPath, err := generations.OutPath(cur)
	if err != nil {
		return err
	}

	newPath, err := generations.OutPath(to)
	if err != nil {
		return err
	}

	log.Messagef("Comparing changes (%d -> %d)", cur, to)

	external.Nvd(curPath, newPath)

	log.Messagef("Activating %d", to)

	external.ActivateLocal(newPath)

	return nil
}

func main() {
	cmd := cli.Command{
		Name: "rollback",
		Usage: "a convenience program for rolling back on NixOS",
		ArgsUsage: "<generation (default: previous)>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "list",
				Usage: "list available generations instead of rolling back",
				Aliases: []string{"l"},
				HideDefault: true,
			},
			&cli.IntFlag {
				Name: "to",
				Usage: "the generation to roll back to - may be a negative, in which it's relative to the current generation",
				Aliases: []string{"t", "generation", "g"},
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
