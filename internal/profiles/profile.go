package profiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)


type Profile int


func (p Profile) String() string {
	return strconv.Itoa(int(p))
}


func (p *Profiles) fileToProfile(fileName string) (profile Profile, ok bool) {
	fileName, found := strings.CutPrefix(fileName, p.Name + "-")
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

func (p *Profiles) Current() (profile Profile, err error) {
	currentProfileLink := filepath.Join(p.Directory, p.Name);
	currentProfile, err := os.Readlink(currentProfileLink)
	if err != nil {
		return
	}

	profile, ok := p.fileToProfile(currentProfile)
	if !ok {
		return profile, errors.New(currentProfile + " points to an invalid profile (is your system broken?)")
	}

	return
}

func (p *Profiles) Previous(cur Profile, n int) (Profile, error) {
	// Binary search is actually not faster here, since the current profile
	// is very likely to be somewhere near the start
	for i, profile := range p.Data {
		if profile == cur {
			left := len(p.Data) - 1 - i
			if left < n {
				return profile, errors.New(fmt.Sprintf(
					"Looking for a profile %d before the current (%d), but there are only %d left",
					n, cur, left,
				))
			}
			
			return p.Data[i + n], nil
		}
	}

	return 0, errors.New("Profile loop ended without finding current")
}

func (p *Profiles) ProfilePath(profile Profile) string {
	return fmt.Sprintf("%s/%s-%d-link", p.Directory, p.Name, profile)
}

func (p *Profiles) OutPath(profile Profile) (string, error) {
	return os.Readlink(p.ProfilePath(profile))
}

func (p *Profiles) BuildDate(profile Profile) (t time.Time, err error) {
	stat, err := os.Lstat(p.ProfilePath(profile))
	if err != nil {
		return
	}

	return stat.ModTime(), nil
}
