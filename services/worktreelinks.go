package services

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"workspace-manager/native"
	"workspace-manager/types"
)

// Worktree link states, as reported in types.WorktreeLinkState.State.
const (
	LinkLinked    = "linked"     // the worktree path points at the app's own copy
	LinkMissing   = "missing"    // nothing in the worktree yet; linking creates it
	LinkExists    = "exists"     // a real file/folder is there; it is left alone
	LinkOther     = "other_link" // a link to somewhere else is there; left alone
	LinkNoSource  = "no_source"  // the app's own folder has nothing at that path
	LinkFailed    = "error"      // creating the link failed
	excludeHeader = "# workspace-manager: shared worktree links"
	// overrideFile, in a linked worktree's private git folder, holds the shared
	// paths chosen for that worktree when it was added (one per line).
	overrideFile = "workspace-manager-links"
)

// worktreeAdminDir returns the private git folder (.git/worktrees/<name>) of the
// linked worktree at root, read from its .git file. The main worktree has none.
func worktreeAdminDir(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, ".git"))
	if err != nil {
		return "", fmt.Errorf("%s is not a linked worktree: %w", root, err)
	}
	gitdir, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir:")
	if !ok {
		return "", fmt.Errorf("%s is not a linked worktree", root)
	}
	gitdir = filepath.FromSlash(strings.TrimSpace(gitdir))
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(root, gitdir)
	}
	return filepath.Clean(gitdir), nil
}

// WorktreeLinksOverride returns the shared paths chosen for the linked worktree
// at root when it was added, and whether it has such a list. Without one it
// uses the app's list.
func WorktreeLinksOverride(root string) ([]string, bool) {
	admin, err := worktreeAdminDir(root)
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(filepath.Join(admin, overrideFile))
	if err != nil {
		return nil, false
	}
	paths, err := NormalizeWorktreeLinks(strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n"))
	if err != nil {
		return nil, false
	}
	return paths, true
}

// SaveWorktreeLinksOverride gives the linked worktree at root its own shared
// paths in place of the app's. Git deletes the list with the worktree.
func SaveWorktreeLinksOverride(root string, paths []string) error {
	admin, err := worktreeAdminDir(root)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(admin, overrideFile), []byte(strings.Join(paths, "\n")+"\n"), 0o644)
}

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

// WorktreeLinkStates reports, for each shared path, how the worktree app folder
// dst relates to the app's own folder src.
func WorktreeLinkStates(src, dst string, paths []string) []types.WorktreeLinkState {
	out := make([]types.WorktreeLinkState, 0, len(paths))
	for _, p := range paths {
		state, msg := linkState(filepath.Join(src, filepath.FromSlash(p)), filepath.Join(dst, filepath.FromSlash(p)))
		out = append(out, types.WorktreeLinkState{Path: p, State: state, Message: msg})
	}
	return out
}

// LinkWorktree creates, in the worktree app folder dst, a link to each shared
// path of the app's own folder src that is not there yet. Existing files and
// folders are never replaced. Created links are added to the repository's
// info/exclude (root is the worktree root) so git does not list them: a
// "node_modules/" ignore rule does not match a symlink.
func LinkWorktree(src, dst, root string, paths []string) []types.WorktreeLinkState {
	out := WorktreeLinkStates(src, dst, paths)
	var linked []string
	for i := range out {
		s := &out[i]
		if s.State == LinkMissing {
			from := filepath.Join(src, filepath.FromSlash(s.Path))
			to := filepath.Join(dst, filepath.FromSlash(s.Path))
			how, err := createLink(from, to)
			if err != nil {
				s.State, s.Message = LinkFailed, err.Error()
				continue
			}
			s.State, s.Message = LinkLinked, "created ("+how+")"
		}
		if s.State == LinkLinked {
			linked = append(linked, filepath.Join(dst, filepath.FromSlash(s.Path)))
		}
	}
	if len(linked) > 0 {
		if err := excludeFromGit(root, linked); err != nil {
			for i := range out {
				if out[i].State == LinkLinked && out[i].Message == "" {
					out[i].Message = "git may list it as untracked: " + err.Error()
				}
			}
		}
	}
	return out
}

// UnlinkWorktree removes the shared-path links in the worktree app folder dst,
// leaving their targets alone. Real files and folders are kept. It runs before
// a worktree is deleted so the deletion never walks into the app's own folder.
func UnlinkWorktree(dst string, paths []string) error {
	var errs []error
	for _, p := range paths {
		to := filepath.Join(dst, filepath.FromSlash(p))
		if !isLink(to) {
			continue
		}
		if err := os.Remove(to); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func linkState(src, dst string) (string, string) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return LinkNoSource, "not found in the app folder"
	}
	if _, err := os.Lstat(dst); errors.Is(err, fs.ErrNotExist) {
		return LinkMissing, ""
	} else if err != nil {
		return LinkFailed, err.Error()
	}
	if dstInfo, err := os.Stat(dst); err == nil && os.SameFile(srcInfo, dstInfo) {
		return LinkLinked, ""
	}
	if isLink(dst) {
		target, _ := os.Readlink(dst)
		return LinkOther, "links to " + target
	}
	return LinkExists, "a real copy is already there; delete it to share the app's one"
}

// isLink reports whether p is a symlink or a Windows junction.
func isLink(p string) bool {
	_, err := os.Readlink(p)
	return err == nil
}

// createLink links to at from. It tries a symlink first; on Windows, where
// that needs Developer Mode or admin rights, a folder falls back to a junction
// and a file to a hard link. It returns which kind it made.
func createLink(from, to string) (string, error) {
	info, err := os.Stat(from)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return "", err
	}
	symErr := os.Symlink(from, to)
	if symErr == nil {
		return "symlink", nil
	}
	if runtime.GOOS != "windows" {
		return "", symErr
	}
	if info.IsDir() {
		res, err := native.Run([]string{"cmd", "/c", "mklink", "/J", to, from})
		if err == nil && res.Code == 0 {
			return "junction", nil
		}
		msg := strings.TrimSpace(res.Stdout + " " + res.Stderr)
		if err != nil {
			msg = err.Error()
		}
		return "", fmt.Errorf("symlink: %v; junction: %s", symErr, msg)
	}
	if err := os.Link(from, to); err != nil {
		return "", fmt.Errorf("symlink: %v; hard link: %v", symErr, err)
	}
	return "hard link", nil
}

// excludeFromGit adds each link (absolute paths inside the worktree at root)
// to the repository's shared info/exclude as an anchored pattern.
func excludeFromGit(root string, links []string) error {
	common, err := git(root, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return err
	}
	file := filepath.Join(filepath.FromSlash(strings.TrimSpace(common)), "info", "exclude")
	data, err := os.ReadFile(file)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	have := map[string]bool{}
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		have[strings.TrimSpace(line)] = true
	}
	var add []string
	for _, l := range links {
		rel, err := filepath.Rel(root, l)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		pattern := "/" + filepath.ToSlash(rel)
		if !have[pattern] {
			have[pattern] = true
			add = append(add, pattern)
		}
	}
	if len(add) == 0 {
		return nil
	}
	text := string(data)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if !have[excludeHeader] {
		text += excludeHeader + "\n"
	}
	text += strings.Join(add, "\n") + "\n"
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, []byte(text), 0o644)
}

// SameDir reports whether a and b are the same folder once symlinks and short
// names are resolved.
func SameDir(a, b string) bool {
	if ea, err := filepath.EvalSymlinks(a); err == nil {
		a = ea
	}
	if eb, err := filepath.EvalSymlinks(b); err == nil {
		b = eb
	}
	return samePath(a, b)
}
