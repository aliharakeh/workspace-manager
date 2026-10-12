package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"workspace-manager/api"
	"workspace-manager/api/types"
	
)

func newBlueprintCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "blueprint", Short: "Manage blueprints: the recipes that create new apps"}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List blueprints",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				var out []types.Blueprint
				if err := call(api.MethodBlueprintsList, nil, &out); err != nil {
					return err
				}
				printJSON(cmd, out)
				return nil
			},
		},
		&cobra.Command{
			Use:   "show <id|name>",
			Short: "Show one blueprint",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				bp, err := blueprintByName(args[0])
				if err != nil {
					return err
				}
				printJSON(cmd, bp)
				return nil
			},
		},
		func() *cobra.Command {
			var description, sampleName string
			var command []string
			var label []string
			var noFolder bool
			c := &cobra.Command{
				Use:   "add <name>",
				Short: "Create a blueprint",
				Args:  cobra.ExactArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					in, err := blueprintInput(args[0], description, sampleName, !noFolder, command, label)
					if err != nil {
						return err
					}
					var out types.Blueprint
					if err := call(api.MethodBlueprintsCreate, in, &out); err != nil {
						return err
					}
					printJSON(cmd, out)
					return nil
				},
			}
			c.Flags().StringVar(&description, "description", "", "one sentence about what it creates")
			c.Flags().StringVar(&sampleName, "sample-name", "", "example app name for the new-app dialog")
			c.Flags().StringArrayVar(&command, "command", nil, "a shell line, in order")
			c.Flags().StringArrayVar(&label, "label", nil, "a label for the matching --command")
			c.Flags().BoolVar(&noFolder, "no-create-folder", false, "commands run in the parent folder and create the app folder themselves")
			return c
		}(),
		func() *cobra.Command {
			var description, sampleName string
			var command []string
			var label []string
			var noFolder, folderSet bool
			c := &cobra.Command{
				Use:   "edit <id|name>",
				Short: "Replace a blueprint's fields",
				Args:  cobra.ExactArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					bp, err := blueprintByName(args[0])
					if err != nil {
						return err
					}
					if !cmd.Flags().Changed("description") {
						description = bp.Description
					}
					if !cmd.Flags().Changed("sample-name") {
						sampleName = bp.SampleName
					}
					createFolder := bp.CreateFolder
					if cmd.Flags().Changed("no-create-folder") {
						createFolder = !noFolder
						folderSet = true
					}
					_ = folderSet
					cmds := command
					labels := label
					if len(cmds) == 0 {
						for _, bc := range bp.Commands {
							cmds = append(cmds, bc.Command)
							if bc.Label != nil {
								labels = append(labels, *bc.Label)
							} else {
								labels = append(labels, "")
							}
						}
					}
					in, err := blueprintInput(bp.Name, description, sampleName, createFolder, cmds, labels)
					if err != nil {
						return err
					}
					var out types.Blueprint
					if err := call(api.MethodBlueprintsUpdate, map[string]any{"id": bp.ID, "body": in}, &out); err != nil {
						return err
					}
					printJSON(cmd, out)
					return nil
				},
			}
			c.Flags().StringVar(&description, "description", "", "new description")
			c.Flags().StringVar(&sampleName, "sample-name", "", "new sample app name")
			c.Flags().StringArrayVar(&command, "command", nil, "replace the commands with these, in order")
			c.Flags().StringArrayVar(&label, "label", nil, "a label for the matching --command")
			c.Flags().BoolVar(&noFolder, "no-create-folder", false, "commands create the app folder themselves")
			return c
		}(),
		newBlueprintRenderCmd(),
		newBlueprintNewCmd(),
	)
	return cmd
}

