package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"sync"
)

type banlist struct {
	Bans []string // List of IDs

	filename string // Used for saving/reloading file
	mu       sync.Mutex
}

func banlistFromFile(filename string) (*banlist, error) {
	b := banlist{filename: filename}
	err := b.load()
	return &b, err
}

func (b *banlist) load() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	banfile, err := os.Open(b.filename)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return err
	}
	defer banfile.Close()

	newBans := []string{}

	err = json.NewDecoder(banfile).Decode(&newBans)
	if err != nil {
		return err
	}

	b.Bans = newBans
	return nil
}

func (b *banlist) reload() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	banfile, err := os.Open(b.filename)
	if err != nil {
		return err
	}
	defer banfile.Close()

	newBans := []string{}

	err = json.NewDecoder(banfile).Decode(&newBans)
	if err != nil {
		return err
	}

	b.Bans = newBans
	return nil
}

func (b *banlist) save() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	banfile, err := os.Create(b.filename)
	if err != nil {
		return err
	}
	defer banfile.Close()

	j := json.NewEncoder(banfile)
	j.SetIndent("", "   ")
	return j.Encode(b.Bans)
}

// contains returns true if the addr or id is found in the bans list
func (b *banlist) contains(id string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, bannedID := range b.Bans {
		if id == bannedID {
			return true
		}
	}

	return false
}
