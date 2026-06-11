// Package store persists best times as JSON in the user config directory.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Scores maps a preset key ("beginner", "intermediate", "expert") to the
// best completion time in seconds.
type Scores map[string]int

type Store struct {
	path string
}

func New(path string) *Store { return &Store{path: path} }

// DefaultPath is os.UserConfigDir()/mines/scores.json
// (~/Library/Application Support/mines/scores.json on macOS).
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mines", "scores.json"), nil
}

// Load returns saved scores. A missing or corrupted file yields empty scores —
// score I/O must never take the game down.
func (s *Store) Load() Scores {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return Scores{}
	}
	var sc Scores
	if err := json.Unmarshal(data, &sc); err != nil || sc == nil {
		return Scores{}
	}
	return sc
}

// Save writes scores atomically (temp file + rename).
func (s *Store) Save(sc Scores) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// RecordIfBest saves secs under key if it beats the current best.
// It reports whether a new record was written.
func (s *Store) RecordIfBest(key string, secs int) (bool, error) {
	sc := s.Load()
	if best, ok := sc[key]; ok && best <= secs {
		return false, nil
	}
	sc[key] = secs
	return true, s.Save(sc)
}
