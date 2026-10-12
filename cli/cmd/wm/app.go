package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"workspace-manager/api"
	"workspace-manager/api/types"
)

func newAppCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "app", Short: "Manage apps"}
	cmd.AddCommand(
		newAppListCmd(),
		newAppShowCmd(),
		newAppAddCmd(),
		newAppEditCmd(),
		newAppReorderCmd(),
		newAppRmCmd(),
		newAppOpenCmd(),
		newAppEditorsCmd(),
	)
	return cmd
}

func newAppListCmd() *cobra.Command {
	var workspace string
	c := &cobra.Command{
		Use:   "list",
		Short: "List apps",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if workspace != "" {
				ws, err := resolveWorkspace(workspace)
				if err != nil {
					return err
				}
				var out []types.App
				if err := call(api.MethodAppsList, map[string]any{"workspaceID": ws.ID}, &out); err != nil {
					return err
				}
				printJSON(cmd, out)
				return nil
			}
			out, err := allApps()
			if err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&workspace, "workspace", "", "limit to one workspace")
	return c
}

func newAppShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id|name>",
		Short: "Show one app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := resolveApp(args[0])
			if err != nil {
				return err
			}
			printJSON(cmd, app)
			return nil
		},
	}
}

func newAppAddCmd() *cobra.Command {
	var workspace, path string
	c := &cobra.Command{
		Use:   "add <name>",
		Short: "Register an app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := resolveWorkspace(workspace)
			if err != nil {
				return err
			}
			if path == "" {
				return usage("--path is required")
			}
			if abs, err := filepath.Abs(path); err == nil {
				path = abs
			}
			var out types.App
			if err := call(api.MethodAppsCreate, map[string]any{
				"workspaceID": ws.ID,
				"body":        types.AppCreateInput{Name: args[0], ProjectPath: path},
			}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&workspace, "workspace", "", "workspace id or name")
	c.Flags().StringVar(&path, "path", "", "project folder")
	_ = c.MarkFlagRequired("path")
	return c
}

func newAppEditCmd() *cobra.Command {
	var name, path string
	c := &cobra.Command{
		Use:   "edit <id|name>",
		Short: "Change an app's name or project path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := resolveApp(args[0])
			if err != nil {
				return err
			}
			var in types.AppUpdateInput
			if cmd.Flags().Changed("name") {
				in.Name = &name
			}
			if cmd.Flags().Changed("path") {
				if abs, err := filepath.Abs(path); err == nil {
					path = abs
				}
				in.ProjectPath = &path
			}
			var out types.App
			if err := call(api.MethodAppsUpdate, map[string]any{"id": app.ID, "body": in}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "new name")
	c.Flags().StringVar(&path, "path", "", "new project folder")
	return c
}

func newAppReorderCmd() *cobra.Command {
	var workspace string
	c := &cobra.Command{
		Use:   "reorder <id|name> [<id|name>...]",
		Short: "Set the order of a workspace's apps",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			first, err := resolveApp(args[0])
			if err != nil {
				return err
			}
			wsID := first.WorkspaceID
			if workspace != "" {
				ws, err := resolveWorkspace(workspace)
				if err != nil {
					return err
				}
				wsID = ws.ID
			}
			ids := make([]int64, 0, len(args))
			for _, a := range args {
				app, err := resolveApp(a)
				if err != nil {
					return err
				}
				if app.WorkspaceID != wsID {
					return usage("app %s is not in workspace %d", app.Name, wsID)
				}
				ids = append(ids, app.ID)
			}
			var out types.Ok
			if err := call(api.MethodAppsReorder, map[string]any{"workspaceID": wsID, "ids": ids}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&workspace, "workspace", "", "workspace id or name")
	return c
}

func newAppRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id|name>",
		Short: "Remove an app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := resolveApp(args[0])
			if err != nil {
				return err
			}
			if err := confirm(cmd, "Remove app "+app.Name); err != nil {
				return err
			}
			var out types.Ok
			if err := call(api.MethodAppsDelete, map[string]any{"id": app.ID}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

func newAppOpenCmd() *cobra.Command {
	var editor string
	c := &cobra.Command{
		Use:   "open <id|name>",
		Short: "Open the app's folder, in the file manager or an editor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := resolveApp(args[0])
			if err != nil {
				return err
			}
			method := api.MethodAppsOpenFolder
			params := map[string]any{"id": app.ID}
			if editor != "" {
				method = api.MethodAppsOpenInEditor
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

func newAppEditorsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "editors",
		Short: "List editors installed on this machine",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var out []types.Editor
			if err := call(api.MethodAppsEditors, nil, &out); err != nil {
				return err
			}
			if jsonOutput {
				printJSON(cmd, out)
				return nil
			}
			for _, e := range out {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", e.ID, e.Name)
			}
			return nil
		},
	}
}
