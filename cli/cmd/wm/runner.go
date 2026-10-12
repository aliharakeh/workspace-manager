package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"workspace-manager/api"
	"workspace-manager/api/types"
)

// newRunConfigCmd manages the run, build and setup command lists.
func newRunConfigCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "commands", Short: "Show or set an app's command lists"}
	cmd.AddCommand(newCommandsShowCmd(), newCommandsSetCmd(), newCommandsScriptsCmd())
	return cmd
}

func newCommandsShowCmd() *cobra.Command {
	var app string
	c := &cobra.Command{
		Use:   "show <run|build|setup>",
		Short: "Show one command list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := appFromArgs([]string{app})
			if err != nil {
				return err
			}
			if err := checkKind(args[0]); err != nil {
				return err
			}
			var out types.RunConfig
			if err := call(api.MethodRunConfigGet, map[string]any{"appID": a.ID, "kind": args[0]}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&app, "app", "", "app id or name")
	return c
}

func newCommandsSetCmd() *cobra.Command {
	var app, mode string
	var command, label []string
	c := &cobra.Command{
		Use:   "set <run|build|setup>",
		Short: "Replace a command list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := appFromArgs([]string{app})
			if err != nil {
				return err
			}
			if err := checkKind(args[0]); err != nil {
				return err
			}
			if len(command) == 0 {
				return usage("pass at least one --command")
			}
			cmds := make([]types.RunCommandInput, 0, len(command))
			for i, line := range command {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				in := types.RunCommandInput{Command: line}
				if i < len(label) && strings.TrimSpace(label[i]) != "" {
					l := strings.TrimSpace(label[i])
					in.Label = &l
				}
				cmds = append(cmds, in)
			}
			body := types.RunConfigSaveInput{Commands: cmds}
			if mode != "" {
				body.Mode = &mode
			}
			var out types.RunConfig
			if err := call(api.MethodRunConfigSave, map[string]any{"appID": a.ID, "kind": args[0], "body": body}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&app, "app", "", "app id or name")
	c.Flags().StringVar(&mode, "mode", "", "parallel or sequential")
	c.Flags().StringArrayVar(&command, "command", nil, "a command line, in order")
	c.Flags().StringArrayVar(&label, "label", nil, "a label for the matching --command, in order")
	return c
}

func newCommandsScriptsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scripts [app]",
		Short: "List package.json scripts, if the project has any",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := appFromArgs(args)
			if err != nil {
				return err
			}
			var out types.PackageScripts
			if err := call(api.MethodPackageScriptsList, map[string]any{"appID": a.ID}, &out); err != nil {
				return err
			}
			if jsonOutput || !out.HasPackageJSON {
				printJSON(cmd, out)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "package manager: %s\n", out.PackageManager)
			for _, s := range out.Scripts {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", s.Name, s.Command)
			}
			return nil
		},
	}
}

// runOpts are the shared flags of run, build and setup.
type runOpts struct {
	app       string
	configSet string
	worktree  string
	waitReady bool
	timeout   time.Duration
}

func addRunFlags(c *cobra.Command, o *runOpts) {
	c.Flags().StringVar(&o.app, "app", "", "app id or name")
	c.Flags().StringVar(&o.configSet, "config-set", "", "config set id or name")
	c.Flags().StringVar(&o.worktree, "worktree", "", "run inside this git worktree")
	c.Flags().BoolVar(&o.waitReady, "wait-ready", false, "wait until a ready URL appears")
	c.Flags().DurationVar(&o.timeout, "timeout", 60*time.Second, "how long --wait-ready waits")
}

// startSession starts a run, build or setup session.
func startSession(kind string, o runOpts) (types.StatusEvent, error) {
	app, err := appFromArgs([]string{o.app})
	if err != nil {
		return types.StatusEvent{}, err
	}
	var setID int64
	if o.configSet != "" {
		set, err := activeConfigSet(app, o.configSet)
		if err != nil {
			return types.StatusEvent{}, err
		}
		setID = set.ID
	}
	var out types.StatusEvent
	if o.worktree != "" {
		err = call(api.MethodGitWorktreeStart, map[string]any{
			"appID": app.ID, "path": o.worktree, "kind": kind, "configSetID": setID,
		}, &out)
	} else {
		method := api.MethodRunnerRun
		switch kind {
		case types.KindBuild:
			method = api.MethodRunnerBuild
		case types.KindSetup:
			method = api.MethodRunnerSetup
		}
		err = call(method, map[string]any{"appID": app.ID}, &out)
	}
	if err != nil {
		return types.StatusEvent{}, err
	}
	return out, nil
}

func newRunCmd(name, short, kind string) *cobra.Command {
	o := &runOpts{}
	c := &cobra.Command{
		Use:   name + " [app]",
		Short: short,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.app = pickArg(args, o.app)
			res, err := startSession(kind, *o)
			if err != nil {
				return err
			}
			if o.waitReady {
				return waitReady(cmd, res, o.timeout)
			}
			printSession(cmd, res)
			return nil
		},
	}
	addRunFlags(c, o)
	return c
}

