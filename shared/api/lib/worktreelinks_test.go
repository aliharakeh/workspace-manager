package lib

import (
	"strings"
	"testing"
)

func TestNormalizeWorktreeLinks(t *testing.T) {
	got, err := NormalizeWorktreeLinks([]string{" node_modules/ ", "", "web\\node_modules", "./.env", "."})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"node_modules", "web/node_modules", ".env"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", got, want)
	}
	for _, bad := range [][]string{
		{"../x"},
		{"/abs"},
		{"C:/abs"},
		{".git/hooks"},
		{"a", "a/"},
		{"a", "a/b"},
		{"a/b", "a"},
	} {
		if _, err := NormalizeWorktreeLinks(bad); err == nil {
			t.Errorf("%v: expected an error", bad)
		}
	}
}
