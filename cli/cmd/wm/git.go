package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"workspace-manager/api"
	"workspace-manager/api/types"
)

func newGitCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "git", Short: "Branches, worktrees and shared paths"}
	cmd.AddCommand(
		newGitInfoCmd(),
		newGitFetchCmd(),
		newBranchCmd(),
		newWorktreeCmd(),
		newGitOpenFileCmd(),
	)
	return cmd
}

func newGitInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info [app]",
		Short: "Show the repository state: worktrees and branches",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(args)
			if err != nil {
				return err
			}
			var out types.GitInfo
			if err := call(api.MethodGitInfo, map[string]any{"appID": app.ID}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

func newGitFetchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch [app]",
		Short: "Fetch every remote and prune deleted branches",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(args)
			if err != nil {
				return err
			}
			var out types.GitOutput
			if err := call(api.MethodGitFetchAll, map[string]any{"appID": app.ID}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

func newGitOpenFileCmd() *cobra.Command {
	var editor string
	c := &cobra.Command{
		Use:   "open-file <file> [app]",
		Short: "Open a file of one of the repo's worktrees in an editor",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			if editor == "" {
				return usage("--editor is required")
			}
			var out types.Ok
			if err := call(api.MethodGitFileOpenInEditor, map[string]any{
				"appID": app.ID, "file": args[0], "editor": editor,
			}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&editor, "editor", "", "editor id from 'wm app editors'")
	return c
}

func newBranchCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "branch", Short: "Manage local branches"}
	cmd.AddCommand(
		newBranchAddCmd(),
		newBranchRenameCmd(),
		newBranchCheckoutCmd(),
		newBranchPullCmd(),
		newBranchPushCmd(),
		newBranchRmCmd(),
	)
	return cmd
}

func newBranchAddCmd() *cobra.Command {
	var base string
	c := &cobra.Command{
		Use:   "add <name> [app]",
		Short: "Create a local branch without checking it out",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			var out types.Ok
			if err := call(api.MethodGitBranchCreate, map[string]any{
				"appID": app.ID, "name": args[0], "base": base,
			}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&base, "base", "", "create from this ref (default HEAD)")
	return c
}

func newBranchRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old> <new> [app]",
		Short: "Rename a local branch",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			var out types.Ok
			if err := call(api.MethodGitBranchRename, map[string]any{
				"appID": app.ID, "oldName": args[0], "newName": args[1],
			}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

func newBranchCheckoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "checkout <name> [app]",
		Short: "Switch the app's own folder to a local branch",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			var out types.Ok
			if err := call(api.MethodGitBranchCheckout, map[string]any{
				"appID": app.ID, "name": args[0],
			}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

func newBranchPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull <name> [app]",
		Short: "Pull a branch in the worktree that has it checked out",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			var out types.GitPullResult
			if err := call(api.MethodGitBranchPull, map[string]any{
				"appID": app.ID, "name": args[0],
			}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			if len(out.Conflicts) > 0 {
				return api.Errf(api.KindConflict, "the pull stopped on %d conflicted files; resolve them and commit", len(out.Conflicts))
			}
			return nil
		},
	}
}

func newBranchPushCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "push <name> [app]",
		Short: "Push a branch, publishing it when it has no upstream",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			var out types.GitOutput
			if err := call(api.MethodGitBranchPush, map[string]any{
				"appID": app.ID, "name": args[0], "force": force,
			}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().BoolVar(&force, "force", false, "use --force-with-lease")
	return c
}

func newBranchRmCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "rm <name> [app]",
		Short: "Delete a local branch",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			if err := confirm(cmd, "Delete branch "+args[0]); err != nil {
				return err
			}
			var out types.Ok
			if err := call(api.MethodGitBranchDelete, map[string]any{
				"appID": app.ID, "name": args[0], "force": force,
			}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().BoolVar(&force, "force", false, "delete even when it is not merged")
	return c
}

func newWorktreeCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "worktree", Short: "Manage git worktrees"}
	cmd.AddCommand(
		newWorktreeAddCmd(),
		newWorktreeRmCmd(),
		newWorktreePruneCmd(),
		newWorktreeStartCmd(),
		newWorktreeLinksCmd(),
		newWorktreeOpenCmd(),
		newWorktreeStatusCmd(),
	)
	return cmd
}

