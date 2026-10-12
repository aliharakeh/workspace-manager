package ops

import (
	"fmt"
	"strings"

	"workspace-manager/api"
	"workspace-manager/api/types"
	"workspace-manager/server/native"
	"workspace-manager/server/services"
)

func (o *Ops) GitInfo(appID int64) (types.GitInfo, error) {
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return types.GitInfo{}, err
	}
	return services.GitInfo(dir)
}

func (o *Ops) GitFetchAll(appID int64) (types.GitOutput, error) {
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return types.GitOutput{}, err
	}
	out, err := services.GitFetchAll(dir)
	if err != nil {
		return types.GitOutput{}, err
	}
	return types.GitOutput{Output: out}, nil
}

// GitBranchCreate creates a local branch at base (default HEAD), not checked out.
func (o *Ops) GitBranchCreate(appID int64, name string, base string) (types.Ok, error) {
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return types.Ok{}, err
	}
	if err := services.GitBranchCreate(dir, name, base); err != nil {
		return types.Ok{}, err
	}
	o.changed("git", appID)
	return types.Ok{Ok: true}, nil
}

func (o *Ops) GitBranchRename(appID int64, oldName string, newName string) (types.Ok, error) {
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return types.Ok{}, err
	}
	if err := services.GitBranchRename(dir, oldName, newName); err != nil {
		return types.Ok{}, err
	}
	o.changed("git", appID)
	return types.Ok{Ok: true}, nil
}

// GitBranchCheckout switches the app's own folder to a local branch. It is
// refused while the app has a runner session, which works in that folder.
func (o *Ops) GitBranchCheckout(appID int64, name string) (types.Ok, error) {
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return types.Ok{}, err
	}
	if o.Runner.GetStatus(appID).Running {
		return types.Ok{}, api.Errf(api.KindConflict, "stop the running app before switching branches")
	}
	if err := services.GitBranchCheckout(dir, name); err != nil {
		return types.Ok{}, err
	}
	o.changed("git", appID)
	return types.Ok{Ok: true}, nil
}

// GitBranchPull pulls a local branch's upstream in the worktree that has it
// checked out. Merge conflicts come back in the result, not as an error. It is
// refused while the app has a runner session.
func (o *Ops) GitBranchPull(appID int64, name string) (types.GitPullResult, error) {
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return types.GitPullResult{}, err
	}
	if o.Runner.GetStatus(appID).Running {
		return types.GitPullResult{}, api.Errf(api.KindConflict, "stop the running app before pulling")
	}
	res, err := services.GitBranchPull(dir, name)
	if err == nil {
		o.changed("git", appID)
	}
	return res, err
}

// GitBranchPush pushes a local branch to its upstream, or publishes it when it
// has none; force uses --force-with-lease.
func (o *Ops) GitBranchPush(appID int64, name string, force bool) (types.GitOutput, error) {
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return types.GitOutput{}, err
	}
	out, err := services.GitBranchPush(dir, name, force)
	if err != nil {
		return types.GitOutput{}, err
	}
	o.changed("git", appID)
	return types.GitOutput{Output: out}, nil
}

// GitFileOpenInEditor opens a file of one of the repo's worktrees (e.g. a
// conflicted one) in an editor (an id from AppsEditors).
func (o *Ops) GitFileOpenInEditor(appID int64, file string, editor string) (types.Ok, error) {
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return types.Ok{}, err
	}
	file, err = services.FindWorktreeFile(dir, file)
	if err != nil {
		return types.Ok{}, err
	}
	if err := native.OpenInEditor(editor, file); err != nil {
		return types.Ok{}, err
	}
	return types.Ok{Ok: true}, nil
}

// GitBranchDelete deletes a local branch; force also deletes an unmerged one.
func (o *Ops) GitBranchDelete(appID int64, name string, force bool) (types.Ok, error) {
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return types.Ok{}, err
	}
	if err := services.GitBranchDelete(dir, name, force); err != nil {
		return types.Ok{}, err
	}
	o.changed("git", appID)
	return types.Ok{Ok: true}, nil
}

func (o *Ops) GitWorktreeAdd(appID int64, body types.GitWorktreeAddInput) (types.GitWorktreeAddResult, error) {
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return types.GitWorktreeAddResult{}, err
	}
	path, err := services.GitWorktreeAdd(dir, body)
	if err != nil {
		return types.GitWorktreeAddResult{}, err
	}
	res := types.GitWorktreeAddResult{Path: path, Links: []types.WorktreeLinkState{}}
	if !body.LinkShared {
		o.changed("git", appID)
		return res, nil
	}
	// The worktree exists now, so a link that fails is reported in Links
	// instead of failing the add.
	appDir, root, err := services.WorktreeAppDir(dir, path)
	if err == nil {
		res.Links, err = o.linkWorktree(appID, dir, appDir, root)
	}
	if err != nil {
		o.changed("git", appID)
		return res, fmt.Errorf("worktree created at %s, but shared paths were not linked: %w", path, err)
	}
	o.changed("git", appID)
	return res, nil
}

func (o *Ops) GitWorktreeRemove(appID int64, path string, force bool) (types.Ok, error) {
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return types.Ok{}, err
	}
	// Drop the shared links first so deleting the worktree cannot reach into
	// the app's own copies through them.
	if appDir, root, err := services.WorktreeAppDir(dir, path); err == nil {
		links, err := o.DB.ListWorktreeLinksT(o.Ctx(), appID)
		if err != nil {
			return types.Ok{}, err
		}
		// The worktree's own list too: it may link paths the app's does not.
		if own, ok := services.WorktreeLinksOverride(root); ok {
			links = append(links, own...)
		}
		if err := services.UnlinkWorktree(appDir, links); err != nil {
			return types.Ok{}, fmt.Errorf("removing shared links: %w", err)
		}
	}
	if err := services.GitWorktreeRemove(dir, path, force); err != nil {
		return types.Ok{}, err
	}
	o.changed("git", appID)
	return types.Ok{Ok: true}, nil
}

