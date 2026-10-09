package services

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"workspace-manager/native"
	"workspace-manager/types"
)

// git runs git in dir and returns its combined output. A non-zero exit becomes
// an error carrying git's own message.
func git(dir string, args ...string) (string, error) {
	res, err := native.Run(append([]string{"git", "-C", dir}, args...))
	if err != nil {
		return "", fmt.Errorf("git: %w", err)
	}
	out := strings.TrimSpace(strings.TrimSpace(res.Stdout) + "\n" + strings.TrimSpace(res.Stderr))
	if res.Code != 0 {
		if out == "" {
			out = fmt.Sprintf("git %s exited with code %d", args[0], res.Code)
		}
		return "", fmt.Errorf("%s", out)
	}
	return out, nil
}

// GitInfo reports whether dir is inside a git repository and, if so, its
// worktrees and branches. It never fails for a folder that is not a repo.
func GitInfo(dir string) (types.GitInfo, error) {
	info := types.GitInfo{Worktrees: []types.GitWorktree{}, Branches: []string{}}
	if _, err := git(dir, "rev-parse", "--git-dir"); err != nil {
		return info, nil
	}
	info.IsRepo = true
	wts, err := GitWorktrees(dir)
	if err != nil {
		return info, err
	}
	for i := range wts {
		if !wts[i].Main && !wts[i].Bare {
			wts[i].SharedLinks, _ = WorktreeLinksOverride(wts[i].Path)
		}
	}
	info.Worktrees = wts
	branches, err := GitBranches(dir)
	if err != nil {
		return info, err
	}
	info.Branches = branches
	return info, nil
}

// GitFetchAll fetches every remote and prunes deleted remote branches.
func GitFetchAll(dir string) (string, error) {
	out, err := git(dir, "fetch", "--all", "--prune")
	if err != nil {
		return "", err
	}
	if out == "" {
		out = "Already up to date."
	}
	return out, nil
}

// GitWorktrees lists the worktrees of the repo containing dir. The first one is
// the main worktree; Current marks the one dir is in.
func GitWorktrees(dir string) ([]types.GitWorktree, error) {
	out, err := git(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	top, _ := git(dir, "rev-parse", "--show-toplevel")
	wts := parseWorktrees(out)
	for i := range wts {
		wts[i].Main = i == 0
		wts[i].Current = top != "" && samePath(wts[i].Path, top)
	}
	return wts, nil
}

func parseWorktrees(out string) []types.GitWorktree {
	wts := []types.GitWorktree{}
	var cur *types.GitWorktree
	for _, line := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		key, val, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			wts = append(wts, types.GitWorktree{Path: filepath.Clean(val)})
			cur = &wts[len(wts)-1]
		case "HEAD":
			if cur != nil {
				cur.Head = val
			}
		case "branch":
			if cur != nil {
				cur.Branch = strings.TrimPrefix(val, "refs/heads/")
			}
		case "detached":
			if cur != nil {
				cur.Detached = true
			}
		case "bare":
			if cur != nil {
				cur.Bare = true
			}
		case "locked":
			if cur != nil {
				cur.Locked = true
			}
		case "prunable":
			if cur != nil {
				cur.Prunable = true
			}
		}
	}
	return wts
}

// GitBranches lists local branches, then remote-tracking ones (without the
// remote HEAD symref).
func GitBranches(dir string) ([]string, error) {
	// Full ref names: the short form of refs/remotes/origin/HEAD is just "origin".
	out, err := git(dir, "for-each-ref", "--format=%(refname)", "refs/heads", "refs/remotes")
	if err != nil {
		return nil, err
	}
	branches := []string{}
	for _, ref := range strings.Split(out, "\n") {
		ref = strings.TrimSpace(ref)
		if ref == "" || strings.HasSuffix(ref, "/HEAD") {
			continue
		}
		if b, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
			branches = append(branches, b)
		} else if b, ok := strings.CutPrefix(ref, "refs/remotes/"); ok {
			branches = append(branches, b)
		}
	}
	return branches, nil
}

var branchSlugRe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// DefaultWorktreePath is where a worktree for branch goes when no path is
// given: a sibling of the main worktree named "<repo>-<branch>".
func DefaultWorktreePath(mainPath, branch string) string {
	slug := strings.Trim(branchSlugRe.ReplaceAllString(branch, "-"), "-")
	return filepath.Join(filepath.Dir(mainPath), filepath.Base(mainPath)+"-"+slug)
}

