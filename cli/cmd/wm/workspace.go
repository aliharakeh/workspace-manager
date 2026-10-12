package main

import (
	"github.com/spf13/cobra"

	"workspace-manager/api"
	"workspace-manager/api/types"
)

func newWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "workspace", Short: "Manage workspaces"}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List workspaces",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				var out []types.Workspace
				if err := call(api.MethodWorkspacesList, nil, &out); err != nil {
					return err
				}
				printJSON(cmd, out)
				return nil
			},
		},
		func() *cobra.Command {
			var icon, color string
			c := &cobra.Command{
				Use:   "add <name>",
				Short: "Create a workspace",
				Args:  cobra.ExactArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					in := types.WorkspaceCreateInput{Name: args[0]}
					if icon != "" {
						in.Icon = &icon
					}
					if color != "" {
						in.Color = &color
					}
					var out types.Workspace
					if err := call(api.MethodWorkspacesCreate, in, &out); err != nil {
						return err
					}
					printJSON(cmd, out)
					return nil
				},
			}
			c.Flags().StringVar(&icon, "icon", "", "icon name")
			c.Flags().StringVar(&color, "color", "", "hex color like #3b82f6")
			return c
		}(),
		func() *cobra.Command {
			var name, icon, color string
			c := &cobra.Command{
				Use:   "edit <id|name>",
				Short: "Change a workspace",
				Args:  cobra.ExactArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					ws, err := resolveWorkspace(args[0])
					if err != nil {
						return err
					}
					var in types.WorkspaceUpdateInput
					if cmd.Flags().Changed("name") {
						in.Name = &name
					}
					if cmd.Flags().Changed("icon") {
						in.Icon = &icon
					}
					if cmd.Flags().Changed("color") {
						in.Color = &color
					}
					var out types.Workspace
					if err := call(api.MethodWorkspacesUpdate, map[string]any{"id": ws.ID, "body": in}, &out); err != nil {
						return err
					}
					printJSON(cmd, out)
					return nil
				},
			}
			c.Flags().StringVar(&name, "name", "", "new name")
			c.Flags().StringVar(&icon, "icon", "", "icon name")
			c.Flags().StringVar(&color, "color", "", "hex color; empty clears it")
			return c
		}(),
		&cobra.Command{
			Use:   "rm <id|name>",
			Short: "Delete a workspace and its apps",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				ws, err := resolveWorkspace(args[0])
				if err != nil {
					return err
				}
				if err := confirm(cmd, "Delete workspace "+ws.Name+" and its apps"); err != nil {
					return err
				}
				var out types.Ok
				if err := call(api.MethodWorkspacesDelete, map[string]any{"id": ws.ID}, &out); err != nil {
					return err
				}
				printJSON(cmd, out)
				return nil
			},
		},
	)
	return cmd
}
