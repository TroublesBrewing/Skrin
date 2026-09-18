package frontmatter

import "testing"

func TestEnd(t *testing.T) {
	cases := []struct {
		name  string
		lines []string
		want  int
	}{
		{"closed by ---", []string{"---", "a: 1", "---", "text"}, 2},
		{"closed by ...", []string{"---", "a: 1", "...", "text"}, 2},
		{"trailing spaces and CR", []string{"--- \r", "a: 1\r", "---  \r"}, 2},
		{"empty block", []string{"---", "---"}, 1},
		{"never closed", []string{"---", "a: 1", "text"}, 0},
		{"not on the first line", []string{"", "---", "a: 1", "---"}, 0},
		{"a lone rule", []string{"---"}, 0},
		{"no lines", nil, 0},
	}
	for _, c := range cases {
		if got := End(c.lines); got != c.want {
			t.Errorf("%s: End = %d, want %d", c.name, got, c.want)
		}
	}
}