// ResolveWorktreeAdd validates in and fills in an absolute path. Relative paths
// are taken from dir.
func ResolveWorktreeAdd(dir string, wts []types.GitWorktree, in types.GitWorktreeAddInput) (types.GitWorktreeAddInput, error) {
	in.Branch = strings.TrimSpace(in.Branch)
	in.Base = strings.TrimSpace(in.Base)
	in.Path = strings.TrimSpace(in.Path)
	if in.Branch == "" {
		return in, fmt.Errorf("branch is required")
	}
	if strings.HasPrefix(in.Branch, "-") || strings.HasPrefix(in.Base, "-") {
		return in, fmt.Errorf("branch names cannot start with '-'")
	}
	if !in.NewBranch {
		in.Base = ""
	}
	if in.Links != nil {
		links, err := NormalizeWorktreeLinks(in.Links)
		if err != nil {
			return in, err
		}
		in.Links = links
	}
	if in.Path == "" {
		if len(wts) == 0 {
			return in, fmt.Errorf("path is required")
		}
		in.Path = DefaultWorktreePath(wts[0].Path, in.Branch)
	} else if !filepath.IsAbs(in.Path) {
		in.Path = filepath.Join(dir, in.Path)
	}
	in.Path = filepath.Clean(in.Path)
	for _, w := range wts {
		if samePath(w.Path, in.Path) {
			return in, fmt.Errorf("a worktree already exists at %s", in.Path)
		}
		if !in.NewBranch && w.Branch != "" && w.Branch == in.Branch {
			return in, fmt.Errorf("branch %s is already checked out at %s", in.Branch, w.Path)
		}
	}
	return in, nil
}

// GitWorktreeAdd creates a worktree. With NewBranch it creates Branch from Base
// (or HEAD); otherwise it checks out the existing Branch. Returns the worktree path.
func GitWorktreeAdd(dir string, in types.GitWorktreeAddInput) (string, error) {
	wts, err := GitWorktrees(dir)
	if err != nil {
		return "", err
	}
	in, err = ResolveWorktreeAdd(dir, wts, in)
	if err != nil {
		return "", err
	}
	args := []string{"worktree", "add"}
	if in.NewBranch {
		args = append(args, "-b", in.Branch, "--", in.Path)
		if in.Base != "" {
			args = append(args, in.Base)
		}
	} else {
		args = append(args, "--", in.Path, in.Branch)
	}
	if _, err := git(dir, args...); err != nil {
		return "", err
	}
	if len(in.Links) > 0 {
		if err := SaveWorktreeLinksOverride(in.Path, in.Links); err != nil {
			return in.Path, fmt.Errorf("worktree created at %s, but its shared paths were not saved: %w", in.Path, err)
		}
	}
	return in.Path, nil
}

// FindRemovableWorktree returns the worktree at path, refusing the main one.
func FindRemovableWorktree(dir string, wts []types.GitWorktree, path string) (types.GitWorktree, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return types.GitWorktree{}, fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	for _, w := range wts {
		if !samePath(w.Path, path) {
			continue
		}
		if w.Main {
			return w, fmt.Errorf("the main worktree cannot be removed")
		}
		if w.Current {
			return w, fmt.Errorf("this app's own worktree cannot be removed")
		}
		return w, nil
	}
	return types.GitWorktree{}, fmt.Errorf("no worktree at %s", path)
}

// GitWorktreeRemove deletes a linked worktree. force also removes one with
// uncommitted changes. The branch is kept.
func GitWorktreeRemove(dir, path string, force bool) error {
	wts, err := GitWorktrees(dir)
	if err != nil {
		return err
	}
	w, err := FindRemovableWorktree(dir, wts, path)
	if err != nil {
		return err
	}
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	_, err = git(dir, append(args, "--", w.Path)...)
	return err
}

// GitWorktreePrune drops git's records of worktrees whose folder is gone.
func GitWorktreePrune(dir string) (string, error) {
	out, err := git(dir, "worktree", "prune", "--verbose")
	if err != nil {
		return "", err
	}
	if out == "" {
		out = "Nothing to prune."
	}
	return out, nil
}

// WorktreeAppDir maps the app's project path dir into the worktree at path: for
// an app in a subfolder of the repo it is the same subfolder of the worktree.
// It returns that folder and the worktree's root.
func WorktreeAppDir(dir, path string) (appDir, root string, err error) {
	wts, err := GitWorktrees(dir)
	if err != nil {
		return "", "", err
	}
	var w *types.GitWorktree
	for i := range wts {
		if samePath(wts[i].Path, path) {
			w = &wts[i]
			break
		}
	}
	if w == nil {
		return "", "", fmt.Errorf("no worktree at %s", path)
	}
	if w.Bare || w.Prunable {
		return "", "", fmt.Errorf("worktree %s has no checked-out folder", w.Path)
	}
	top, err := git(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", err
	}
	// --show-toplevel resolves symlinks/short names; resolve dir the same way.
	absDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		absDir = dir
	}
	rel, err := filepath.Rel(filepath.Clean(filepath.FromSlash(top)), absDir)
	if err != nil || strings.HasPrefix(rel, "..") {
		rel = "."
	}
	appDir = filepath.Join(w.Path, rel)
	if fi, err := os.Stat(appDir); err != nil || !fi.IsDir() {
		return "", "", fmt.Errorf("folder %s does not exist in the worktree", appDir)
	}
	return appDir, w.Path, nil
}

func samePath(a, b string) bool {
	a, b = filepath.Clean(filepath.FromSlash(a)), filepath.Clean(filepath.FromSlash(b))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
