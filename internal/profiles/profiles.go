package profiles

import (
	"fmt"
	"os"
	"slices"
	"sort"
)


const SystemProfiles = "/nix/var/nix/profiles"
const UserProfiles = ".local/state/nix/profiles"


type Profiles struct {
	directory string
	name string
	Data []Profile
}


func NewProfiles(directory, name string) (profiles Profiles) {
	profiles.directory = directory;
	profiles.name = name;
	return
}

func (g *Profiles) Populate() error {
	entries, err := os.ReadDir(g.directory)
	if err != nil {
		return err
	}

	g.Data = make([]Profile, 0, len(entries))
	for _, entry := range entries {
		if profile, ok := g.fileToProfile(entry.Name()); ok {
			g.Data = append(g.Data, profile)
		}
	}

	return nil
}

func (g *Profiles) Sort() {
	slices.Sort(g.Data)
}

func (g *Profiles) ReverseSort() {
	sort.Slice(g.Data, func(i, j int) bool {
		return g.Data[i] > g.Data[j]
	})
}

func (g *Profiles) Print() error {
	cur, err := g.Current()
	if err != nil {
		return err
	}
		
	for _, profile := range g.Data {
		if profile != cur {
			fmt.Printf("%d\n", profile)
		} else {
			fmt.Printf("%d (current)\n", profile)
		}
	}

	return nil
}
