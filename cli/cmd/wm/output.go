package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"workspace-manager/api/types"
)

// printJSON writes the result as JSON, or a short human-readable form.
func printJSON(cmd *cobra.Command, v any) {
	if !jsonOutput {
		writeHuman(cmd, v)
		return
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "wm:", err)
	}
}

func writeHuman(cmd *cobra.Command, v any) {
	switch t := v.(type) {
	case nil:
		return
	case string:
		if t != "" {
			fmt.Fprintln(cmd.OutOrStdout(), t)
		}
	case types.Ok:
		if t.Ok {
			fmt.Fprintln(cmd.OutOrStdout(), "ok")
		}
	case []types.Workspace:
		for _, w := range t {
			fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\n", w.ID, w.Name)
		}
	case []types.App:
		for _, a := range t {
			fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%s\n", a.ID, a.Name, a.ProjectPath)
		}
	case []types.ConfigSet:
		for _, s := range t {
			fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\n", s.ID, s.Name)
		}
	case []types.EnvVar:
		for _, e := range t {
			fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", e.Key, e.Value)
		}
	case []types.Template:
		for _, tp := range t {
			fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\n", tp.ID, tp.FilePath)
		}
	case []types.Blueprint:
		for _, b := range t {
			fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%s\n", b.ID, b.Name, b.Description)
		}
	case []string:
		for _, s := range t {
			fmt.Fprintln(cmd.OutOrStdout(), s)
		}
	case map[string]string:
		for k, val := range t {
			fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", k, val)
		}
	default:
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(v)
	}
}

// confirm asks before a destructive change unless --yes was given. Without a
// terminal there is nobody to ask, so it refuses and says so.
func confirm(cmd *cobra.Command, what string) error {
	if assumeYes {
		return nil
	}
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return usage("%s needs --yes when there is no terminal to confirm in", what)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s? [y/N] ", what)
	var answer string
	_, _ = fmt.Fscanln(os.Stdin, &answer)
	if strings.ToLower(strings.TrimSpace(answer)) != "y" {
		return fmt.Errorf("cancelled")
	}
	return nil
}
