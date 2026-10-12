package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"workspace-manager/api"
	"workspace-manager/api/types"
)

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "env", Short: "Manage an app's environment variables"}
	cmd.AddCommand(newEnvListCmd(), newEnvSetCmd(), newEnvRmCmd(), newEnvImportCmd())
	return cmd
}

func newEnvListCmd() *cobra.Command {
	var configSet string
	c := &cobra.Command{
		Use:   "list [app]",
		Short: "List the active config set's env vars",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(args)
			if err != nil {
				return err
			}
			if configSet != "" {
				set, err := activeConfigSet(app, configSet)
				if err != nil {
					return err
				}
				var detail types.ConfigSetDetail
				if err := call(api.MethodConfigSetsDetail, map[string]any{"id": set.ID}, &detail); err != nil {
					return err
				}
				printJSON(cmd, detail.EnvVars)
				return nil
			}
			vars, err := envVarsOf(app)
			if err != nil {
				return err
			}
			printJSON(cmd, vars)
			return nil
		},
	}
	c.Flags().StringVar(&configSet, "config-set", "", "config set id or name")
	return c
}

func newEnvSetCmd() *cobra.Command {
	var value string
	var hidden bool
	c := &cobra.Command{
		Use:   "set <KEY> [app]",
		Short: "Create or update an env var on the active config set",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			key := strings.TrimSpace(args[0])
			if key == "" {
				return usage("the key cannot be empty")
			}
			include := !hidden
			vars, err := envVarsOf(app)
			if err != nil {
				return err
			}
			var out types.EnvVar
			for _, v := range vars {
				if v.Key != key {
					continue
				}
				up := types.EnvVarUpdateInput{Key: &key, IncludeInAI: &include}
				if cmd.Flags().Changed("value") {
					up.Value = &value
				}
				if err := call(api.MethodEnvVarsUpdate, map[string]any{"id": v.ID, "body": up}, &out); err != nil {
					return err
				}
				printJSON(cmd, out)
				return nil
			}
			in := types.EnvVarCreateInput{Key: key, IncludeInAI: &include}
			if cmd.Flags().Changed("value") {
				in.Value = &value
			}
			if err := call(api.MethodEnvVarsCreate, map[string]any{"appID": app.ID, "body": in}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&value, "value", "", "the value")
	c.Flags().BoolVar(&hidden, "hidden", false, "hide the value from AI clients (include_in_ai=false)")
	return c
}

func newEnvRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <KEY> [app]",
		Short: "Delete an env var",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			vars, err := envVarsOf(app)
			if err != nil {
				return err
			}
			for _, v := range vars {
				if v.Key != args[0] {
					continue
				}
				if err := confirm(cmd, "Delete env var "+v.Key); err != nil {
					return err
				}
				var out types.Ok
				if err := call(api.MethodEnvVarsDelete, map[string]any{"id": v.ID}, &out); err != nil {
					return err
				}
				printJSON(cmd, out)
				return nil
			}
			return api.Errf(api.KindNotFound, "no env var named %q on app %s", args[0], app.Name)
		},
	}
}

func newEnvImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <file> [app]",
		Short: "Import a .env or YAML file into the active config set",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			path := args[0]
			if !filepath.IsAbs(path) {
				if abs, err := filepath.Abs(path); err == nil {
					path = abs
				}
			}
			if _, err := os.Stat(path); err != nil {
				return api.Errf(api.KindNotFound, "cannot read %s", path)
			}
			var out types.ImportEnvResult
			if err := call(api.MethodEnvVarsImport, map[string]any{"appID": app.ID, "path": path}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

// envVarsOf returns the active config set's env vars.
func envVarsOf(app types.App) ([]types.EnvVar, error) {
	var out []types.EnvVar
	if err := call(api.MethodEnvVarsList, map[string]any{"appID": app.ID}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
