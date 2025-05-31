package profiles

import (
	"fmt"
	"os"
	"slices"
	"sort"

	"github.com/diniamo/nq/internal/log"
	"github.com/fatih/color"
)


const SystemProfiles = "/nix/var/nix/profiles"
const UserProfiles = ".local/state/nix/profiles"


type Profiles struct {
	Directory string
	Name string
	Data []Profile
}


func NewProfiles(directory, name string) (profiles Profiles) {
	profiles.Directory = directory;
	profiles.Name = name;
	return
}

func (p *Profiles) Populate() error {
	entries, err := os.ReadDir(p.Directory)
	if err != nil {
		return err
	}

	p.Data = make([]Profile, 0, len(entries))
	for _, entry := range entries {
		if profile, ok := p.fileToProfile(entry.Name()); ok {
			p.Data = append(p.Data, profile)
		}
	}

	return nil
}

func (p *Profiles) Sort() {
	slices.Sort(p.Data)
}

func (p *Profiles) ReverseSort() {
	sort.Slice(p.Data, func(i, j int) bool {
		return p.Data[i] > p.Data[j]
	})
}

func (p *Profiles) Print() error {
	cur, err := p.Current()
	if err != nil {
		return err
	}

	for _, profile := range p.Data {
		date, err := p.BuildDate(profile)
		if err != nil {
			log.Warn(err)
		}

		path, err := p.OutPath(profile)
		if err != nil {
			log.Warn(err)
		}

		line := fmt.Sprintf("%d - %d/%02d/%02d %02d:%02d:%02d - %s\n", profile, date.Year(), date.Month(), date.Day(), date.Hour(), date.Minute(), date.Second(), path)
		if profile != cur {
			fmt.Print(line)
		} else {
			color.New(color.Bold).Print(line)
		}
	}

	return nil
}
