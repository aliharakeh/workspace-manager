package git

import (
	"fmt"
	"strings"

	"workspace-manager/shared/api"
	"workspace-manager/shared/api/lib"
	"workspace-manager/shared/api/types"
	"workspace-manager/shared/native"
)

// Backend is the daemon side of the git operations.
type Backend interface {
	// App reads one app.
	App(appID int64) (types.App, error)
	// Running reports whether the app has a runner session.
	Running(appID int64) bool
	// Links returns the app's shared worktree paths.
	Links(appID int64) ([]string, error)
	// StartIn starts a run in the app's folder inside a worktree.
	StartIn(appID int64, kind, appDir, worktree string, configSetID int64) (types.StatusEvent, error)
	// Changed tells the UI that an entity was written, so it reloads it.
	Changed(entity string, ids ...int64)
}

// Git is the set of git operations the app's Git panel uses. They run in the
// GUI process, never in the daemon; what they need from the daemon (the app's
// folder, whether it is running, the shared paths, starting a run in a
// worktree) comes through a Backend.
type Git struct {
	B Backend
}

// New returns the operations bound to a backend.
func New(b Backend) *Git { return &Git{B: b} }

// appProjectDir returns the validated project path of an app.
func (g *Git) appProjectDir(appID int64) (string, error) {
	app, err := g.B.App(appID)
	if err != nil {
		return "", err
	}
	ok, resolved, errMsg := lib.ValidateProjectPath(app.ProjectPath)
	if !ok {
		return "", fmt.Errorf("%s", errMsg)
	}
	return resolved, nil
}

func checkConfigKind(kind string) error {
	switch kind {
	case types.KindDev, types.KindProd, types.KindBuild, types.KindSetup:
		return nil
	}
	return fmt.Errorf("kind must be dev, prod, build or setup")
}

func (g *Git) Info(appID int64) (types.GitInfo, error) {
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return types.GitInfo{}, err
	}
	return GitInfo(dir)
}

func (g *Git) FetchAll(appID int64) (types.GitOutput, error) {
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return types.GitOutput{}, err
	}
	out, err := GitFetchAll(dir)
	if err != nil {
		return types.GitOutput{}, err
	}
	return types.GitOutput{Output: out}, nil
}

// BranchCreate creates a local branch at base (default HEAD), not checked out.
func (g *Git) BranchCreate(appID int64, name string, base string) (types.Ok, error) {
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return types.Ok{}, err
	}
	if err := GitBranchCreate(dir, name, base); err != nil {
		return types.Ok{}, err
	}
	g.B.Changed("git", appID)
	return types.Ok{Ok: true}, nil
}

func (g *Git) BranchRename(appID int64, oldName string, newName string) (types.Ok, error) {
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return types.Ok{}, err
	}
	if err := GitBranchRename(dir, oldName, newName); err != nil {
		return types.Ok{}, err
	}
	g.B.Changed("git", appID)
	return types.Ok{Ok: true}, nil
}

// BranchCheckout switches the app's own folder to a local branch. It is
// refused while the app has a runner session, which works in that folder.
func (g *Git) BranchCheckout(appID int64, name string) (types.Ok, error) {
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return types.Ok{}, err
	}
	if g.B.Running(appID) {
		return types.Ok{}, api.Errf(api.KindConflict, "stop the running app before switching branches")
	}
	if err := GitBranchCheckout(dir, name); err != nil {
		return types.Ok{}, err
	}
	g.B.Changed("git", appID)
	return types.Ok{Ok: true}, nil
}

// BranchPull pulls a local branch's upstream in the worktree that has it
// checked out. Merge conflicts come back in the result, not as an error. It is
// refused while the app has a runner session.
func (g *Git) BranchPull(appID int64, name string) (types.GitPullResult, error) {
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return types.GitPullResult{}, err
	}
	if g.B.Running(appID) {
		return types.GitPullResult{}, api.Errf(api.KindConflict, "stop the running app before pulling")
	}
	res, err := GitBranchPull(dir, name)
	if err == nil {
		g.B.Changed("git", appID)
	}
	return res, err
}

