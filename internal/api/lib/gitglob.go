package lib

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	maxWalkFiles = 10000
	maxGlobHits  = 200
)

func globToRegexp(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteByte('^')
	i := 0
	for i < len(pattern) {
		if strings.HasPrefix(pattern[i:], "**/") {
			b.WriteString("(?:.*/)?")
			i += 3
			continue
		}
		if strings.HasPrefix(pattern[i:], "**") {
			b.WriteString(".*")
			i += 2
			continue
		}
		ch := pattern[i]
		switch ch {
		case '*':
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(ch)))
		}
		i++
	}
	b.WriteByte('$')
	return regexp.Compile(b.String())
}

func SanitizeGlobPattern(pattern string) (string, string) {
	cleaned := strings.ReplaceAll(strings.TrimSpace(pattern), "\\", "/")
	if cleaned == "" {
		return "", "pattern is required"
	}
	if strings.HasPrefix(cleaned, "/") || (len(cleaned) >= 2 && cleaned[1] == ':') {
		return "", "pattern must be relative to the app directory"
	}
	for _, p := range strings.Split(cleaned, "/") {
		if p == ".." {
			return "", "pattern must not contain .."
		}
	}
	return cleaned, ""
}

func gitVisibleFiles(root string) []string {
	cmd := exec.Command("git", "-C", root, "ls-files", "-co", "--exclude-standard")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.ReplaceAll(strings.TrimSpace(line), "\\", "/")
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

func walkFiles(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() && path != root && (name == ".git" || name == "node_modules") {
			return filepath.SkipDir
		}
		if len(out) >= maxWalkFiles {
			if d.IsDir() && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, ok := ToProjectRelative(root, path)
		if ok {
			out = append(out, rel)
		}
		return nil
	})
	return out
}

// SearchProjectFiles globs under root, omitting gitignored files when git works.
func SearchProjectFiles(root, pattern string) (files []string, truncated bool, errMsg string) {
	pat, errMsg := SanitizeGlobPattern(pattern)
	if errMsg != "" {
		return nil, false, errMsg
	}
	re, err := globToRegexp(pat)
	if err != nil {
		return nil, false, err.Error()
	}
	candidates := gitVisibleFiles(root)
	if candidates == nil {
		candidates = walkFiles(root)
	}
	for _, f := range candidates {
		if !re.MatchString(f) {
			continue
		}
		files = append(files, f)
		if len(files) > maxGlobHits {
			return files[:maxGlobHits], true, ""
		}
	}
	if files == nil {
		files = []string{}
	}
	return files, false, ""
}
