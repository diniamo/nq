package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/diniamo/glog"
	
	"github.com/diniamo/nq/internal/process"
	"github.com/diniamo/nq/internal/profiles"
	"github.com/diniamo/nq/internal/message"
)


const gcrootsDirectory = "/nix/var/nix/gcroots/auto"
	

func doRemoveProfiles(p profiles.Profiles, displayName string) {
	err := p.Populate()
	if err != nil {
		log.Errorf("Failed to get %s profiles, skipping all: %s", displayName, err)
		return
	}

	current, err := p.Current()
	if err != nil {
		log.Errorf("Failed to get current %s profile, skipping all: %s", displayName, err)
		return
	}
	
	previous, err := p.Previous(current, 1)
	hasPrevious := err == nil

	for _, profile := range p.Data {
		if profile == current || (hasPrevious && profile == previous) {
			continue
		}

		profilePath := p.ProfilePath(profile)
		err = os.Remove(profilePath)
		if err != nil {
			log.Errorf("Failed to remove %s, skipping: %s", profilePath, err)
			continue
		}

		fmt.Println(profilePath)
	}
}

func main() {
	// clean isn't meant to be run as root, this code path is for internal use
	if process.IsElevated() {
		message.Step("Cleaning system profiles")
	
		doRemoveProfiles(
			profiles.NewProfiles(profiles.SystemProfiles, "system"),
			"system",
		)

		return
	} else {
		cmd := exec.Command("sudo", "--", os.Args[0])
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			log.Errorf("Failed to run elevated instance for clearing system profiles: %s", err)
		}
	}

	
	home := os.Getenv("HOME")

	message.Step("Cleaning user profiles")
	
	userProfilesDirectory := filepath.Join(home, profiles.UserProfiles)

	doRemoveProfiles(
		profiles.NewProfiles(userProfilesDirectory, "profile"),
		"user",
	)

	
	message.Step("Cleaning home-manager profiles")

	doRemoveProfiles(
		profiles.NewProfiles(userProfilesDirectory, "home-manager"),
		"home-manager",
	)


	message.Step("Cleaning gcroots (.direnv, result)")

	entries, err := os.ReadDir(gcrootsDirectory)
	if err == nil {
		for _, entry := range entries {
			linkPath := filepath.Join(gcrootsDirectory, entry.Name())
		
			path, err := os.Readlink(linkPath)
			if err != nil {
				log.Errorf("Failed to read link %s, skipping", linkPath)
				continue
			}

			if strings.HasSuffix(path, "result") || strings.Contains(path, ".direnv") {
				err = os.Remove(path)
				if err != nil {
					log.Errorf("Failed to remove %s, skipping: %s", path, err)
					continue
				}

				fmt.Println(path)
			}
		}
	} else {
		log.Errorf("Failed to read directory %s, skipping gcroots: %s", gcrootsDirectory, err)
	}

	
	message.Step("Running nix store gc")

	nix := exec.Command("nix", "store", "gc")
	nix.Stdout = os.Stdout
	nix.Stderr = os.Stderr
	
	err = nix.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			log.Errorf("nix: non-zero exit code")
		} else {
			log.Errorf("Failed to run nix: %s", err)
		}

		os.Exit(1)
	}
}
