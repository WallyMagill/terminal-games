// Package store persists best times as JSON in the user config directory.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Scores maps a preset key ("beginner", "intermediate", "expert") to the
// best completion time in seconds.
type Scores map[string]int

const recordsVersion = 3

const (
	solitaireDraw3Key         = "draw3"
	solitaireDraw3AssistedKey = "draw3-assisted"
	g2048ClassicKey           = "classic"
)

type SolitaireRecordMode int

const (
	SolitaireNormal SolitaireRecordMode = iota
	SolitaireAssisted
)

type Records struct {
	Version   int                        `json:"version"`
	Mines     map[string]TimeRecord      `json:"mines,omitempty"`
	Solitaire map[string]SolitaireRecord `json:"solitaire,omitempty"`
	G2048     map[string]G2048Record     `json:"g2048,omitempty"`
}

type TimeRecord struct {
	BestSeconds int `json:"bestSeconds"`
}

type SolitaireRecord struct {
	BestSeconds *int `json:"bestSeconds,omitempty"`
	FewestMoves *int `json:"fewestMoves,omitempty"`
}

type G2048Record struct {
	BestScore int `json:"bestScore"`
	BestTile  int `json:"bestTile"`
}

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

// LoadRecords returns saved records. A missing or corrupted file yields empty
// records — score I/O must never take the game down. Old Mines-only score
// files are migrated in memory. Entries with values outside [0, 999] are
// dropped: the Mines UI timer starts at 0 and caps at 999, so such values only
// come from hand edits or valid-JSON corruption, and a negative "best" would
// permanently block every legitimate future record.
func (s *Store) LoadRecords() Records {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return emptyRecords()
	}
	var records Records
	if err := json.Unmarshal(data, &records); err == nil && (records.Version == 2 || records.Version == 3) {
		records.Version = recordsVersion
		return sanitizeRecords(records)
	}
	var old Scores
	if err := json.Unmarshal(data, &old); err != nil || old == nil {
		return emptyRecords()
	}
	records = emptyRecords()
	for key, secs := range old {
		if secs >= 0 && secs <= 999 {
			records.Mines[key] = TimeRecord{BestSeconds: secs}
		}
	}
	return records
}

func emptyRecords() Records {
	return Records{
		Version:   recordsVersion,
		Mines:     map[string]TimeRecord{},
		Solitaire: map[string]SolitaireRecord{},
		G2048:     map[string]G2048Record{},
	}
}

func sanitizeRecords(records Records) Records {
	if records.Version != recordsVersion {
		return emptyRecords()
	}
	if records.Mines == nil {
		records.Mines = map[string]TimeRecord{}
	}
	if records.Solitaire == nil {
		records.Solitaire = map[string]SolitaireRecord{}
	}
	if records.G2048 == nil {
		records.G2048 = map[string]G2048Record{}
	}
	for key, rec := range records.Mines {
		if rec.BestSeconds < 0 || rec.BestSeconds > 999 {
			delete(records.Mines, key)
		}
	}
	for key, rec := range records.Solitaire {
		badTime := rec.BestSeconds != nil && (*rec.BestSeconds < 0 || *rec.BestSeconds > 999)
		badMoves := rec.FewestMoves != nil && *rec.FewestMoves < 0
		if badTime || badMoves || (rec.BestSeconds == nil && rec.FewestMoves == nil) {
			delete(records.Solitaire, key)
		}
	}
	for key, rec := range records.G2048 {
		if rec.BestScore < 0 || rec.BestTile < 0 {
			delete(records.G2048, key)
		}
	}
	return records
}

// Load returns saved Mines scores.
func (s *Store) Load() Scores {
	records := s.LoadRecords()
	scores := Scores{}
	for key, rec := range records.Mines {
		scores[key] = rec.BestSeconds
	}
	return scores
}

// SaveRecords writes records atomically (temp file + rename).
func (s *Store) SaveRecords(records Records) error {
	records.Version = recordsVersion
	if records.Mines == nil {
		records.Mines = map[string]TimeRecord{}
	}
	if records.Solitaire == nil {
		records.Solitaire = map[string]SolitaireRecord{}
	}
	if records.G2048 == nil {
		records.G2048 = map[string]G2048Record{}
	}
	return s.saveJSON(records)
}

