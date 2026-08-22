package services

import (
	"context"
	"fmt"
	"strings"

	"wails_backend/internal/db"
	"wails_backend/internal/lib"
	"wails_backend/internal/types"

	"github.com/firebase/genkit/go/ai"
)

const appAIAgentSystem = `You are a configuration assistant for Workspace Manager.
You edit ONLY the currently selected config set of the current app.

Use tools to inspect and change this set:
- env vars: list_vars, get_var, update_var, delete_var
- templates: list_templates, get_template, update_template (Handlebars {{VAR_NAME}} for env placeholders)
- run config: get_run_config, update_run_config
- project files: search_files (glob; gitignored omitted), read_file (relative path)

You must NOT:
- create, rename, or delete config sets
- invent template file paths; only update templates list_templates returns
- reformat template files unless the user asked

When updating run commands, send the full command list.
Edits are staged for the user to review. Reply in short markdown (lists, inline code). No JSON. No headings.`

type emptyIn struct{}

type keyIn struct {
	Key string `json:"key"`
}

type updateVarIn struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type pathIn struct {
	FilePath string `json:"file_path"`
}

type updateTemplateIn struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

type runCmdIn struct {
	Label   *string `json:"label"`
	Command string  `json:"command"`
}

type updateRunIn struct {
	Mode     *string    `json:"mode,omitempty"`
	Commands []runCmdIn `json:"commands,omitempty"`
}

type searchIn struct {
	Pattern string `json:"pattern"`
}

type filePathIn struct {
	Path string `json:"path"`
}

type envPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type runCmd struct {
	Label   *string `json:"label"`
	Command string  `json:"command"`
}

type appAIState struct {
	projectPath string
	env         map[string]string
	origEnv     map[string]string
	deletedEnv  map[string]bool
	templates   map[string]string
	origTmpl    map[string]string
	runMode     string
	runCommands []runCmd
	runDirty    bool
	calls       []types.AppAIToolCall
	emit        func(types.AppAIStreamEvent)
}

func (s *appAIState) record(name string, in, out any) any {
	call := types.AppAIToolCall{Name: name, Input: in, Output: out}
	s.calls = append(s.calls, call)
	if s.emit != nil {
		c := call
		s.emit(types.AppAIStreamEvent{Type: "tool", Call: &c})
	}
	return out
}

func newAppAIState(detail types.ConfigSetDetail, projectPath string) *appAIState {
	s := &appAIState{
		projectPath: projectPath,
		env:         map[string]string{},
		origEnv:     map[string]string{},
		deletedEnv:  map[string]bool{},
		templates:   map[string]string{},
		origTmpl:    map[string]string{},
		runMode:     "parallel",
	}
	for _, v := range detail.EnvVars {
		s.env[v.Key] = v.Value
		s.origEnv[v.Key] = v.Value
	}
	for _, t := range detail.Templates {
		s.templates[t.FilePath] = t.Content
		s.origTmpl[t.FilePath] = t.Content
	}
	if detail.RunConfig != nil {
		s.runMode = detail.RunConfig.Mode
		for _, c := range detail.RunConfig.Commands {
			label := c.Label
			s.runCommands = append(s.runCommands, runCmd{Label: label, Command: c.Command})
		}
	}
	return s
}

func (s *appAIState) patch() types.AppAIPatch {
	p := types.AppAIPatch{}
	var upsert []types.AppAIEnvUpsert
	for k, v := range s.env {
		orig, ok := s.origEnv[k]
		if !ok || orig != v {
			upsert = append(upsert, types.AppAIEnvUpsert{Key: k, Value: v})
		}
	}
	var del []string
	for k := range s.deletedEnv {
		del = append(del, k)
	}
	if len(upsert) > 0 || len(del) > 0 {
		p.Env = &types.AppAIEnvPatch{Upsert: upsert, Delete: del}
	}
	for path, content := range s.templates {
		orig, ok := s.origTmpl[path]
		if !ok || orig != content {
			p.Templates = append(p.Templates, types.AppAITemplatePatch{FilePath: path, Content: content})
		}
	}
	if s.runDirty {
		cmds := make([]types.AppAIRunCommand, 0, len(s.runCommands))
		for _, c := range s.runCommands {
			cmds = append(cmds, types.AppAIRunCommand{Label: c.Label, Command: c.Command})
		}
		mode := s.runMode
		p.Run = &types.AppAIRunPatch{Mode: &mode, Commands: cmds}
	}
	return p
}

func (s *appAIState) listVars() map[string]any {
	vars := make([]envPair, 0, len(s.env))
	for k, v := range s.env {
		vars = append(vars, envPair{Key: k, Value: v})
	}
	return map[string]any{"vars": vars}
}

func (s *appAIState) getVar(key string) map[string]any {
	k := strings.TrimSpace(key)
	if k == "" {
		return map[string]any{"error": "key is required"}
	}
	v, ok := s.env[k]
	if !ok {
		return map[string]any{"error": "env var not found: " + k}
	}
	return map[string]any{"key": k, "value": v}
}

