package store_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wallymagill/terminal-games/internal/store"
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

func TestLoadRecordsMigratesOldMinesScores(t *testing.T) {
	s, path := tempStore(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `{"beginner":42,"expert":311}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	got := s.LoadRecords()
	if got.Version != 3 {
		t.Fatalf("version = %d, want 3", got.Version)
	}
	if got.Mines["beginner"].BestSeconds != 42 || got.Mines["expert"].BestSeconds != 311 {
		t.Fatalf("migrated mines records = %#v", got.Mines)
	}
}

func TestSaveRecordsLoadRecordsRoundTrip(t *testing.T) {
	s, _ := tempStore(t)
	bestSeconds := 240
	fewestMoves := 91
	want := store.Records{
		Mines: map[string]store.TimeRecord{
			"expert": {BestSeconds: 311},
		},
		Solitaire: map[string]store.SolitaireRecord{
			"draw3": {BestSeconds: &bestSeconds, FewestMoves: &fewestMoves},
		},
	}
	if err := s.SaveRecords(want); err != nil {
		t.Fatal(err)
	}

	got := s.LoadRecords()
	if got.Version != 3 {
		t.Fatalf("version = %d, want 3", got.Version)
	}
	if got.Mines["expert"].BestSeconds != 311 {
		t.Fatalf("mines records = %#v, want expert best 311", got.Mines)
	}
	draw3 := got.Solitaire["draw3"]
	if draw3.BestSeconds == nil || *draw3.BestSeconds != 240 {
		t.Fatalf("draw3 best seconds = %#v, want 240", draw3.BestSeconds)
	}
	if draw3.FewestMoves == nil || *draw3.FewestMoves != 91 {
		t.Fatalf("draw3 fewest moves = %#v, want 91", draw3.FewestMoves)
	}
}

func TestRecordMinesBestPreservesSolitaireRecords(t *testing.T) {
	s, _ := tempStore(t)
	bestSeconds := 240
	fewestMoves := 91
	if err := s.SaveRecords(store.Records{
		Solitaire: map[string]store.SolitaireRecord{
			"draw3": {BestSeconds: &bestSeconds, FewestMoves: &fewestMoves},
		},
	}); err != nil {
		t.Fatal(err)
	}

	saved, err := s.RecordMinesBest("beginner", 42)
	if err != nil {
		t.Fatal(err)
	}
	if !saved {
		t.Fatal("RecordMinesBest() = false, want true for first mines record")
	}
	got := s.LoadRecords()
	if got.Mines["beginner"].BestSeconds != 42 {
		t.Fatalf("mines records = %#v, want beginner best 42", got.Mines)
	}
	draw3 := got.Solitaire["draw3"]
	if draw3.BestSeconds == nil || *draw3.BestSeconds != 240 || draw3.FewestMoves == nil || *draw3.FewestMoves != 91 {
		t.Fatalf("draw3 record = %+v, want preserved 240s/91 moves", draw3)
	}
}

func TestRecordSolitaireDraw3KeepsIndependentBests(t *testing.T) {
	s, _ := tempStore(t)
	newTime, newMoves, err := s.RecordSolitaireDraw3(300, 150)
	if err != nil {
		t.Fatal(err)
	}
	if !newTime || !newMoves {
		t.Fatalf("first record = (%v,%v), want both true", newTime, newMoves)
	}
	newTime, newMoves, err = s.RecordSolitaireDraw3(320, 120)
	if err != nil {
		t.Fatal(err)
	}
	if newTime || !newMoves {
		t.Fatalf("second record = (%v,%v), want only moves", newTime, newMoves)
	}
	rec := s.LoadRecords().Solitaire["draw3"]
	if rec.BestSeconds == nil || *rec.BestSeconds != 300 || rec.FewestMoves == nil || *rec.FewestMoves != 120 {
		t.Fatalf("draw3 record = %+v, want 300s/120 moves", rec)
	}
}

func TestSolitaireAssistedRecordsAreSeparateFromNormalRecords(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")

	normalTime, normalMoves, err := s.RecordSolitaireDraw3Mode(store.SolitaireNormal, 120, 80)
	if err != nil || !normalTime || !normalMoves {
		t.Fatalf("normal record write = %v %v %v, want new records", normalTime, normalMoves, err)
	}
	assistTime, assistMoves, err := s.RecordSolitaireDraw3Mode(store.SolitaireAssisted, 90, 40)
	if err != nil || !assistTime || !assistMoves {
		t.Fatalf("assisted record write = %v %v %v, want new records", assistTime, assistMoves, err)
	}

	normal := s.LoadSolitaireDraw3(store.SolitaireNormal)
	assisted := s.LoadSolitaireDraw3(store.SolitaireAssisted)
	if normal.BestSeconds == nil || *normal.BestSeconds != 120 || normal.FewestMoves == nil || *normal.FewestMoves != 80 {
		t.Fatalf("normal record = %+v, want 120s/80m", normal)
	}
	if assisted.BestSeconds == nil || *assisted.BestSeconds != 90 || assisted.FewestMoves == nil || *assisted.FewestMoves != 40 {
		t.Fatalf("assisted record = %+v, want 90s/40m", assisted)
	}
}

func TestLegacySolitaireRecordWrapperWritesNormalRecord(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	_, _, err := s.RecordSolitaireDraw3(200, 100)
	if err != nil {
		t.Fatal(err)
	}
	if assisted := s.LoadSolitaireDraw3(store.SolitaireAssisted); assisted.BestSeconds != nil || assisted.FewestMoves != nil {
		t.Fatalf("legacy wrapper should not write assisted record: %+v", assisted)
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
			var records store.Records
			if jerr := json.Unmarshal(data, &records); jerr != nil {
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

// The sweep must only ever match the store's own temp naming scheme
// (scores-*.tmp). Old .tmp files in the config dir that the store did not
// create — e.g. a user's manual backup before hand-editing scores.json,
// the same hand-editing scenario Load's [0,999] sanitization defends
// against — must never be touched, no matter how old. Kills the mutant
// that widens the sweep's name filter from scores-*.tmp to *.tmp.
func TestSweepNeverDeletesForeignTempFiles(t *testing.T) {
	s, path := tempStore(t)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	foreign := []string{
		"scores.json.tmp", // manual backup: matches *.tmp, not scores-*.tmp
		"backup.tmp",      // arbitrary user .tmp file
	}
	for _, name := range foreign {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("user data"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.Save(store.Scores{"beginner": 4}); err != nil {
		t.Fatal(err)
	}

	for _, name := range foreign {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("sweep deleted foreign file %q it did not create: %v", name, err)
		}
	}
}

// The sweep must work regardless of what the config dir is named: a glob
// built by embedding the directory path treats characters like "[" in the
// dir name as pattern syntax and silently matches nothing, leaving orphans
// to accumulate forever. Matching must be on entry names, not a path glob.
func TestSweepWorksInGlobMetacharacterDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cfg [backup]", "mines")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "scores.json")
	stale := filepath.Join(dir, "scores-orphan1.tmp")
	if err := os.WriteFile(stale, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}

	if err := store.New(path).Save(store.Scores{"beginner": 4}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale temp %q survived a successful save in a metacharacter dir", stale)
	}
}

func TestLoadRecordsMigratesV2ToV3Shape(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/scores.json"
	bestSeconds := 240
	fewestMoves := 91
	raw := []byte(`{
  "version": 2,
  "mines": {"beginner": {"bestSeconds": 42}},
  "solitaire": {"draw3": {"bestSeconds": 240, "fewestMoves": 91}}
}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	s := store.New(path)
	got := s.LoadRecords()
	if got.Version != 3 {
		t.Fatalf("version = %d, want 3", got.Version)
	}
	if got.Mines["beginner"].BestSeconds != 42 {
		t.Fatalf("mines = %#v", got.Mines)
	}
	draw3 := got.Solitaire["draw3"]
	if draw3.BestSeconds == nil || *draw3.BestSeconds != bestSeconds {
		t.Fatalf("solitaire time = %#v", draw3)
	}
	if draw3.FewestMoves == nil || *draw3.FewestMoves != fewestMoves {
		t.Fatalf("solitaire moves = %#v", draw3)
	}
	if len(got.G2048) != 0 {
		t.Fatalf("g2048 should be empty, got %#v", got.G2048)
	}
}

func TestRecordG2048ClassicPreservesOtherGames(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	if _, err := s.RecordMinesBest("beginner", 42); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.RecordSolitaireDraw3(100, 50); err != nil {
		t.Fatal(err)
	}
	newScore, newTile, err := s.RecordG2048Classic(1200, 256)
	if err != nil || !newScore || !newTile {
		t.Fatalf("first record = %v %v %v", newScore, newTile, err)
	}
	got := s.LoadRecords()
	if got.Mines["beginner"].BestSeconds != 42 {
		t.Fatalf("mines wiped: %#v", got.Mines)
	}
	if got.Solitaire["draw3"].BestSeconds == nil {
		t.Fatal("solitaire wiped")
	}
	rec := s.LoadG2048Classic()
	if rec.BestScore != 1200 || rec.BestTile != 256 {
		t.Fatalf("g2048 = %#v", rec)
	}
}

func TestRecordG2048ClassicOnlyImproves(t *testing.T) {
	s := store.New(t.TempDir() + "/scores.json")
	if _, _, err := s.RecordG2048Classic(1000, 512); err != nil {
		t.Fatal(err)
	}
	ns, nt, err := s.RecordG2048Classic(900, 256)
	if err != nil || ns || nt {
		t.Fatalf("worse values must not update: %v %v %v", ns, nt, err)
	}
	ns, nt, err = s.RecordG2048Classic(1000, 1024)
	if err != nil || ns || !nt {
		t.Fatalf("same score higher tile: %v %v %v", ns, nt, err)
	}
	ns, nt, err = s.RecordG2048Classic(2000, 1024)
	if err != nil || !ns || nt {
		t.Fatalf("higher score same tile: %v %v %v", ns, nt, err)
	}
	rec := s.LoadG2048Classic()
	if rec.BestScore != 2000 || rec.BestTile != 1024 {
		t.Fatalf("rec = %#v", rec)
	}
}

func TestSaveRecordsWritesVersion3(t *testing.T) {
	path := t.TempDir() + "/scores.json"
	s := store.New(path)
	if _, _, err := s.RecordG2048Classic(10, 4); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	ver, ok := raw["version"].(float64)
	if !ok || int(ver) != 3 {
		t.Fatalf("version = %v, want 3", raw["version"])
	}
}
