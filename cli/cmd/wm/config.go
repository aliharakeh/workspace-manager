package main

import (
	"strings"

	"github.com/spf13/cobra"

	"workspace-manager/api"
	"workspace-manager/api/types"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Manage config sets"}
	cmd.AddCommand(
		newConfigListCmd(),
		newConfigShowCmd(),
		newConfigAddCmd(),
		newConfigRenameCmd(),
		newConfigUseCmd(),
		newConfigCopyCmd(),
		newConfigRmCmd(),
	)
	return cmd
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [app]",
		Short: "List an app's config sets",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(args)
			if err != nil {
				return err
			}
			var out []types.ConfigSet
			if err := call(api.MethodConfigSetsList, map[string]any{"appID": app.ID}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show one config set with its env vars, templates and command lists",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0], "config set")
			if err != nil {
				return err
			}
			var out types.ConfigSetDetail
			if err := call(api.MethodConfigSetsDetail, map[string]any{"id": id}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

func newConfigAddCmd() *cobra.Command {
	var from, parts string
	var noActivate bool
	c := &cobra.Command{
		Use:   "add <name> [app]",
		Short: "Create a config set, optionally copying from another",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			in := types.ConfigSetCreateInput{Name: args[0]}
			if noActivate {
				active := false
				in.Activate = &active
			}
			if from != "" {
				source, err := activeConfigSet(app, from)
				if err != nil {
					return err
				}
				in.CopyFromID = &source.ID
				if parts != "" {
					in.Parts = parseParts(parts)
				}
			}
			var out types.ConfigSet
			if err := call(api.MethodConfigSetsCreate, map[string]any{"appID": app.ID, "body": in}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&from, "from", "", "config set to copy from, by id or name")
	c.Flags().StringVar(&parts, "parts", "env,templates,run,build,setup", "parts to copy")
	c.Flags().BoolVar(&noActivate, "no-activate", false, "do not make the new set active")
	return c
}

func newConfigRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <id|name> <new-name> [app]",
		Short: "Rename a config set",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			set, err := activeConfigSet(app, args[0])
			if err != nil {
				return err
			}
			var out types.ConfigSet
			if err := call(api.MethodConfigSetsUpdate, map[string]any{
				"id":   set.ID,
				"body": types.ConfigSetUpdateInput{Name: args[1]},
			}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

func newConfigUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <id|name> [app]",
		Short: "Make a config set the app's active one",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			set, err := activeConfigSet(app, args[0])
			if err != nil {
				return err
			}
			var out types.ConfigSetActivateResult
			if err := call(api.MethodConfigSetsActivate, map[string]any{"id": set.ID}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

func newConfigCopyCmd() *cobra.Command {
	var from, parts string
	c := &cobra.Command{
		Use:   "copy <id|name> --from <id|name> [app]",
		Short: "Copy parts of another config set into this one",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			target, err := activeConfigSet(app, args[0])
			if err != nil {
				return err
			}
			if from == "" {
				return usage("--from is required")
			}
			source, err := activeConfigSet(app, from)
			if err != nil {
				return err
			}
			var out types.ConfigSet
			if err := call(api.MethodConfigSetsCopyFrom, map[string]any{
				"id":       target.ID,
				"sourceID": source.ID,
				"parts":    parseParts(parts),
			}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&from, "from", "", "source config set, by id or name")
	c.Flags().StringVar(&parts, "parts", "env,templates,run,build,setup", "parts to copy")
	return c
}

func newConfigRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id|name> [app]",
		Short: "Delete a config set",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			set, err := activeConfigSet(app, args[0])
			if err != nil {
				return err
			}
			if err := confirm(cmd, "Delete config set "+set.Name); err != nil {
				return err
			}
			var out types.Ok
			if err := call(api.MethodConfigSetsDelete, map[string]any{"id": set.ID}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

// parseParts turns "env,templates" into the parts object the API expects.
func parseParts(list string) *types.CopyParts {
	parts := &types.CopyParts{}
	for _, p := range strings.Split(list, ",") {
		switch strings.TrimSpace(p) {
		case "env":
			parts.Env = true
		case "templates":
			parts.Templates = true
		case "run":
			parts.Run = true
		case "build":
			parts.Build = true
		case "setup":
			parts.Setup = true
		}
	}
	return parts
}

// rest returns everything after the first argument, for commands whose last
// argument is the app.
func rest(args []string) []string {
	if len(args) > 1 {
		return args[1:]
	}
	return nil
}

// appFromArgs picks the app argument out of a command's trailing arguments, or
// falls back to the only app when there is exactly one.
func appFromArgs(args []string) (types.App, error) {
	if len(args) > 0 && args[0] != "" {
		return resolveApp(args[0])
	}
	apps, err := allApps()
	if err != nil {
		return types.App{}, err
	}
	switch len(apps) {
	case 0:
		return types.App{}, api.Errf(api.KindNotFound, "there are no apps yet; run 'wm app add' first")
	case 1:
		return apps[0], nil
	default:
		return types.App{}, usage("several apps exist; pass the app as the last argument")
	}
}