func (s *appAIState) updateVar(key, value string) map[string]any {
	k := strings.TrimSpace(key)
	if k == "" {
		return map[string]any{"error": "key is required"}
	}
	delete(s.deletedEnv, k)
	s.env[k] = value
	return map[string]any{"ok": true, "key": k, "value": value}
}

func (s *appAIState) deleteVar(key string) map[string]any {
	k := strings.TrimSpace(key)
	if k == "" {
		return map[string]any{"error": "key is required"}
	}
	_, live := s.env[k]
	_, orig := s.origEnv[k]
	if !live && !orig {
		return map[string]any{"error": "env var not found: " + k}
	}
	delete(s.env, k)
	if orig {
		s.deletedEnv[k] = true
	}
	return map[string]any{"ok": true, "key": k}
}

func (s *appAIState) listTemplates() map[string]any {
	paths := make([]string, 0, len(s.templates))
	for p := range s.templates {
		paths = append(paths, p)
	}
	return map[string]any{"file_paths": paths}
}

func (s *appAIState) getTemplate(filePath string) map[string]any {
	p := strings.ReplaceAll(strings.TrimSpace(filePath), "\\", "/")
	if p == "" {
		return map[string]any{"error": "file_path is required"}
	}
	c, ok := s.templates[p]
	if !ok {
		return map[string]any{"error": "template not found: " + p}
	}
	return map[string]any{"file_path": p, "content": c}
}

func (s *appAIState) updateTemplate(filePath, content string) map[string]any {
	p := strings.ReplaceAll(strings.TrimSpace(filePath), "\\", "/")
	if p == "" {
		return map[string]any{"error": "file_path is required"}
	}
	if _, ok := s.templates[p]; !ok {
		return map[string]any{"error": "template not on this config set: " + p}
	}
	s.templates[p] = content
	return map[string]any{"ok": true, "file_path": p}
}

func (s *appAIState) getRun() map[string]any {
	return map[string]any{"mode": s.runMode, "commands": s.runCommands}
}

func (s *appAIState) updateRun(in updateRunIn) map[string]any {
	if in.Mode != nil && *in.Mode != "" {
		m := *in.Mode
		if m != "parallel" && m != "sequential" {
			return map[string]any{"error": `mode must be "parallel" or "sequential"`}
		}
		s.runMode = m
		s.runDirty = true
	}
	if in.Commands != nil {
		cmds := make([]runCmd, 0, len(in.Commands))
		for _, c := range in.Commands {
			command := strings.TrimSpace(c.Command)
			if command == "" {
				continue
			}
			var label *string
			if c.Label != nil {
				l := strings.TrimSpace(*c.Label)
				if l != "" {
					label = &l
				}
			}
			cmds = append(cmds, runCmd{Label: label, Command: command})
		}
		s.runCommands = cmds
		s.runDirty = true
	}
	if !s.runDirty {
		return map[string]any{"error": "provide mode and/or commands"}
	}
	return map[string]any{"ok": true, "mode": s.runMode, "commands": s.runCommands}
}

func (s *appAIState) searchFiles(pattern string) map[string]any {
	files, truncated, errMsg := lib.SearchProjectFiles(s.projectPath, pattern)
	if errMsg != "" {
		return map[string]any{"error": errMsg}
	}
	return map[string]any{"files": files, "truncated": truncated}
}

func (s *appAIState) readFile(path string) map[string]any {
	content, rel, errMsg := lib.ReadProjectFile(s.projectPath, path)
	if errMsg != "" {
		return map[string]any{"error": errMsg}
	}
	return map[string]any{"file_path": rel, "content": content}
}

func (s *appAIState) tools() []ai.ToolRef {
	return []ai.ToolRef{
		ai.NewTool("list_vars", "List env var keys and values on the active config set.",
			func(_ *ai.ToolContext, in emptyIn) (any, error) {
				return s.record("list_vars", in, s.listVars()), nil
			}),
		ai.NewTool("get_var", "Get one env var by key.",
			func(_ *ai.ToolContext, in keyIn) (any, error) {
				return s.record("get_var", in, s.getVar(in.Key)), nil
			}),
		ai.NewTool("update_var", "Create or update an env var on the active config set.",
			func(_ *ai.ToolContext, in updateVarIn) (any, error) {
				return s.record("update_var", in, s.updateVar(in.Key, in.Value)), nil
			}),
		ai.NewTool("delete_var", "Delete an env var from the active config set.",
			func(_ *ai.ToolContext, in keyIn) (any, error) {
				return s.record("delete_var", in, s.deleteVar(in.Key)), nil
			}),
		ai.NewTool("list_templates", "List template file paths on the active config set (no content).",
			func(_ *ai.ToolContext, in emptyIn) (any, error) {
				return s.record("list_templates", in, s.listTemplates()), nil
			}),
		ai.NewTool("get_template", "Get the full content of one template by file_path.",
			func(_ *ai.ToolContext, in pathIn) (any, error) {
				return s.record("get_template", in, s.getTemplate(in.FilePath)), nil
			}),
		ai.NewTool("update_template", "Replace the content of an existing template. file_path must already be on this config set.",
			func(_ *ai.ToolContext, in updateTemplateIn) (any, error) {
				return s.record("update_template", in, s.updateTemplate(in.FilePath, in.Content)), nil
			}),
		ai.NewTool("get_run_config", "Get run mode and commands for the active config set.",
			func(_ *ai.ToolContext, in emptyIn) (any, error) {
				return s.record("get_run_config", in, s.getRun()), nil
			}),
		ai.NewTool("update_run_config", "Update run mode and/or replace the full command list for this config set.",
			func(_ *ai.ToolContext, in updateRunIn) (any, error) {
				return s.record("update_run_config", in, s.updateRun(in)), nil
			}),
		ai.NewTool("search_files", "Glob search under the app project directory. Respects .gitignore. Returns matching relative paths.",
			func(_ *ai.ToolContext, in searchIn) (any, error) {
				return s.record("search_files", in, s.searchFiles(in.Pattern)), nil
			}),
		ai.NewTool("read_file", "Read a text file under the app project directory. Path must be relative and stay inside the project.",
			func(_ *ai.ToolContext, in filePathIn) (any, error) {
				return s.record("read_file", in, s.readFile(in.Path)), nil
			}),
	}
}