// newBlueprintRenderCmd previews the commands a blueprint would run.
func newBlueprintRenderCmd() *cobra.Command {
	var name, parent, folder string
	c := &cobra.Command{
		Use:   "render <id|name>",
		Short: "Show the commands a blueprint would run, without running them",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bp, err := blueprintByName(args[0])
			if err != nil {
				return err
			}
			if name == "" {
				name = bp.SampleName
			}
			if name == "" {
				return usage("--name is required (the blueprint has no sample name)")
			}
			if folder == "" {
				folder = slugify(name)
			}
			rendered := renderBlueprint(bp, name, folder, parent)
			if jsonOutput {
				printJSON(cmd, rendered)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "cwd: %s\n", rendered.Cwd)
			for _, line := range rendered.Commands {
				fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "app name (defaults to the blueprint's sample name)")
	c.Flags().StringVar(&parent, "parent", "", "parent folder")
	c.Flags().StringVar(&folder, "folder", "", "folder name (defaults to a slug of the app name)")
	return c
}

// blueprintInput builds the API input from the flag values.
func blueprintInput(name, description, sampleName string, createFolder bool, commands, labels []string) (types.BlueprintInput, error) {
	in := types.BlueprintInput{
		Name:         strings.TrimSpace(name),
		Description:  strings.TrimSpace(description),
		SampleName:   strings.TrimSpace(sampleName),
		CreateFolder: createFolder,
	}
	for i, line := range commands {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		bc := types.BlueprintCommand{Command: line}
		if i < len(labels) && strings.TrimSpace(labels[i]) != "" {
			l := strings.TrimSpace(labels[i])
			bc.Label = &l
		}
		in.Commands = append(in.Commands, bc)
	}
	return in, nil
}
// blueprintByName looks a blueprint up by id or by name.
func blueprintByName(arg string) (types.Blueprint, error) {
	var all []types.Blueprint
	if err := call(api.MethodBlueprintsList, nil, &all); err != nil {
		return types.Blueprint{}, err
	}
	if id, err := strconv.ParseInt(strings.TrimSpace(arg), 10, 64); err == nil {
		for _, bp := range all {
			if bp.ID == id {
				return bp, nil
			}
		}
		return types.Blueprint{}, api.Errf(api.KindNotFound, "no blueprint with id %d", id)
	}
	for _, bp := range all {
		if strings.EqualFold(bp.Name, strings.TrimSpace(arg)) {
			return bp, nil
		}
	}
	return types.Blueprint{}, api.Errf(api.KindNotFound, "no blueprint named %q", arg)
}

// slugify makes a folder-safe name from an app name.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// newBlueprintNewCmd creates an app from a blueprint, streaming its log.
func newBlueprintNewCmd() *cobra.Command {
	var opts struct {
		name        string
		parent      string
		folder      string
		workspace   string
		onError     string
		createDir   bool
		configSetID int64
	}
	c := &cobra.Command{
		Use:   "new <id|name>",
		Short: "Create an app from a blueprint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bp, err := blueprintByName(args[0])
			if err != nil {
				return err
			}
			ws, err := resolveWorkspace(opts.workspace)
			if err != nil {
				return err
			}
			if strings.TrimSpace(opts.parent) == "" {
				return usage("--parent is required")
			}
			if abs, err := filepath.Abs(opts.parent); err == nil {
				opts.parent = abs
			}
			name := opts.name
			if name == "" {
				name = bp.SampleName
			}
			if strings.TrimSpace(name) == "" {
				return usage("--name is required (the blueprint has no sample name)")
			}
			folder := opts.folder
			if folder == "" {
				folder = slugify(name)
			}
			switch opts.onError {
			case "", "ask", "skip", "abort":
			default:
				return usage("--on-error must be ask, skip or abort")
			}

			runID := fmt.Sprintf("cli-%d", os.Getpid())
			// "ask" and "skip" both pause on a failed command so a decision
			// can be made; "abort" lets the run stop with the error.
			in := types.BlueprintRunInput{
				RunID:        runID,
				BlueprintID:  bp.ID,
				WorkspaceID:  ws.ID,
				Name:         name,
				ParentPath:   opts.parent,
				FolderName:   folder,
				CreateFolder: bp.CreateFolder,
				AskOnError:   opts.onError != "abort",
			}
			return runBlueprint(cmd, in, opts.onError)
		},
	}
	c.Flags().StringVar(&opts.name, "name", "", "app name")
	c.Flags().StringVar(&opts.parent, "parent", "", "parent folder for the new app")
	c.Flags().StringVar(&opts.folder, "folder", "", "folder name (defaults to a slug of the app name)")
	c.Flags().StringVar(&opts.workspace, "workspace", "", "workspace id or name")
	c.Flags().StringVar(&opts.onError, "on-error", "abort", "what to do when a command fails: ask, skip or abort")
	return c
}

// runBlueprint streams the run's log, forwards input, and prints the created
// app at the end.
func runBlueprint(cmd *cobra.Command, in types.BlueprintRunInput, onError string) error {
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer stop()

	go func() {
		errCh := make(chan error, 1)
		errCh <- conn.Subscribe(api.SubscribeInput{
			Types: []api.EventType{api.EventBlueprint},
		}, func(ev api.Event) {
			raw, err := json.Marshal(ev.Data)
			if err != nil {
				return
			}
			var payload types.BlueprintLogEvent
			if err := json.Unmarshal(raw, &payload); err != nil {
				return
			}
			if payload.RunID != in.RunID {
				return
			}
			switch payload.Stream {
			case "data":
				data, err := base64.StdEncoding.DecodeString(payload.Data)
				if err != nil {
					return
				}
				_, _ = cmd.OutOrStdout().Write(data)
			case "failed":
				fmt.Fprintln(cmd.OutOrStdout(), payload.Text)
				resolve := "abort"
				if onError == "skip" {
					resolve = "skip"
				} else if onError == "ask" {
					fmt.Fprint(cmd.OutOrStdout(), "skipped? [y/N] ")
					var answer string
					_, _ = fmt.Fscanln(cmd.InOrStdin(), &answer)
					if strings.ToLower(strings.TrimSpace(answer)) == "y" {
						resolve = "skip"
					}
				}
				_ = call(api.MethodBlueprintsResolve, map[string]any{"runID": in.RunID, "action": resolve}, nil)
			}
		})
		_ = errCh
	}()

	// Forward typed input to the command's terminal until the run ends.
	if in.AskOnError || isTerminal(cmd.InOrStdin()) {
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := cmd.InOrStdin().Read(buf)
				if n > 0 {
					_ = call(api.MethodBlueprintsSendInput, map[string]any{
						"runID": in.RunID, "data": string(buf[:n]),
					}, nil)
				}
				if err != nil {
					return
				}
			}
		}()
	}

	type result struct {
		res types.BlueprintRunResult
		err error
	}
	done := make(chan result, 1)
	go func() {
		var res types.BlueprintRunResult
		err := call(api.MethodBlueprintsCreateApp, in, &res)
		done <- result{res: res, err: err}
	}()

	select {
	case <-ctx.Done():
		_ = call(api.MethodBlueprintsCancel, map[string]any{"runID": in.RunID}, nil)
		return api.Errf(api.KindConflict, "cancelled")
	case r := <-done:
		if r.err != nil {
			return r.err
		}
		if r.res.Warning != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), "warning:", r.res.Warning)
		}
		printJSON(cmd, r.res)
		return nil
	}
}

func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
