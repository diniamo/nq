package generations

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)


type Generation int


func profileToGeneration(profile string) (gen Generation, ok bool) {
	profile, found := strings.CutPrefix(profile, "system-")
	if !found {
		return
	}

	profile, found = strings.CutSuffix(profile, "-link")
	if !found {
		return
	}

	index, err := strconv.ParseUint(profile, 10, 0)
	if err != nil {
		return
	}

	return Generation(index), true
}

func Current() (gen Generation, err error) {
	systemProfileLink := filepath.Join(profilesDirectory, "system");
	systemProfile, err := os.Readlink(systemProfileLink)
	if err != nil {
		return
	}

	gen, ok := profileToGeneration(systemProfile)
	if !ok {
		return gen, errors.New(systemProfileLink + " points to an invalid profile (is your system broken?)")
	}

	return
}

func Previous(cur Generation, n int) (gen Generation, err error) {
	gens, err := sorted()
	if err != nil {
		return
	}

	// Binary search is actually not faster here, since the current generation
	// is very likely to be somewhere near the start
	for i, gen := range gens {
		if gen == cur {
			left := len(gens) - 1 - i
			if left < n {
				return gen, errors.New(fmt.Sprintf(
					"Looking for a generation %d before the current (%d), but there are only %d left",
					n, cur, left,
				))
			}
			
			return gens[i + n], nil
		}
	}

	return gen, errors.New("Generation loop ended without finding current")
}

func OutPath(gen Generation) (string, error) {
	profilePath := fmt.Sprintf("%s/system-%d-link", profilesDirectory, gen)
	return os.Readlink(profilePath)
}