// BranchPush pushes a local branch to its upstream, or publishes it when it
// has none; force uses --force-with-lease.
func (g *Git) BranchPush(appID int64, name string, force bool) (types.GitOutput, error) {
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return types.GitOutput{}, err
	}
	out, err := GitBranchPush(dir, name, force)
	if err != nil {
		return types.GitOutput{}, err
	}
	g.B.Changed("git", appID)
	return types.GitOutput{Output: out}, nil
}

// FileOpenInEditor opens a file of one of the repo's worktrees (e.g. a
// conflicted one) in an editor (an id from AppsEditors).
func (g *Git) FileOpenInEditor(appID int64, file string, editor string) (types.Ok, error) {
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return types.Ok{}, err
	}
	file, err = FindWorktreeFile(dir, file)
	if err != nil {
		return types.Ok{}, err
	}
	if err := native.OpenInEditor(editor, file); err != nil {
		return types.Ok{}, err
	}
	return types.Ok{Ok: true}, nil
}

// BranchDelete deletes a local branch; force also deletes an unmerged one.
func (g *Git) BranchDelete(appID int64, name string, force bool) (types.Ok, error) {
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return types.Ok{}, err
	}
	if err := GitBranchDelete(dir, name, force); err != nil {
		return types.Ok{}, err
	}
	g.B.Changed("git", appID)
	return types.Ok{Ok: true}, nil
}

func (g *Git) WorktreeAdd(appID int64, body types.GitWorktreeAddInput) (types.GitWorktreeAddResult, error) {
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return types.GitWorktreeAddResult{}, err
	}
	path, err := GitWorktreeAdd(dir, body)
	if err != nil {
		return types.GitWorktreeAddResult{}, err
	}
	res := types.GitWorktreeAddResult{Path: path, Links: []types.WorktreeLinkState{}}
	if !body.LinkShared {
		g.B.Changed("git", appID)
		return res, nil
	}
	// The worktree exists now, so a link that fails is reported in Links
	// instead of failing the add.
	appDir, root, err := WorktreeAppDir(dir, path)
	if err == nil {
		res.Links, err = g.linkWorktree(appID, dir, appDir, root)
	}
	if err != nil {
		g.B.Changed("git", appID)
		return res, fmt.Errorf("worktree created at %s, but shared paths were not linked: %w", path, err)
	}
	g.B.Changed("git", appID)
	return res, nil
}

func (g *Git) WorktreeRemove(appID int64, path string, force bool) (types.Ok, error) {
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return types.Ok{}, err
	}
	// Drop the shared links first so deleting the worktree cannot reach into
	// the app's own copies through them.
	if appDir, root, err := WorktreeAppDir(dir, path); err == nil {
		links, err := g.B.Links(appID)
		if err != nil {
			return types.Ok{}, err
		}
		// The worktree's own list too: it may link paths the app's does not.
		if own, ok := WorktreeLinksOverride(root); ok {
			links = append(links, own...)
		}
		if err := UnlinkWorktree(appDir, links); err != nil {
			return types.Ok{}, fmt.Errorf("removing shared links: %w", err)
		}
	}
	if err := GitWorktreeRemove(dir, path, force); err != nil {
		return types.Ok{}, err
	}
	g.B.Changed("git", appID)
	return types.Ok{Ok: true}, nil
}

func (g *Git) WorktreePrune(appID int64) (types.GitOutput, error) {
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return types.GitOutput{}, err
	}
	out, err := GitWorktreePrune(dir)
	if err != nil {
		return types.GitOutput{}, err
	}
	g.B.Changed("git", appID)
	return types.GitOutput{Output: out}, nil
}

