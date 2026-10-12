package main

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"workspace-manager/api"
	"workspace-manager/api/types"
)

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "template", Short: "Manage an app's file templates"}
	cmd.AddCommand(newTemplateListCmd(), newTemplateAddCmd(), newTemplateEditCmd(), newTemplateRmCmd())
	return cmd
}

func newTemplateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [app]",
		Short: "List templates on the active config set",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(args)
			if err != nil {
				return err
			}
			var out []types.Template
			if err := call(api.MethodTemplatesList, map[string]any{"appID": app.ID}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

func newTemplateAddCmd() *cobra.Command {
	var fromFile, content string
	var hidden bool
	c := &cobra.Command{
		Use:   "add <file> [app]",
		Short: "Add a template for a file in the project",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			include := !hidden
			in := types.TemplateCreateInput{
				FilePath:    strings.ReplaceAll(strings.TrimSpace(args[0]), "\\", "/"),
				IncludeInAI: &include,
			}
			switch {
			case fromFile != "":
				data, err := os.ReadFile(fromFile)
				if err != nil {
					return api.Errf(api.KindNotFound, "cannot read %s", fromFile)
				}
				s := string(data)
				in.Content = &s
			case cmd.Flags().Changed("content"):
				in.Content = &content
			}
			var out types.Template
			if err := call(api.MethodTemplatesCreate, map[string]any{"appID": app.ID, "body": in}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&fromFile, "from-file", "", "read the content from this file")
	c.Flags().StringVar(&content, "content", "", "template content")
	c.Flags().BoolVar(&hidden, "hidden", false, "hide the content from AI clients")
	return c
}

func newTemplateEditCmd() *cobra.Command {
	var content, fromFile string
	var hidden bool
	c := &cobra.Command{
		Use:   "edit <file> [app]",
		Short: "Change a template's content",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			tpl, err := templateOf(app, args[0])
			if err != nil {
				return err
			}
			in := types.TemplateUpdateInput{}
			switch {
			case fromFile != "":
				data, err := os.ReadFile(fromFile)
				if err != nil {
					return api.Errf(api.KindNotFound, "cannot read %s", fromFile)
				}
				s := string(data)
				in.Content = &s
			case cmd.Flags().Changed("content"):
				in.Content = &content
			}
			if cmd.Flags().Changed("hidden") {
				include := !hidden
				in.IncludeInAI = &include
			}
			var out types.Template
			if err := call(api.MethodTemplatesUpdate, map[string]any{"id": tpl.ID, "body": in}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
	c.Flags().StringVar(&fromFile, "from-file", "", "read the new content from this file")
	c.Flags().StringVar(&content, "content", "", "new content")
	c.Flags().BoolVar(&hidden, "hidden", false, "hide the content from AI clients")
	return c
}

func newTemplateRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <file> [app]",
		Short: "Delete a template",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := appFromArgs(rest(args))
			if err != nil {
				return err
			}
			tpl, err := templateOf(app, args[0])
			if err != nil {
				return err
			}
			if err := confirm(cmd, "Delete template "+tpl.FilePath); err != nil {
				return err
			}
			var out types.Ok
			if err := call(api.MethodTemplatesDelete, map[string]any{"id": tpl.ID}, &out); err != nil {
				return err
			}
			printJSON(cmd, out)
			return nil
		},
	}
}

// templateOf finds a template by its file path.
func templateOf(app types.App, file string) (types.Template, error) {
	want := strings.ReplaceAll(strings.TrimSpace(file), "\\", "/")
	var out []types.Template
	if err := call(api.MethodTemplatesList, map[string]any{"appID": app.ID}, &out); err != nil {
		return types.Template{}, err
	}
	for _, t := range out {
		if strings.EqualFold(t.FilePath, want) {
			return t, nil
		}
	}
	return types.Template{}, api.Errf(api.KindNotFound, "no template for %q on app %s", file, app.Name)
}
