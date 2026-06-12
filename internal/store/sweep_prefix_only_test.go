package store_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wallymagill/mines/internal/store"
)

// The sweep must require BOTH halves of the temp naming scheme: the
// "scores-" prefix AND the ".tmp" suffix. The foreign-files pin covers
// .tmp files without the prefix; this covers the mirror image — a file
// with the prefix but not the suffix, e.g. a user's "scores-backup.json"
// manual backup made before hand-editing scores.json (the same scenario
// Load's [0,999] sanitization defends against). Kills the mutant that
// drops the HasSuffix(name, ".tmp") condition.
func TestSweepNeverDeletesPrefixOnlyForeignFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mines", "scores.json")
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	foreign := []string{
		"scores-backup.json", // manual backup: matches scores-*, not *.tmp
		"scores-old",         // arbitrary prefix-matching user file
	}
	for _, name := range foreign {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(`{"beginner": 7}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}

	if err := store.New(path).Save(store.Scores{"beginner": 4}); err != nil {
		t.Fatal(err)
	}

	for _, name := range foreign {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("sweep deleted foreign file %q it did not create: %v", name, err)
		}
	}
}
