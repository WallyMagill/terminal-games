package store_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wallymagill/mines/internal/store"
)

func tempStore(t *testing.T) (*store.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mines", "scores.json")
	return store.New(path), path
}

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	s, _ := tempStore(t)
	if got := s.Load(); len(got) != 0 {
		t.Errorf("Load() = %v, want empty", got)
	}
}

func TestLoadCorruptedFileReturnsEmpty(t *testing.T) {
	s, path := tempStore(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := s.Load(); len(got) != 0 {
		t.Errorf("Load() on corrupt file = %v, want empty", got)
	}
}

func TestLoadNullScoresFileNeverPanics(t *testing.T) {
	s, path := tempStore(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	// json.Unmarshal accepts the literal `null` with err == nil and a nil
	// map; Load must still return a usable empty Scores so the next
	// RecordIfBest does not panic on assignment to a nil map.
	if err := os.WriteFile(path, []byte(`null`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := s.Load(); len(got) != 0 {
		t.Errorf("Load() on null file = %v, want empty", got)
	}
	saved, err := s.RecordIfBest("beginner", 33)
	if !saved || err != nil {
		t.Errorf("RecordIfBest over null file = (%v, %v), want (true, nil)", saved, err)
	}
	if got := s.Load()["beginner"]; got != 33 {
		t.Errorf("best after recovery = %d, want 33", got)
	}
}

func TestLoadDropsOutOfRangeValues(t *testing.T) {
	s, path := tempStore(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	// The UI timer starts at 0 and caps at 999, so no real game produces a
	// value outside [0, 999]; such entries only come from hand edits or
	// valid-JSON corruption. A negative "best" would otherwise block every
	// legitimate future record for the preset forever.
	raw := `{"beginner": -3, "intermediate": 1000, "expert": 250}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	got := s.Load()
	if len(got) != 1 || got["expert"] != 250 {
		t.Errorf("Load() = %v, want only expert:250 kept", got)
	}
	saved, err := s.RecordIfBest("beginner", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !saved {
		t.Error("RecordIfBest(1s) over tampered negative best = false: corrupt value permanently blocks records")
	}
}

func TestLoadKeepsBoundaryValues(t *testing.T) {
	s, path := tempStore(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	// 0 (sub-second win) and 999 (timer cap) are both reachable in play.
	raw := `{"beginner": 0, "expert": 999}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	got := s.Load()
	if len(got) != 2 || got["beginner"] != 0 || got["expert"] != 999 {
		t.Errorf("Load() = %v, want beginner:0 and expert:999 kept", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	s, _ := tempStore(t)
	want := store.Scores{"beginner": 42, "expert": 311}
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	got := s.Load()
	if got["beginner"] != 42 || got["expert"] != 311 || len(got) != 2 {
		t.Errorf("round trip = %v, want %v", got, want)
	}
}

// Saves use a unique temp name per call, so "no temp left behind" is
// pinned by listing the directory rather than statting one fixed path.
func TestSaveLeavesNoTempFile(t *testing.T) {
	s, path := tempStore(t)
	if err := s.Save(store.Scores{"beginner": 9}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "scores.json" {
			t.Errorf("stray file %q left behind after atomic save", e.Name())
		}
	}
}

func TestSaveRenameFailureLeavesNoTempFile(t *testing.T) {
	// Force the rename (not the write) to fail: occupy s.path with a
	// directory. The temp-file write succeeds, the rename cannot.
	base := t.TempDir()
	path := filepath.Join(base, "scores.json")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	s := store.New(path)
	if err := s.Save(store.Scores{"beginner": 1}); err == nil {
		t.Fatal("Save onto a directory should fail")
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "scores.json" {
			t.Errorf("temp file %q left behind after rename failure", e.Name())
		}
	}
}

// A save that dies between creating and renaming its unique temp file
// orphans it; the next successful Save sweeps such leftovers — but only
// old ones, so a concurrent process's in-flight temp is never touched.
func TestSaveSweepsStaleTempsButNotFresh(t *testing.T) {
	s, path := tempStore(t)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dir, "scores-stale.tmp")
	if err := os.WriteFile(stale, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	fresh := filepath.Join(dir, "scores-fresh.tmp")
	if err := os.WriteFile(fresh, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := s.Save(store.Scores{"beginner": 4}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale temp from a crashed save survived a successful save")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Error("fresh temp (a concurrent save's in-flight file) was swept")
	}
	if got := s.Load()["beginner"]; got != 4 {
		t.Errorf("best after sweep save = %d, want 4", got)
	}
}

// The temp file is created at 0600; Save must promote the scores file with
// the 0644 it has always had.
func TestSaveFilePermissions(t *testing.T) {
	s, path := tempStore(t)
	if err := s.Save(store.Scores{"beginner": 9}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o644 {
		t.Errorf("scores.json mode = %v, want -rw-r--r--", got)
	}
}

// Two terminal windows on one config dir mean two Store instances saving
// to the same path. Concurrent saves may race — last writer wins — but the
// promoted scores.json must always be one writer's complete payload, never
// an interleaved hybrid. (The old fixed shared temp path s.path+".tmp" let
// writer B's truncate+write land inside writer A's write→rename window, so
// a hybrid of both payloads could be renamed into place: invalid JSON that
// the next Load silently discarded, losing every recorded best.)
func TestConcurrentSavesNeverCorruptScoresFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scores.json")
	s1, s2 := store.New(path), store.New(path)

	big := store.Scores{}
	for r := 0; r < 40; r++ {
		big[string(rune('a'+r%26))+string(rune('a'+r/26))] = r
	}
	small := store.Scores{"x": 1}

	var stop atomic.Bool
	var saveErr1, saveErr2 error // each written by exactly one goroutine
	var writers sync.WaitGroup
	writers.Add(2)
	go func() {
		defer writers.Done()
		for i := 0; i < 3000; i++ {
			if err := s1.Save(big); err != nil && saveErr1 == nil {
				saveErr1 = err // keep saving: contention must persist
			}
		}
	}()
	go func() {
		defer writers.Done()
		for i := 0; i < 3000; i++ {
			if err := s2.Save(small); err != nil && saveErr2 == nil {
				saveErr2 = err
			}
		}
	}()

	var corrupt []byte
	var corruptErr error
	var reader sync.WaitGroup
	reader.Add(1)
	go func() {
		defer reader.Done()
		for !stop.Load() {
			data, err := os.ReadFile(path)
			if err != nil || len(data) == 0 {
				continue // not promoted yet
			}
			var sc map[string]int
			if jerr := json.Unmarshal(data, &sc); jerr != nil {
				corrupt, corruptErr = data, jerr
				return
			}
		}
	}()

	writers.Wait()
	stop.Store(true)
	reader.Wait()

	if saveErr1 != nil {
		t.Errorf("big-payload Save errored under concurrency: %v", saveErr1)
	}
	if saveErr2 != nil {
		t.Errorf("small-payload Save errored under concurrency: %v", saveErr2)
	}
	if corrupt != nil {
		t.Errorf("promoted scores.json observed corrupt (%v):\n%q", corruptErr, corrupt)
	}
	// Last writer wins, but the final file must be exactly one writer's
	// complete payload.
	final := store.New(path).Load()
	if len(final) != len(big) && len(final) != len(small) {
		data, _ := os.ReadFile(path)
		t.Errorf("final scores.json is neither writer's payload (%d keys):\n%q", len(final), data)
	}
}

func TestRecordIfBest(t *testing.T) {
	s, _ := tempStore(t)
	tests := []struct {
		name     string
		secs     int
		wantSave bool
		wantBest int
	}{
		{"first time always records", 100, true, 100},
		{"slower time ignored", 150, false, 100},
		{"equal time ignored", 100, false, 100},
		{"faster time records", 60, true, 60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved, err := s.RecordIfBest("beginner", tt.secs)
			if err != nil {
				t.Fatal(err)
			}
			if saved != tt.wantSave {
				t.Errorf("saved = %v, want %v", saved, tt.wantSave)
			}
			if got := s.Load()["beginner"]; got != tt.wantBest {
				t.Errorf("best = %d, want %d", got, tt.wantBest)
			}
		})
	}
}

func TestCorruptFileRecoversOnNextSave(t *testing.T) {
	s, path := tempStore(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	saved, err := s.RecordIfBest("expert", 250)
	if err != nil || !saved {
		t.Fatalf("RecordIfBest over corrupt file = (%v, %v), want (true, nil)", saved, err)
	}
	if got := s.Load()["expert"]; got != 250 {
		t.Errorf("best after recovery = %d, want 250", got)
	}
}
