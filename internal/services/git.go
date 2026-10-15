package services

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"workspace-manager/internal/native"
	"workspace-manager/internal/api/types"
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
	info := types.GitInfo{
		Worktrees:     []types.GitWorktree{},
		Branches:      []string{},
		LocalBranches: []types.GitLocalBranch{},
	}
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
	local, err := GitLocalBranches(dir, wts)
	if err != nil {
		return info, err
	}
	info.LocalBranches = local
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
		wts[i].Current = top != "" && SamePath(wts[i].Path, top)
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

// GitLocalBranches lists local branches with their upstream state and the
// worktree (from wts) that has each checked out.
func GitLocalBranches(dir string, wts []types.GitWorktree) ([]types.GitLocalBranch, error) {
	const format = "%(refname)%09%(objectname:short)%09%(upstream:short)%09%(upstream:track)%09%(contents:subject)"
	out, err := git(dir, "for-each-ref", "--format="+format, "refs/heads")
	if err != nil {
		return nil, err
	}
	branches := []types.GitLocalBranch{}
	for _, line := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		b := parseLocalBranch(line)
		if b.Name == "" {
			continue
		}
		for _, w := range wts {
			if w.Branch == b.Name {
				b.Worktree = w.Path
				break
			}
		}
		branches = append(branches, b)
	}
	return branches, nil
}

// parseLocalBranch reads one line of the for-each-ref format used by
// GitLocalBranches. The track field looks like "[ahead 1, behind 2]", "[gone]"
// or is empty.
func parseLocalBranch(line string) types.GitLocalBranch {
	f := strings.SplitN(line, "\t", 5)
	for len(f) < 5 {
		f = append(f, "")
	}
	name, ok := strings.CutPrefix(f[0], "refs/heads/")
	if !ok {
		return types.GitLocalBranch{}
	}
	b := types.GitLocalBranch{Name: name, Head: f[1], Upstream: f[2], Subject: f[4]}
	track := strings.Trim(strings.TrimSpace(f[3]), "[]")
	for _, part := range strings.Split(track, ",") {
		part = strings.TrimSpace(part)
		switch {
		case part == "gone":
			b.Gone = true
		case strings.HasPrefix(part, "ahead "):
			b.Ahead, _ = strconv.Atoi(strings.TrimPrefix(part, "ahead "))
		case strings.HasPrefix(part, "behind "):
			b.Behind, _ = strconv.Atoi(strings.TrimPrefix(part, "behind "))
		}
	}
	return b
}

// validBranchName trims name and checks it with git's own rules.
func validBranchName(dir, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("branch name is required")
	}
	if strings.HasPrefix(name, "-") {
		return "", fmt.Errorf("branch names cannot start with '-'")
	}
	if _, err := git(dir, "check-ref-format", "--branch", name); err != nil {
		return "", fmt.Errorf("%q is not a valid branch name", name)
	}
	return name, nil
}

// GitBranchCreate creates local branch name at base (default HEAD) without
// checking it out.
func GitBranchCreate(dir, name, base string) error {
	name, err := validBranchName(dir, name)
	if err != nil {
		return err
	}
	base = strings.TrimSpace(base)
	if strings.HasPrefix(base, "-") {
		return fmt.Errorf("base cannot start with '-'")
	}
	args := []string{"branch", "--", name}
	if base != "" {
		args = append(args, base)
	}
	_, err = git(dir, args...)
	return err
}

// GitBranchRename renames local branch oldName to newName. Git moves the HEAD
// of any worktree that has it checked out.
func GitBranchRename(dir, oldName, newName string) error {
	oldName = strings.TrimSpace(oldName)
	if oldName == "" || strings.HasPrefix(oldName, "-") {
		return fmt.Errorf("invalid branch %q", oldName)
	}
	newName, err := validBranchName(dir, newName)
	if err != nil {
		return err
	}
	if newName == oldName {
		return fmt.Errorf("the new name is the same as the current one")
	}
	_, err = git(dir, "branch", "-m", "--", oldName, newName)
	return err
}

// GitBranchCheckout switches the working tree of dir to local branch name. Git
// refuses when uncommitted changes would be overwritten or when another
// worktree has the branch checked out.
func GitBranchCheckout(dir, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, "-") {
		return fmt.Errorf("invalid branch %q", name)
	}
	// Only local branches: switch would otherwise create one from a remote.
	if _, err := git(dir, "show-ref", "--verify", "--quiet", "refs/heads/"+name); err != nil {
		return fmt.Errorf("no local branch %q", name)
	}
	_, err := git(dir, "switch", name)
	return err
}

