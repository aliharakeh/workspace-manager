package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"workspace-manager/api"
	"workspace-manager/api/client"
)

// Global flags shared by every command.
var (
	jsonOutput bool
	socketPath string
	assumeYes  bool
)

// client is the connection to the daemon, filled in by the root command.
var conn *client.Client

// newRootCmd builds the command tree.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "wm",
		Short:         "Manage your local apps, their config and their git state",
		Long:          "wm talks to the workspace-manager daemon (wmd), which it starts when it is not running.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Annotations["local"] == "true" {
				return nil
			}
			c, err := connect()
			if err != nil {
				return err
			}
			conn = c
			return nil
		},
		PersistentPostRun: func(*cobra.Command, []string) {
			if conn != nil {
				_ = conn.Close()
				conn = nil
			}
		},
	}
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON (default for agent use)")
	root.PersistentFlags().StringVar(&socketPath, "socket", "", "daemon socket path (defaults to the data folder)")
	root.PersistentFlags().BoolVarP(&assumeYes, "yes", "y", false, "assume yes for confirmation prompts")

	root.AddCommand(
		newWorkspaceCmd(),
		newAppCmd(),
		newConfigCmd(),
		newEnvCmd(),
		newTemplateCmd(),
		newRunConfigCmd(),
		newRunCmd("run", "Start the app", "run"),
		newBuildCmd(),
		newSetupCmd(),
		newStopCmd(),
		newReloadCmd(),
		newStatusCmd(),
		newLogsCmd(),
		newGitCmd(),
		newPortsCmd(),
		newPatternsCmd(),
		newSettingsCmd(),
		newFsCmd(),
		newOpenCmd(),
		newBlueprintCmd(),
		newServerStopCmd(),
		newDaemonCmd(),
		newVerbListCmd(),
	)
	return root
}

// connect starts the daemon when it is not running and returns a client.
func connect() (*client.Client, error) {
	return client.Dial(client.Options{
		Kind:    "cli",
		Version: version,
		Socket:  socketPath,
		Start:   client.StartDaemon(),
	})
}

// call runs a method and decodes the result into out.
func call(method string, params, out any) error {
	if conn == nil {
		return fmt.Errorf("not connected to the server")
	}
	return conn.Call(method, params, out)
}

// fail reports an error and stops with the matching exit code.
func fail(err error) {
	emitError(err, api.KindOf(err).ExitCode())
	os.Exit(api.KindOf(err).ExitCode())
}
