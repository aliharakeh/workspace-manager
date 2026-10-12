package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"workspace-manager/api"
	"workspace-manager/api/types"
)

func newPortsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "ports", Short: "Find local port conflicts"}

	cmd.AddCommand(
		func() *cobra.Command {
			var port int64
			c := &cobra.Command{
				Use:   "list",
				Short: "List listening processes",
				Args:  cobra.NoArgs,
				RunE: func(cmd *cobra.Command, _ []string) error {
					var out types.PortsListResult
					if err := call(api.MethodPortsList, nil, &out); err != nil {
						return err
					}
					if jsonOutput {
						printJSON(cmd, out)
						return nil
					}
					for _, p := range out.Processes {
						if port != 0 && p.Port != port {
							continue
						}
						fmt.Fprintf(cmd.OutOrStdout(), "%d\t%d\t%s\n", p.Port, p.PID, p.Name)
					}
					return nil
				},
			}
			c.Flags().Int64Var(&port, "port", 0, "only this port")
			return c
		}(),
		&cobra.Command{
			Use:   "kill <pid>",
			Short: "Stop the process listening on a port",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				pid, err := parseID(args[0], "pid")
				if err != nil {
					return err
				}
				if err := confirm(cmd, fmt.Sprintf("Stop process %d", pid)); err != nil {
					return err
				}
				var out types.PortsKillResult
				if err := call(api.MethodPortsKill, map[string]any{"pid": pid}, &out); err != nil {
					return err
				}
				printJSON(cmd, out)
				return nil
			},
		},
	)
	return cmd
}

func newPatternsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "patterns", Short: "Manage the ready-URL patterns"}

	help := "Pattern must include a named group `url` and/or `port`"

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List ready-URL patterns",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				var out []types.ReadyUrlPattern
				if err := call(api.MethodReadyUrlPatternsList, nil, &out); err != nil {
					return err
				}
				printJSON(cmd, out)
				return nil
			},
		},
		func() *cobra.Command {
			var flags string
			c := &cobra.Command{
				Use:   "add <label> <pattern>",
				Short: "Add a pattern",
				Long:  help,
				Args:  cobra.ExactArgs(2),
				RunE: func(cmd *cobra.Command, args []string) error {
					in := types.ReadyUrlPatternCreateInput{Label: args[0], Pattern: args[1]}
					if flags != "" {
						in.Flags = &flags
					}
					var out types.ReadyUrlPattern
					if err := call(api.MethodReadyUrlPatternsCreate, in, &out); err != nil {
						return err
					}
					printJSON(cmd, out)
					return nil
				},
			}
			c.Flags().StringVar(&flags, "flags", "i", "regex flags")
			return c
		}(),
		func() *cobra.Command {
			var label, pattern, flags string
			c := &cobra.Command{
				Use:   "edit <id>",
				Short: "Change a pattern",
				Args:  cobra.ExactArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					id, err := parseID(args[0], "pattern")
					if err != nil {
						return err
					}
					in := types.ReadyUrlPatternUpdateInput{}
					if cmd.Flags().Changed("label") {
						in.Label = &label
					}
					if cmd.Flags().Changed("pattern") {
						in.Pattern = &pattern
					}
					if cmd.Flags().Changed("flags") {
						in.Flags = &flags
					}
					var out types.ReadyUrlPattern
					if err := call(api.MethodReadyUrlPatternsUpdate, map[string]any{"id": id, "body": in}, &out); err != nil {
						return err
					}
					printJSON(cmd, out)
					return nil
				},
			}
			c.Flags().StringVar(&label, "label", "", "new label")
			c.Flags().StringVar(&pattern, "pattern", "", "new regex")
			c.Flags().StringVar(&flags, "flags", "", "new flags")
			return c
		}(),
		&cobra.Command{
			Use:   "rm <id>",
			Short: "Delete a pattern",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				id, err := parseID(args[0], "pattern")
				if err != nil {
					return err
				}
				var out types.Ok
				if err := call(api.MethodReadyUrlPatternsDelete, map[string]any{"id": id}, &out); err != nil {
					return err
				}
				printJSON(cmd, out)
				return nil
			},
		},
	)
	return cmd
}

func newSettingsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "settings", Short: "Read and change settings"}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "get [key]",
			Short: "Show settings",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				var out map[string]string
				if err := call(api.MethodSettingsGet, nil, &out); err != nil {
					return err
				}
				if len(args) == 1 {
					value, okValue := out[args[0]]
					if !okValue && !jsonOutput {
						return api.Errf(api.KindNotFound, "no setting %q", args[0])
					}
					printJSON(cmd, map[string]string{args[0]: value})
					return nil
				}
				printJSON(cmd, out)
				return nil
			},
		},
		&cobra.Command{
			Use:   "set <key> <value>",
			Short: "Change a setting",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				var out map[string]string
				if err := call(api.MethodSettingsSet, types.SettingsSetInput{Key: args[0], Value: args[1]}, &out); err != nil {
					return err
				}
				printJSON(cmd, out)
				return nil
			},
		},
	)
	return cmd
}

func newFsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "fs", Short: "Inspect the files of an app"}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "validate <path>",
			Short: "Check that a path is a usable project folder",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				var out types.ValidatePathResult
				if err := call(api.MethodFsValidatePath, map[string]any{"path": args[0]}, &out); err != nil {
					return err
				}
				printJSON(cmd, out)
				if !out.Ok {
					return api.Errf(api.KindInvalid, "%s", out.Error)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "read <app> <file>",
			Short: "Read a text file inside an app's project folder",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				app, err := resolveApp(args[0])
				if err != nil {
					return err
				}
				var out types.ReadAppFileResult
				if err := call(api.MethodFsReadAppFile, map[string]any{"appID": app.ID, "path": args[1]}, &out); err != nil {
					return err
				}
				if jsonOutput {
					printJSON(cmd, out)
					return nil
				}
				fmt.Fprint(cmd.OutOrStdout(), out.Content)
				return nil
			},
		},
	)
	return cmd
}

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <url>",
		Short: "Open a URL in the browser",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out types.Ok
			if err := call(api.MethodOpenExternal, map[string]any{"url": args[0]}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

func newServerStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Control the daemon",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon; its running apps stop too",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var out types.Ok
			if err := call(api.MethodStop, nil, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	})
	return cmd
}

func newDaemonCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "daemon",
		Short:  "Run the server in the foreground",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return api.Errf(api.KindUnavailable, "the daemon is its own binary; run wmd")
		},
	}
}

func newVerbListCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "methods",
		Short:  "List the API methods this CLI can call",
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			names := make([]string, 0, len(api.Methods))
			for _, m := range api.Methods {
				if m.CLI {
					names = append(names, m.Name)
				}
			}
			printJSON(cmd, names)
			return nil
		},
	}
}