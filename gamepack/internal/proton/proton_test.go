package proton

import "testing"

func TestStatState(t *testing.T) {
	// /proc/<pid>/stat: pid (comm) state ... — comm may contain spaces/parens.
	cases := []struct {
		stat string
		want byte
	}{
		{"123 (SplitFiction.exe) S 1 2 3 0 0 0 0 0 0", 'S'},
		{"456 (python3 (with spaces)) Z 1 2 3 0 0 0 0 0 0", 'Z'},
		{"789 (bash) R 1 2 3 0 0 0 0 0 0", 'R'},
		{"1000 (a) X 1 2 3", 'X'},
		{"garbage", 0},
	}
	for _, c := range cases {
		if got := statState(c.stat); got != c.want {
			t.Errorf("statState(%q) = %q, want %q", c.stat, got, c.want)
		}
	}
}