// Save writes Mines scores.
func (s *Store) Save(sc Scores) error {
	records := emptyRecords()
	for key, secs := range sc {
		records.Mines[key] = TimeRecord{BestSeconds: secs}
	}
	return s.SaveRecords(records)
}

// saveJSON writes v atomically (temp file + rename). The temp file is unique
// to this call: every process shares one scores path, so two instances saving
// concurrently (two terminal windows) must never share a temp file —
// interleaved writes into one shared temp could promote a hybrid of two
// payloads. With a unique temp per call, concurrent saves degrade to
// last-writer-wins and the promoted file is always one writer's complete
// payload. The data is not fsynced before the rename: losing the very last
// save to a crash is an accepted tradeoff for a scores file.
func (s *Store) saveJSON(v any) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
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
// unique names need an explicit sweep). Matching is on directory-entry
// names — a path glob would treat metacharacters in the config dir's own
// name (e.g. "cfg [backup]") as pattern syntax and silently match nothing.
// Best effort: the save itself has already succeeded, so every error here
// is ignored.
func (s *Store) sweepStaleTemps(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "scores-") || !strings.HasSuffix(name, ".tmp") {
			continue
		}
		p := filepath.Join(dir, name)
		if fi, err := os.Stat(p); err == nil && time.Since(fi.ModTime()) >= staleTempAge {
			os.Remove(p)
		}
	}
}

// RecordMinesBest saves secs under key if it beats the current Mines best.
// It reports whether a new record was achieved; on error the record was
// achieved but could not be persisted.
func (s *Store) RecordMinesBest(key string, secs int) (bool, error) {
	records := s.LoadRecords()
	if best, ok := records.Mines[key]; ok && best.BestSeconds <= secs {
		return false, nil
	}
	records.Mines[key] = TimeRecord{BestSeconds: secs}
	return true, s.SaveRecords(records)
}

func (mode SolitaireRecordMode) draw3Key() string {
	if mode == SolitaireAssisted {
		return solitaireDraw3AssistedKey
	}
	return solitaireDraw3Key
}

func (s *Store) LoadSolitaireDraw3(mode SolitaireRecordMode) SolitaireRecord {
	return s.LoadRecords().Solitaire[mode.draw3Key()]
}

func (s *Store) RecordSolitaireDraw3(secs, moves int) (bool, bool, error) {
	return s.RecordSolitaireDraw3Mode(SolitaireNormal, secs, moves)
}

func (s *Store) RecordSolitaireDraw3Mode(mode SolitaireRecordMode, secs, moves int) (bool, bool, error) {
	records := s.LoadRecords()
	key := mode.draw3Key()
	rec := records.Solitaire[key]
	newBestTime := rec.BestSeconds == nil || secs < *rec.BestSeconds
	newFewestMoves := rec.FewestMoves == nil || moves < *rec.FewestMoves
	if !newBestTime && !newFewestMoves {
		return false, false, nil
	}
	if newBestTime {
		best := secs
		rec.BestSeconds = &best
	}
	if newFewestMoves {
		fewest := moves
		rec.FewestMoves = &fewest
	}
	records.Solitaire[key] = rec
	err := s.SaveRecords(records)
	return newBestTime, newFewestMoves, err
}

// RecordIfBest saves secs under key if it beats the current Mines best.
// It reports whether a new record was achieved; on error the record was
// achieved but could not be persisted.
func (s *Store) RecordIfBest(key string, secs int) (bool, error) {
	return s.RecordMinesBest(key, secs)
}

func (s *Store) LoadG2048Classic() G2048Record {
	return s.LoadRecords().G2048[g2048ClassicKey]
}

// RecordG2048Classic updates best score and best tile when improved.
func (s *Store) RecordG2048Classic(score, highestTile int) (newBestScore, newBestTile bool, err error) {
	if score < 0 || highestTile < 0 {
		return false, false, nil
	}
	records := s.LoadRecords()
	if records.G2048 == nil {
		records.G2048 = map[string]G2048Record{}
	}
	rec := records.G2048[g2048ClassicKey]
	if score > rec.BestScore {
		rec.BestScore = score
		newBestScore = true
	}
	if highestTile > rec.BestTile {
		rec.BestTile = highestTile
		newBestTile = true
	}
	if !newBestScore && !newBestTile {
		return false, false, nil
	}
	records.G2048[g2048ClassicKey] = rec
	return newBestScore, newBestTile, s.SaveRecords(records)
}
