package ops

import "workspace-manager/shared/api/lib"

// WorktreeLinksGet returns the app's shared worktree paths: files and folders,
// relative to the app folder, that its worktrees link to instead of copying.
func (o *Ops) WorktreeLinksGet(appID int64) ([]string, error) {
	return o.DB.ListWorktreeLinksT(o.Ctx(), appID)
}

// WorktreeLinksSave replaces the app's shared worktree paths and returns them
// cleaned up.
func (o *Ops) WorktreeLinksSave(appID int64, paths []string) ([]string, error) {
	clean, err := lib.NormalizeWorktreeLinks(paths)
	if err != nil {
		return nil, err
	}
	if err := o.DB.SetWorktreeLinksT(o.Ctx(), appID, clean); err != nil {
		return nil, err
	}
	o.changed("worktreeLinks", appID)
	return clean, nil
}
