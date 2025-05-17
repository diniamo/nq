package profiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)


type Profile int


func (g *Profiles) fileToProfile(fileName string) (profile Profile, ok bool) {
	fileName, found := strings.CutPrefix(fileName, g.name + "-")
	if !found {
		return
	}

	fileName, found = strings.CutSuffix(fileName, "-link")
	if !found {
		return
	}

	index, err := strconv.Atoi(fileName)
	if err != nil {
		return
	}

	return Profile(index), true
}

func (g *Profiles) Current() (profile Profile, err error) {
	currentProfileLink := filepath.Join(g.directory, g.name);
	currentProfile, err := os.Readlink(currentProfileLink)
	if err != nil {
		return
	}

	profile, ok := g.fileToProfile(currentProfile)
	if !ok {
		return profile, errors.New(currentProfile + " points to an invalid profile (is your system broken?)")
	}

	return
}

func (g *Profiles) Previous(cur Profile, n int) (Profile, error) {
	// Binary search is actually not faster here, since the current profile
	// is very likely to be somewhere near the start
	for i, profile := range g.Data {
		if profile == cur {
			left := len(g.Data) - 1 - i
			if left < n {
				return profile, errors.New(fmt.Sprintf(
					"Looking for a profile %d before the current (%d), but there are only %d left",
					n, cur, left,
				))
			}
			
			return g.Data[i + n], nil
		}
	}

	return 0, errors.New("Profile loop ended without finding current")
}

func (g *Profiles) ProfilePath(profile Profile) string {
	return fmt.Sprintf("%s/%s-%d-link", g.directory, g.name, profile)
}

func (g *Profiles) OutPath(profile Profile) (string, error) {
	return os.Readlink(g.ProfilePath(profile))
}
