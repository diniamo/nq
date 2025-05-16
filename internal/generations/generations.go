package generations

import (
	"fmt"
	"os"
	"slices"
	"sort"
)


const profilesDirectory = "/nix/var/nix/profiles"


type Generations []Generation


func all() (gens Generations, err error) {
	entries, err := os.ReadDir(profilesDirectory)
	if err != nil {
		return
	}

	gens = make(Generations, 0, len(entries))
	for _, entry := range entries {
		if gen, ok := profileToGeneration(entry.Name()); ok {
			gens = append(gens, gen)
		}
	}

	return
}

func sorted() (gens Generations, err error) {
	gens, err = all()
	if err != nil {
		return nil, err
	}
	
	// Reverse sort
	sort.Slice(gens, func(i, j int) bool {
		return gens[i] > gens[j]
	})
	
	return
}

func Print() error {
	gens, err := all()
	if err != nil {
		return err
	}
	// Normal sort for top-down printing
	// (because newer generations are usually more relevant)
	slices.Sort(gens)
	
	cur, err := Current()
	if err != nil {
		return err
	}
		
	for _, gen := range gens {
		if gen != cur {
			fmt.Printf("%d\n", gen)
		} else {
			fmt.Printf("%d (current)\n", gen)
		}
	}

	return nil
}