func newWorktreeAddCmd() *cobra.Command {
	var newBranch, linkShared bool
	var base, path string
	var links []string
	c := &cobra.Command{
		Use:   "add <branch> [app]",
		Short: "Add a worktree",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			in := types.GitWorktreeAddInput{
				Path:       path,
				Branch:     args[0],
				NewBranch:  newBranch,
				Base:       base,
				LinkShared: linkShared,
			}
			if cmd.Flags().Changed("links") {
				in.Links = links
			}
			var out types.GitWorktreeAddResult
			if err := call(api.MethodGitWorktreeAdd, map[string]any{"appID": app.ID, "body": in}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().BoolVar(&newBranch, "new-branch", false, "create the branch instead of checking out an existing one")
	c.Flags().StringVar(&base, "base", "", "create the new branch from this ref (default HEAD)")
	c.Flags().StringVar(&path, "path", "", "where to put the worktree (default: a sibling of the repo)")
	c.Flags().BoolVar(&linkShared, "link-shared", true, "link the app's shared paths into the worktree")
	c.Flags().StringArrayVar(&links, "links", nil, "shared paths for this worktree only, instead of the app's")
	return c
}

func newWorktreeRmCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "rm <path> [app]",
		Short: "Remove a worktree; the branch is kept",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			if err := confirm(cmd, "Remove worktree "+args[0]); err != nil {
				return err
			}
			var out types.Ok
			if err := call(api.MethodGitWorktreeRemove, map[string]any{
				"appID": app.ID, "path": args[0], "force": force,
			}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().BoolVar(&force, "force", false, "remove even with uncommitted changes")
	return c
}

func newWorktreePruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prune [app]",
		Short: "Drop records of worktrees whose folder is gone",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(args)
			if err != nil {
				return err
			}
			var out types.GitOutput
			if err := call(api.MethodGitWorktreePrune, map[string]any{"appID": app.ID}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

func newWorktreeStartCmd() *cobra.Command {
	var kind, configSet string
	c := &cobra.Command{
		Use:   "start <path> [app]",
		Short: "Run the app's run, build or setup config inside a worktree",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			if err := checkKind(kind); err != nil {
				return err
			}
			var setID int64
			if configSet != "" {
				set, err := activeConfigSet(app, configSet)
				if err != nil {
					return err
				}
				setID = set.ID
			}
			var out types.StatusEvent
			if err := call(api.MethodGitWorktreeStart, map[string]any{
				"appID": app.ID, "path": args[0], "kind": kind, "configSetID": setID,
			}, &out); err != nil {
				return err
			}
			printSession(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&kind, "kind", "setup", "run, build or setup")
	c.Flags().StringVar(&configSet, "config-set", "", "config set id or name")
	return c
}

func newWorktreeLinksCmd() *cobra.Command {
	var links []string
	c := &cobra.Command{
		Use:   "links [app]",
		Short: "Show or replace the app's shared worktree paths",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(args)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("set") {
				var out []string
				if err := call(api.MethodWorktreeLinksGet, map[string]any{"appID": app.ID}, &out); err != nil {
					return err
				}
				printJSON(cmd, out)
				return nil
			}
			var out []string
			if err := call(api.MethodWorktreeLinksSave, map[string]any{"appID": app.ID, "paths": links}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringArrayVar(&links, "set", nil, "replace the app's shared paths with these")
	return c
}

func newWorktreeOpenCmd() *cobra.Command {
	var editor string
	c := &cobra.Command{
		Use:   "open <path> [app]",
		Short: "Open the app's folder inside a worktree",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			method := api.MethodGitWorktreeOpenFolder
			params := map[string]any{"appID": app.ID, "path": args[0]}
			if editor != "" {
				method = api.MethodGitWorktreeOpenEditor
				params["editor"] = editor
			}
			var out types.Ok
			if err := call(method, params, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&editor, "editor", "", "editor id from 'wm app editors'")
	return c
}

func newWorktreeStatusCmd() *cobra.Command {
	var create bool
	c := &cobra.Command{
		Use:   "status <path> [app]",
		Short: "Report each shared path in a worktree",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			var out []types.WorktreeLinkState
			if err := call(api.MethodGitWorktreeLinks, map[string]any{
				"appID": app.ID, "path": args[0], "create": create,
			}, &out); err != nil {
				return err
			}
			if jsonOutput {
				printJSON(cmd, out)
				return nil
			}
			for _, s := range out {
				line := fmt.Sprintf("%s\t%s", s.Path, s.State)
				if s.Message != "" {
					line += "\t" + s.Message
				}
				fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&create, "create", false, "link the missing paths first")
	return c
}
