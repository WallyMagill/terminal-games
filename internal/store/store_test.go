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
