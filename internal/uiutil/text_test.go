package uiutil

import (
	"reflect"
	"testing"
)

func TestFitLineChoosesCompactWhenFullOverflows(t *testing.T) {
	got := FitLine(8, "long full text", "compact")
	if got != "compact" {
		t.Fatalf("FitLine() = %q, want compact", got)
	}
}

func TestAllFit(t *testing.T) {
	if !AllFit([]string{"abc", "de"}, 3) {
		t.Fatal("all lines fit width 3")
	}
	if AllFit([]string{"abcd"}, 3) {
		t.Fatal("abcd does not fit width 3")
	}
}

func TestWrapTextHardBreaksWideWords(t *testing.T) {
	got := WrapText("abcdef", 4)
	want := []string{"abc", "def"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("WrapText() = %#v, want %#v", got, want)
	}
}

func TestClampLine(t *testing.T) {
	got := ClampLine("abcdef", 3)
	if got != "abc" {
		t.Fatalf("ClampLine() = %q, want abc", got)
	}
}
