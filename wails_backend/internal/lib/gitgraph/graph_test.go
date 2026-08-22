package gitgraph

import "testing"

func TestCommitURLPrefix(t *testing.T) {
	cases := []struct{ remote, hash, want string }{
		{"git@github.com:acme/repo.git", "abc", "https://github.com/acme/repo/commit/abc"},
		{"https://github.com/acme/repo.git", "abc", "https://github.com/acme/repo/commit/abc"},
		{"https://user:token@github.com/acme/repo.git", "abc", "https://github.com/acme/repo/commit/abc"},
		{"https://gitlab.com/acme/repo.git", "abc", "https://gitlab.com/acme/repo/-/commit/abc"},
		{"git@bitbucket.org:acme/repo.git", "abc", "https://bitbucket.org/acme/repo/commits/abc"},
		{"", "abc", ""},
	}
	for _, c := range cases {
		p := commitURLPrefix(toWebBase(c.remote))
		got := p
		if p != "" {
			got = p + c.hash
		}
		if got != c.want {
			t.Fatalf("%q: got %q want %q", c.remote, got, c.want)
		}
	}
}

func TestParseMergeSubject(t *testing.T) {
	cases := []struct {
		in, src, dst string
	}{
		{"Merge branch 'feature/login' into develop", "feature/login", "develop"},
		{"Merge branch 'hotfix/patch'", "hotfix/patch", ""},
		{"Merge pull request #12 from acme/feature/x", "feature/x", ""},
		{"Merge remote-tracking branch 'origin/feat' into main", "feat", "main"},
		{"Merge branch 'main' of https://github.com/acme/repo", "main", ""},
		{"Merge branch 'foo' of github.com:acme/repo into develop", "foo", "develop"},
		{"Merge branch 'foo' into 'bar'", "foo", "bar"},
		{"Merge branch 'refs/heads/dev' into epic/TR-2369-rework", "dev", "epic/TR-2369-rework"},
		{"Merge tag 'v1.2.3' into main", "v1.2.3", "main"},
		{"Merged in feature/x (pull request #9)", "feature/x", ""},
		{"regular commit", "", ""},
	}
	for _, c := range cases {
		src, dst := parseMergeSubject(c.in)
		if src != c.src || dst != c.dst {
			t.Fatalf("%q: got %q/%q want %q/%q", c.in, src, dst, c.src, c.dst)
		}
	}
}