func (o *Ops) GitWorktreePrune(appID int64) (types.GitOutput, error) {
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return types.GitOutput{}, err
	}
	out, err := services.GitWorktreePrune(dir)
	if err != nil {
		return types.GitOutput{}, err
	}
	o.changed("git", appID)
	return types.GitOutput{Output: out}, nil
}

// GitWorktreeStart runs the app's run, build or setup config (by kind) inside
// the worktree at path, with config set configSetID (0: the app's active set).
// It is the app's one runner session, like RunnerRun.
func (o *Ops) GitWorktreeStart(appID int64, path string, kind string, configSetID int64) (types.StatusEvent, error) {
	if kind != types.KindRun && kind != types.KindBuild && kind != types.KindSetup {
		return types.StatusEvent{}, fmt.Errorf("unknown kind %q", kind)
	}
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return types.StatusEvent{}, err
	}
	appDir, root, err := services.WorktreeAppDir(dir, path)
	if err != nil {
		return types.StatusEvent{}, err
	}
	if kind == types.KindSetup && !services.SameDir(appDir, dir) {
		// Setup links the shared paths first, so e.g. an install reuses them.
		states, err := o.linkWorktree(appID, dir, appDir, root)
		if err != nil {
			return types.StatusEvent{}, err
		}
		var failed []string
		for _, s := range states {
			if s.State == services.LinkFailed {
				failed = append(failed, s.Path+": "+s.Message)
			}
		}
		if len(failed) > 0 {
			return types.StatusEvent{}, fmt.Errorf("could not link shared paths, setup not started:\n%s", strings.Join(failed, "\n"))
		}
	}
	return o.Runner.StartIn(o.Ctx(), appID, kind, appDir, root, configSetID)
}

func (o *Ops) linkWorktree(appID int64, dir, appDir, root string) ([]types.WorktreeLinkState, error) {
	links, err := o.worktreeLinkPaths(appID, root)
	if err != nil {
		return nil, err
	}
	return services.LinkWorktree(dir, appDir, root, links), nil
}

// worktreeLinkPaths is the shared paths of the worktree at root: its own list
// if it was given one when added, otherwise the app's.
func (o *Ops) worktreeLinkPaths(appID int64, root string) ([]string, error) {
	if own, ok := services.WorktreeLinksOverride(root); ok {
		return own, nil
	}
	return o.DB.ListWorktreeLinksT(o.Ctx(), appID)
}

// WorktreeLinksGet returns the app's shared worktree paths: files and folders,
// relative to the app folder, that its worktrees link to instead of copying.
func (o *Ops) WorktreeLinksGet(appID int64) ([]string, error) {
	return o.DB.ListWorktreeLinksT(o.Ctx(), appID)
}

// WorktreeLinksSave replaces the app's shared worktree paths and returns them
// cleaned up.
func (o *Ops) WorktreeLinksSave(appID int64, paths []string) ([]string, error) {
	clean, err := services.NormalizeWorktreeLinks(paths)
	if err != nil {
		return nil, err
	}
	if err := o.DB.SetWorktreeLinksT(o.Ctx(), appID, clean); err != nil {
		return nil, err
	}
	o.changed("worktreeLinks", appID)
	return clean, nil
}

// GitWorktreeLinks reports each shared path in the worktree at path; with
// create it first links the ones that are missing there. Existing files and
// folders are never replaced.
func (o *Ops) GitWorktreeLinks(appID int64, path string, create bool) ([]types.WorktreeLinkState, error) {
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return nil, err
	}
	appDir, root, err := services.WorktreeAppDir(dir, path)
	if err != nil {
		return nil, err
	}
	if services.SameDir(appDir, dir) {
		return nil, fmt.Errorf("this is the app's own folder: its worktrees link to it")
	}
	if create {
		states, err := o.linkWorktree(appID, dir, appDir, root)
		if err == nil {
			o.changed("git", appID)
		}
		return states, err
	}
	links, err := o.worktreeLinkPaths(appID, root)
	if err != nil {
		return nil, err
	}
	return services.WorktreeLinkStates(dir, appDir, links), nil
}

// worktreeAppDir is the app's folder inside the worktree at path, as used by
// GitWorktreeStart.
func (o *Ops) worktreeAppDir(appID int64, path string) (string, error) {
	dir, err := o.appProjectDir(appID)
	if err != nil {
		return "", err
	}
	appDir, _, err := services.WorktreeAppDir(dir, path)
	return appDir, err
}

// GitWorktreeOpenFolder opens the app's folder inside the worktree at path.
func (o *Ops) GitWorktreeOpenFolder(appID int64, path string) (types.Ok, error) {
	appDir, err := o.worktreeAppDir(appID, path)
	if err != nil {
		return types.Ok{}, err
	}
	if err := native.OpenFolder(appDir); err != nil {
		return types.Ok{}, err
	}
	return types.Ok{Ok: true}, nil
}

// GitWorktreeOpenInEditor opens the app's folder inside the worktree at path in
// an editor (an id from AppsEditors).
func (o *Ops) GitWorktreeOpenInEditor(appID int64, path string, editor string) (types.Ok, error) {
	appDir, err := o.worktreeAppDir(appID, path)
	if err != nil {
		return types.Ok{}, err
	}
	if err := native.OpenInEditor(editor, appDir); err != nil {
		return types.Ok{}, err
	}
	return types.Ok{Ok: true}, nil
}
