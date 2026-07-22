package engine

import "testing"

func TestSlideLineEmpty(t *testing.T) {
	got, score := slideLine([Size]int{0, 0, 0, 0})
	want := [Size]int{0, 0, 0, 0}
	if got != want || score != 0 {
		t.Fatalf("slideLine empty = %v score %d, want %v score 0", got, score, want)
	}
}

func TestSlideLineCompressGaps(t *testing.T) {
	got, score := slideLine([Size]int{0, 2, 0, 4})
	want := [Size]int{2, 4, 0, 0}
	if got != want || score != 0 {
		t.Fatalf("slideLine gaps = %v score %d, want %v score 0", got, score, want)
	}
}

func TestSlideLineMergeEqualsOnce(t *testing.T) {
	got, score := slideLine([Size]int{2, 2, 2, 2})
	want := [Size]int{4, 4, 0, 0}
	if got != want || score != 8 {
		t.Fatalf("slideLine [2,2,2,2] = %v score %d, want %v score 8", got, score, want)
	}
}

func TestSlideLineMergePairs(t *testing.T) {
	cases := []struct {
		in    [Size]int
		want  [Size]int
		score int
	}{
		{[Size]int{2, 2, 4, 0}, [Size]int{4, 4, 0, 0}, 4},
		{[Size]int{4, 4, 8, 8}, [Size]int{8, 16, 0, 0}, 24},
		{[Size]int{2, 0, 2, 0}, [Size]int{4, 0, 0, 0}, 4},
		{[Size]int{2, 4, 8, 16}, [Size]int{2, 4, 8, 16}, 0},
		{[Size]int{8, 8, 8, 0}, [Size]int{16, 8, 0, 0}, 16},
	}
	for _, tc := range cases {
		got, score := slideLine(tc.in)
		if got != tc.want || score != tc.score {
			t.Fatalf("slideLine(%v) = %v score %d, want %v score %d", tc.in, got, score, tc.want, tc.score)
		}
	}
}