func buildAppAIAgentPrompt(appName, projectPath, setName string, setID int64, history []types.AppAIChatTurn, instruction string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "App: %s\nProject path: %s\n\nActive config set (ONLY edit this one):\nid: %d\nname: %s\n\n", appName, projectPath, setID, setName)
	b.WriteString("Use tools to read or edit env vars, templates, and run config. Use search_files and read_file when you need project files.\n\nPrior conversation:\n")
	if len(history) == 0 {
		b.WriteString("(none)\n")
	} else {
		for _, t := range history {
			who := "Assistant"
			if t.Role == "user" {
				who = "User"
			}
			fmt.Fprintf(&b, "%s: %s\n", who, t.Text)
		}
	}
	fmt.Fprintf(&b, "\nUser instruction:\n%s", strings.TrimSpace(instruction))
	return b.String()
}

func patchHasEdits(p types.AppAIPatch) bool {
	if p.Env != nil && (len(p.Env.Upsert) > 0 || len(p.Env.Delete) > 0) {
		return true
	}
	if len(p.Templates) > 0 {
		return true
	}
	return p.Run != nil
}

// AppChatAI runs the config-set agent with tools. Updates are staged in the
// returned patch for the UI to review; nothing is written until the user applies.
func AppChatAI(ctx context.Context, d *db.DB, appID, setID int64, history []types.AppAIChatTurn, instruction string, emit func(types.AppAIStreamEvent)) (types.AppAIChatResult, error) {
	instruction = strings.TrimSpace(instruction)
	if instruction == "" {
		return types.AppAIChatResult{}, fmt.Errorf("instruction is required")
	}
	app, err := d.GetAppT(ctx, appID)
	if err != nil {
		return types.AppAIChatResult{}, err
	}
	set, err := d.GetConfigSetT(ctx, setID)
	if err != nil {
		return types.AppAIChatResult{}, err
	}
	if set.AppID != app.ID {
		return types.AppAIChatResult{}, fmt.Errorf("Config set not found")
	}
	detail, err := loadConfigSetDetail(ctx, d, set)
	if err != nil {
		return types.AppAIChatResult{}, err
	}
	state := newAppAIState(detail, app.ProjectPath)
	state.emit = emit

	store, err := LoadAIStore()
	if err != nil {
		return types.AppAIChatResult{}, err
	}
	_, conn, err := store.ActiveAIConnection()
	if err != nil {
		return types.AppAIChatResult{}, err
	}

	prompt := buildAppAIAgentPrompt(app.Name, app.ProjectPath, set.Name, set.ID, history, instruction)
	var onText func(string)
	if emit != nil {
		onText = func(t string) {
			emit(types.AppAIStreamEvent{Type: "text", Text: t})
		}
	}
	text, err := runGeneration(ctx, conn.Provider, conn, appAIAgentSystem, prompt, state.tools(), onText)
	if err != nil {
		return types.AppAIChatResult{}, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		text = "Done."
	}
	patch := state.patch()
	patch.Message = text
	if !patchHasEdits(patch) {
		return types.AppAIChatResult{
			Text:      text,
			Patch:     types.AppAIPatch{Message: text},
			ToolCalls: state.calls,
		}, nil
	}
	return types.AppAIChatResult{Text: text, Patch: patch, ToolCalls: state.calls}, nil
}

func loadConfigSetDetail(ctx context.Context, d *db.DB, set types.ConfigSet) (types.ConfigSetDetail, error) {
	envVars, err := d.ListEnvVarsT(ctx, set.ID)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	templates, err := d.ListTemplatesT(ctx, set.ID)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	runCfg, err := d.GetRunConfigByConfigSetT(ctx, set.ID)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	return types.ConfigSetDetail{ConfigSet: set, EnvVars: envVars, Templates: templates, RunConfig: runCfg}, nil
}