func newBuildCmd() *cobra.Command {
	o := &runOpts{}
	c := &cobra.Command{
		Use:   "build [app]",
		Short: "Build the app",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.app = pickArg(args, o.app)
			res, err := startSession(types.KindBuild, *o)
			if err != nil {
				return err
			}
			printSession(cmd, res)
			return nil
		},
	}
	addRunFlags(c, o)
	return c
}

func newSetupCmd() *cobra.Command {
	o := &runOpts{}
	c := &cobra.Command{
		Use:   "setup [app]",
		Short: "Prepare the app, e.g. install its dependencies",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.app = pickArg(args, o.app)
			res, err := startSession(types.KindSetup, *o)
			if err != nil {
				return err
			}
			printSession(cmd, res)
			return nil
		},
	}
	addRunFlags(c, o)
	return c
}

func newStopCmd() *cobra.Command {
	var app string
	c := &cobra.Command{
		Use:   "stop [app]",
		Short: "Stop the app's session",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := appFromArgs([]string{pickArg(args, app)})
			if err != nil {
				return err
			}
			var out types.StatusEvent
			if err := call(api.MethodRunnerStop, map[string]any{"appID": a.ID}, &out); err != nil {
				return err
			}
			printSession(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&app, "app", "", "app id or name")
	return c
}

func newReloadCmd() *cobra.Command {
	var app string
	c := &cobra.Command{
		Use:   "reload [app]",
		Short: "Restart the app's session where it last ran",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := appFromArgs([]string{pickArg(args, app)})
			if err != nil {
				return err
			}
			var out types.StatusEvent
			if err := call(api.MethodRunnerReload, map[string]any{"appID": a.ID}, &out); err != nil {
				return err
			}
			printSession(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&app, "app", "", "app id or name")
	return c
}

func newStatusCmd() *cobra.Command {
	var workspace string
	var follow bool
	c := &cobra.Command{
		Use:   "status [app]",
		Short: "Show what is running, optionally following changes",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if follow {
				return followStatus(cmd, args, workspace)
			}
			if len(args) == 0 {
				if workspace != "" {
					ws, err := resolveWorkspace(workspace)
					if err != nil {
						return err
					}
					events, err := workspaceStatus(ws.ID)
					if err != nil {
						return err
					}
					printJSON(cmd, events)
					return nil
				}
				workspaces, err := allWorkspaces()
				if err != nil {
					return err
				}
				all := []types.StatusEvent{}
				for _, w := range workspaces {
					events, err := workspaceStatus(w.ID)
					if err != nil {
						return err
					}
					all = append(all, events...)
				}
				printJSON(cmd, all)
				return nil
			}
			app, err := resolveApp(args[0])
			if err != nil {
				return err
			}
			out, err := runnerStatus(app.ID)
			if err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&workspace, "workspace", "", "report every app in this workspace")
	c.Flags().BoolVarP(&follow, "follow", "f", false, "keep printing status changes")
	return c
}

func workspaceStatus(id int64) ([]types.StatusEvent, error) {
	var out []types.StatusEvent
	if err := call(api.MethodRunnerWorkspaceStatus, map[string]any{"workspaceID": id}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func newLogsCmd() *cobra.Command {
	var app string
	var commandID int64
	var tail int
	var follow bool
	c := &cobra.Command{
		Use:   "logs [app]",
		Short: "Show the recent output of the app's commands",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := appFromArgs([]string{pickArg(args, app)})
			if err != nil {
				return err
			}
			status, err := runnerStatus(a.ID)
			if err != nil {
				return err
			}
			if len(status.Processes) == 0 {
				return api.Errf(api.KindNotFound, "app %s has no active session", a.Name)
			}
			ids := []int64{}
			for _, p := range status.Processes {
				if commandID == 0 || p.CommandID == commandID {
					ids = append(ids, p.CommandID)
				}
			}
			if len(ids) == 0 {
				return api.Errf(api.KindNotFound, "no command %d in the app's session", commandID)
			}
			for _, id := range ids {
				var out types.RunnerOutput
				if err := call(api.MethodRunnerOutput, map[string]any{"appID": a.ID, "commandID": id}, &out); err != nil {
					return err
				}
				data, err := base64.StdEncoding.DecodeString(out.Data)
				if err != nil {
					return err
				}
				writeLog(cmd, a.ID, id, tail, data)
			}
			if !follow {
				return nil
			}
			return followLogs(cmd, a.ID, commandID)
		},
	}
	c.Flags().StringVar(&app, "app", "", "app id or name")
	c.Flags().Int64Var(&commandID, "command", 0, "only this command id")
	c.Flags().IntVar(&tail, "tail", 0, "only the last N bytes")
	c.Flags().BoolVarP(&follow, "follow", "f", false, "keep printing new output")
	return c
}

// writeLog prints terminal output; JSON mode wraps it in an event object.
func writeLog(cmd *cobra.Command, appID, commandID int64, tail int, data []byte) {
	if tail > 0 && len(data) > tail {
		data = data[len(data)-tail:]
	}
	if jsonOutput {
		_ = json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"appId": appID, "commandId": commandID, "data": string(data), "ts": time.Now().UnixMilli(),
		})
		return
	}
	_, _ = cmd.OutOrStdout().Write(data)
}

// followLogs subscribes to runner events and prints command output until the
// user interrupts.
func followLogs(cmd *cobra.Command, appID, commandID int64) error {
	if conn == nil {
		return api.Errf(api.KindUnavailable, "not connected")
	}
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer stop()

	events, err := conn.SubscribeRaw(api.SubscribeInput{
		Types: []api.EventType{api.EventRunner},
		Apps:  []int64{appID},
	})
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-conn.Done():
			return api.Errf(api.KindUnavailable, "the connection to the server closed")
		case raw := <-events:
			log, ok := parseRunnerLog(raw)
			if !ok {
				continue
			}
			if commandID != 0 && log.CommandID != commandID {
				continue
			}
			data, err := base64.StdEncoding.DecodeString(log.Data)
			if err != nil {
				continue
			}
			_, _ = cmd.OutOrStdout().Write(data)
		}
	}
}