// GitBranchPull pulls the upstream of local branch name into the worktree that
// has it checked out. When git stops on merge conflicts it is not an error: the
// result lists the conflicted files so the caller can open them.
func GitBranchPull(dir, name string) (types.GitPullResult, error) {
	res := types.GitPullResult{Conflicts: []string{}}
	wts, err := GitWorktrees(dir)
	if err != nil {
		return res, err
	}
	branches, err := GitLocalBranches(dir, wts)
	if err != nil {
		return res, err
	}
	var b *types.GitLocalBranch
	for i := range branches {
		if branches[i].Name == strings.TrimSpace(name) {
			b = &branches[i]
		}
	}
	switch {
	case b == nil:
		return res, fmt.Errorf("no local branch %q", name)
	case b.Worktree == "":
		return res, fmt.Errorf("check out %s first: a pull needs a worktree", b.Name)
	case b.Upstream == "" || b.Gone:
		return res, fmt.Errorf("%s has no upstream branch to pull from", b.Name)
	}
	res.Path = b.Worktree
	args := []string{"pull", "--no-edit"}
	// Without a configured strategy git refuses a diverged pull.
	if cfg, _ := git(res.Path, "config", "pull.rebase"); cfg == "" {
		args = append(args, "--no-rebase")
	}
	out, err := git(res.Path, args...)
	if err == nil {
		if out == "" {
			out = "Already up to date."
		}
		res.Output = out
		return res, nil
	}
	conflicts, _ := git(res.Path, "diff", "--name-only", "--diff-filter=U")
	for _, f := range strings.Split(strings.ReplaceAll(conflicts, "\r\n", "\n"), "\n") {
		if f = strings.TrimSpace(f); f != "" {
			res.Conflicts = append(res.Conflicts, filepath.Join(res.Path, filepath.FromSlash(f)))
		}
	}
	if len(res.Conflicts) == 0 {
		return res, err
	}
	res.Output = err.Error()
	return res, nil
}

// FindWorktreeFile returns file (absolute, or relative to dir) cleaned, when it
// lies inside one of the repo's worktrees.
func FindWorktreeFile(dir string, file string) (string, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return "", fmt.Errorf("file is required")
	}
	if !filepath.IsAbs(file) {
		file = filepath.Join(dir, file)
	}
	file = filepath.Clean(file)
	wts, err := GitWorktrees(dir)
	if err != nil {
		return "", err
	}
	for _, w := range wts {
		rel, err := filepath.Rel(filepath.Clean(w.Path), file)
		if err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return file, nil
		}
	}
	return "", fmt.Errorf("%s is not inside a worktree of this repository", file)
}

// GitBranchPush pushes local branch name to its upstream, or when it has none
// publishes it to origin (else the first remote) and sets that as its upstream.
// force uses --force-with-lease. It needs no checkout: the refspec is explicit.
func GitBranchPush(dir, name string, force bool) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, "-") {
		return "", fmt.Errorf("invalid branch %q", name)
	}
	if _, err := git(dir, "show-ref", "--verify", "--quiet", "refs/heads/"+name); err != nil {
		return "", fmt.Errorf("no local branch %q", name)
	}
	up, err := git(dir, "for-each-ref", "--format=%(upstream:remotename)%09%(upstream:remoteref)", "refs/heads/"+name)
	if err != nil {
		return "", err
	}
	remote, remoteRef, _ := strings.Cut(strings.TrimSpace(up), "\t")
	remote, remoteRef = strings.TrimSpace(remote), strings.TrimSpace(remoteRef)
	setUpstream := false
	if remote == "" || remoteRef == "" {
		setUpstream = true
		remoteRef = "refs/heads/" + name
		remotes, err := git(dir, "remote")
		if err != nil {
			return "", err
		}
		for _, r := range strings.Fields(remotes) {
			if remote == "" || r == "origin" {
				remote = r
			}
		}
		if remote == "" {
			return "", fmt.Errorf("this repository has no remote to push to")
		}
	}
	args := []string{"push"}
	if setUpstream {
		args = append(args, "--set-upstream")
	}
	if force {
		args = append(args, "--force-with-lease")
	}
	args = append(args, remote, "refs/heads/"+name+":"+remoteRef)
	out, err := git(dir, args...)
	if err != nil {
		return "", err
	}
	if out == "" {
		out = "Everything up-to-date."
	}
	return out, nil
}

// GitBranchDelete deletes local branch name. Without force git refuses a
// branch that is not fully merged; it always refuses one checked out in a
// worktree.
func GitBranchDelete(dir, name string, force bool) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, "-") {
		return fmt.Errorf("invalid branch %q", name)
	}
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := git(dir, "branch", flag, "--", name)
	return err
}

var branchSlugRe =regexp.MustCompile(`[^A-Za-z0-9._-]+`)

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
		if SamePath(w.Path, in.Path) {
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
		if !SamePath(w.Path, path) {
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
		if SamePath(wts[i].Path, path) {
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

// SamePath reports whether two paths point at the same location, and is
// case-insensitive on Windows.
func SamePath(a, b string) bool {
	a, b = filepath.Clean(filepath.FromSlash(a)), filepath.Clean(filepath.FromSlash(b))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
