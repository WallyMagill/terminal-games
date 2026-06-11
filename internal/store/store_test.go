package store_test

import (
	"os"
	"path/filepath"
	"testing"

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

func TestSaveLeavesNoTempFile(t *testing.T) {
	s, path := tempStore(t)
	if err := s.Save(store.Scores{"beginner": 9}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file left behind after atomic save")
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