// followStatus prints every status change until interrupted.
func followStatus(cmd *cobra.Command, args []string, workspace string) error {
	if conn == nil {
		return api.Errf(api.KindUnavailable, "not connected")
	}
	filter := api.SubscribeInput{Types: []api.EventType{api.EventRunner}}
	if len(args) > 0 {
		app, err := resolveApp(args[0])
		if err != nil {
			return err
		}
		filter.Apps = []int64{app.ID}
	} else if workspace != "" {
		ws, err := resolveWorkspace(workspace)
		if err != nil {
			return err
		}
		events, err := workspaceStatus(ws.ID)
		if err != nil {
			return err
		}
		apps := make([]int64, 0, len(events))
		for _, ev := range events {
			apps = append(apps, ev.AppID)
		}
		filter.Apps = apps
	}
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer stop()
	events, err := conn.SubscribeRaw(filter)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-conn.Done():
			return api.Errf(api.KindUnavailable, "the connection to the server closed")
		case raw := <-events:
			if status, ok := parseRunnerStatus(raw); ok {
				printJSON(cmd, status)
			}
		}
	}
}

// waitReady blocks until the session reports a ready URL or the timeout passes.
// The subscription is set up before the deadline starts so a URL printed early
// is never missed.
func waitReady(cmd *cobra.Command, res types.StatusEvent, timeout time.Duration) error {
	if readyURLs(res) != "" {
		printSession(cmd, res)
		return nil
	}
	if conn == nil {
		return api.Errf(api.KindUnavailable, "not connected")
	}
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer stop()

	events, err := conn.SubscribeRaw(api.SubscribeInput{
		Types: []api.EventType{api.EventRunner},
		Apps:  []int64{res.AppID},
	})
	if err != nil {
		return err
	}
	deadline := time.After(timeout)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-deadline:
			return api.Errf(api.KindConflict, "no ready URL after %s; check 'wm logs' or 'wm status'", timeout)
		case <-conn.Done():
			return api.Errf(api.KindUnavailable, "the connection to the server closed")
		case raw := <-events:
			status, ok := parseRunnerStatus(raw)
			if !ok || readyURLs(status) == "" {
				continue
			}
			printSession(cmd, status)
			return nil
		}
	}
}

// readyURLs returns the first detected URL of a status event.
func readyURLs(ev types.StatusEvent) string {
	for _, p := range ev.Processes {
		if len(p.URLs) > 0 {
			return p.URLs[0]
		}
	}
	return ""
}

// printSession prints a status event, with detected URLs in human mode.
func printSession(cmd *cobra.Command, ev types.StatusEvent) {
	if jsonOutput {
		printJSON(cmd, ev)
		return
	}
	if ev.Error != "" {
		fmt.Fprintln(cmd.OutOrStdout(), "error:", ev.Error)
	}
	for _, p := range ev.Processes {
		extra := ""
		if p.ExitCode != nil {
			extra = fmt.Sprintf(" exit=%d", *p.ExitCode)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%s%s\n", p.CommandID, p.Label, p.Status, extra)
		for _, u := range p.URLs {
			fmt.Fprintf(cmd.OutOrStdout(), "  url: %s\n", u)
		}
	}
}

func runnerStatus(appID int64) (types.StatusEvent, error) {
	var out types.StatusEvent
	if err := call(api.MethodRunnerStatus, map[string]any{"appID": appID}, &out); err != nil {
		return types.StatusEvent{}, err
	}
	return out, nil
}

func checkKind(kind string) error {
	switch kind {
	case types.KindRun, types.KindBuild, types.KindSetup:
		return nil
	}
	return usage("kind must be run, build or setup")
}

// pickArg prefers the positional argument over the flag.
func pickArg(args []string, flag string) string {
	if len(args) > 0 && args[0] != "" {
		return args[0]
	}
	return flag
}

// itoa64 formats an id for display.
func itoa64(n int64) string { return strconv.FormatInt(n, 10) }
