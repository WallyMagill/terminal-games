// Package store persists best times as JSON in the user config directory.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Scores maps a preset key ("beginner", "intermediate", "expert") to the
// best completion time in seconds.
type Scores map[string]int

// Store reads and writes a single scores file.
type Store struct {
	path string
}

// New returns a Store backed by the JSON file at path.
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
// score I/O must never take the game down. Entries with values outside
// [0, 999] are dropped: the UI timer starts at 0 and caps at 999, so such
// values only come from hand edits or valid-JSON corruption, and a negative
// "best" would permanently block every legitimate future record.
func (s *Store) Load() Scores {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return Scores{}
	}
	var sc Scores
	if err := json.Unmarshal(data, &sc); err != nil || sc == nil {
		return Scores{}
	}
	for key, secs := range sc {
		if secs < 0 || secs > 999 {
			delete(sc, key)
		}
	}
	return sc
}

// Save writes scores atomically (temp file + rename). The temp file is
// unique to this call: every process shares one scores path, so two
// instances saving concurrently (two terminal windows) must never share a
// temp file — interleaved writes into one shared temp could promote a
// hybrid of two payloads. With a unique temp per call, concurrent saves
// degrade to last-writer-wins and the promoted file is always one writer's
// complete payload. The data is not fsynced before the rename: losing the
// very last save to a crash is an accepted tradeoff for a scores file.
func (s *Store) Save(sc Scores) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "scores-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := f.Write(data)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		// CreateTemp opens at 0600; widen to a fixed 0644. Unlike the old
		// WriteFile path, Chmod is not masked by the umask, so unusual
		// umasks (e.g. 077) now still yield 0644 rather than 0600.
		werr = os.Chmod(tmp, 0o644)
	}
	if werr == nil {
		werr = os.Rename(tmp, s.path)
	}
	if werr != nil {
		os.Remove(tmp) // best effort: don't orphan this save's temp file
		return werr
	}
	s.sweepStaleTemps(dir)
	return nil
}

// staleTempAge is how old an orphaned scores-*.tmp must be before a
// successful Save removes it. A temp file only outlives its Save call when
// the process dies between creating and renaming it; an hour is far beyond
// any live save's lifetime, so the sweep can never delete a concurrent
// process's in-flight temp file.
const staleTempAge = time.Hour

// sweepStaleTemps removes temp files orphaned by saves that died before
// their rename (the old fixed-name scheme self-healed by reusing one name;
// unique names need an explicit sweep). Best effort: the save itself has
// already succeeded, so every error here is ignored.
func (s *Store) sweepStaleTemps(dir string) {
	matches, err := filepath.Glob(filepath.Join(dir, "scores-*.tmp"))
	if err != nil {
		return
	}
	for _, m := range matches {
		if fi, err := os.Stat(m); err == nil && time.Since(fi.ModTime()) >= staleTempAge {
			os.Remove(m)
		}
	}
}

// RecordIfBest saves secs under key if it beats the current best.
// It reports whether a new record was achieved; on error the record was
// achieved but could not be persisted.
func (s *Store) RecordIfBest(key string, secs int) (bool, error) {
	sc := s.Load()
	if best, ok := sc[key]; ok && best <= secs {
		return false, nil
	}
	sc[key] = secs
	return true, s.Save(sc)
}
