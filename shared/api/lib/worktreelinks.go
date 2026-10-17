package lib

import (
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// NormalizeWorktreeLinks cleans the shared paths an app's worktrees link to:
// relative to the app folder, slash-separated, inside it, not under .git,
// without duplicates or paths nested in another one. Blank entries are dropped.
func NormalizeWorktreeLinks(paths []string) ([]string, error) {
	out := []string{}
	for _, raw := range paths {
		p := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
		if p == "" {
			continue
		}
		if filepath.IsAbs(p) || filepath.VolumeName(p) != "" || strings.HasPrefix(p, "/") {
			return nil, fmt.Errorf("%s: use a path relative to the app folder", raw)
		}
		p = path.Clean(p)
		if p == "." {
			continue
		}
		if p == ".." || strings.HasPrefix(p, "../") {
			return nil, fmt.Errorf("%s: path must stay inside the app folder", raw)
		}
		if first, _, _ := strings.Cut(p, "/"); strings.EqualFold(first, ".git") {
			return nil, fmt.Errorf("%s: .git cannot be shared", raw)
		}
		for _, q := range out {
			switch {
			case sameRel(p, q):
				return nil, fmt.Errorf("%s is listed twice", p)
			case isUnder(p, q):
				return nil, fmt.Errorf("%s is inside %s, which is already shared", p, q)
			case isUnder(q, p):
				return nil, fmt.Errorf("%s is inside %s, which is already shared", q, p)
			}
		}
		out = append(out, p)
	}
	return out, nil
}

func sameRel(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// isUnder reports whether slash path a is strictly inside b.
func isUnder(a, b string) bool {
	return len(a) > len(b) && a[len(b)] == '/' && sameRel(a[:len(b)], b)
}