// WorktreeStart runs the app's run, build or setup config (by kind) inside
// the worktree at path, with config set configSetID (0: the app's active set).
// The daemon starts it: it is the app's one runner session, like RunnerRun.
func (g *Git) WorktreeStart(appID int64, path string, kind string, configSetID int64) (types.StatusEvent, error) {
	if err := checkConfigKind(kind); err != nil {
		return types.StatusEvent{}, err
	}
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return types.StatusEvent{}, err
	}
	appDir, root, err := WorktreeAppDir(dir, path)
	if err != nil {
		return types.StatusEvent{}, err
	}
	if kind == types.KindSetup && !SameDir(appDir, dir) {
		// Setup links the shared paths first, so e.g. an install reuses them.
		states, err := g.linkWorktree(appID, dir, appDir, root)
		if err != nil {
			return types.StatusEvent{}, err
		}
		var failed []string
		for _, s := range states {
			if s.State == LinkFailed {
				failed = append(failed, s.Path+": "+s.Message)
			}
		}
		if len(failed) > 0 {
			return types.StatusEvent{}, fmt.Errorf("could not link shared paths, setup not started:\n%s", strings.Join(failed, "\n"))
		}
	}
	return g.B.StartIn(appID, kind, appDir, root, configSetID)
}

func (g *Git) linkWorktree(appID int64, dir, appDir, root string) ([]types.WorktreeLinkState, error) {
	links, err := g.worktreeLinkPaths(appID, root)
	if err != nil {
		return nil, err
	}
	return LinkWorktree(dir, appDir, root, links), nil
}

// worktreeLinkPaths is the shared paths of the worktree at root: its own list
// if it was given one when added, otherwise the app's.
func (g *Git) worktreeLinkPaths(appID int64, root string) ([]string, error) {
	if own, ok := WorktreeLinksOverride(root); ok {
		return own, nil
	}
	return g.B.Links(appID)
}

// WorktreeLinks reports each shared path in the worktree at path; with
// create it first links the ones that are missing there. Existing files and
// folders are never replaced.
func (g *Git) WorktreeLinks(appID int64, path string, create bool) ([]types.WorktreeLinkState, error) {
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return nil, err
	}
	appDir, root, err := WorktreeAppDir(dir, path)
	if err != nil {
		return nil, err
	}
	if SameDir(appDir, dir) {
		return nil, fmt.Errorf("this is the app's own folder: its worktrees link to it")
	}
	if create {
		states, err := g.linkWorktree(appID, dir, appDir, root)
		if err == nil {
			g.B.Changed("git", appID)
		}
		return states, err
	}
	links, err := g.worktreeLinkPaths(appID, root)
	if err != nil {
		return nil, err
	}
	return WorktreeLinkStates(dir, appDir, links), nil
}

// worktreeAppDir is the app's folder inside the worktree at path.
func (g *Git) worktreeAppDir(appID int64, path string) (string, error) {
	dir, err := g.appProjectDir(appID)
	if err != nil {
		return "", err
	}
	appDir, _, err := WorktreeAppDir(dir, path)
	return appDir, err
}

// WorktreeOpenFolder opens the app's folder inside the worktree at path.
func (g *Git) WorktreeOpenFolder(appID int64, path string) (types.Ok, error) {
	appDir, err := g.worktreeAppDir(appID, path)
	if err != nil {
		return types.Ok{}, err
	}
	if err := native.OpenFolder(appDir); err != nil {
		return types.Ok{}, err
	}
	return types.Ok{Ok: true}, nil
}

// WorktreeOpenInEditor opens the app's folder inside the worktree at path in
// an editor (an id from AppsEditors).
func (g *Git) WorktreeOpenInEditor(appID int64, path string, editor string) (types.Ok, error) {
	appDir, err := g.worktreeAppDir(appID, path)
	if err != nil {
		return types.Ok{}, err
	}
	if err := native.OpenInEditor(editor, appDir); err != nil {
		return types.Ok{}, err
	}
	return types.Ok{Ok: true}, nil
}
