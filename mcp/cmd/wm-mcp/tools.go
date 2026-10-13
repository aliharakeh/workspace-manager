package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"workspace-manager/api"
)

// tool is one MCP tool: a name, a description and a JSON-Schema input.
type mcpTool struct {
	Name        string
	Description string
	Schema      map[string]any
	Handler     func(args map[string]any) (any, error)
}

// schema builds a JSON-Schema object node.
func schema(desc string, props map[string]any, required ...string) map[string]any {
	s := map[string]any{"type": "object", "description": desc, "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func strProp(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func boolProp(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}
func numProp(desc string) map[string]any { return map[string]any{"type": "number", "description": desc} }
func arrProp(desc, item string) map[string]any {
	return map[string]any{"type": "array", "description": desc, "items": map[string]any{"type": item}}
}

const appRefDesc = "app id or name; omit when there is exactly one app"
const wsRefDesc = "workspace id or name; omit to cover all workspaces (or the only one)"
const setRefDesc = "config set id or name; omit for the app's active set"

func allTools() []mcpTool {
	return []mcpTool{
		overviewTool(),
		appTool(),
		configTool(),
		envTool(),
		templateTool(),
		runTool(),
		gitTool(),
		blueprintTool(),
		portsTool(),
		systemTool(),
		callTool(),
	}
}

func findTool(name string) *mcpTool {
	for _, t := range allTools() {
		if t.Name == name {
			t := t
			return &t
		}
	}
	return nil
}

func toolSchemas() []any {
	out := make([]any, 0, len(allTools()))
	for _, t := range allTools() {
		out = append(out, map[string]any{
			"name": t.Name, "description": t.Description, "inputSchema": t.Schema,
		})
	}
	return out
}

// ---- wm_overview ----

func overviewTool() mcpTool {
	return mcpTool{
		Name:        "wm_overview",
		Description: "List workspaces, their apps and what is currently running. Start here to discover the setup.",
		Schema: schema("Workspace overview", map[string]any{
			"workspace": strProp("optional: " + wsRefDesc),
		}),
		Handler: func(args map[string]any) (any, error) {
			ws, err := allWorkspaces()
			if err != nil {
				return nil, err
			}
			if w, ok := argOpt(args, "workspace"); ok {
				r, err := resolveWorkspace(w)
				if err != nil {
					return nil, err
				}
				filtered := ws[:0:0]
				for _, cand := range ws {
					if id, _ := cand["id"].(float64); int64(id) == r.ID {
						filtered = append(filtered, cand)
					}
				}
				ws = filtered
			}
			type entry struct {
				Workspace map[string]any `json:"workspace"`
				Apps      any            `json:"apps"`
				Status    any            `json:"status"`
			}
			var out []entry
			for _, w := range ws {
				id, _ := w["id"].(float64)
				apps, err := daoCall(api.MethodAppsList, map[string]any{"workspaceID": int64(id)})
				if err != nil {
					return nil, err
				}
				status, err := daoCall(api.MethodRunnerWorkspaceStatus, map[string]any{"workspaceID": int64(id)})
				if err != nil {
					return nil, err
				}
				out = append(out, entry{Workspace: w, Apps: apps, Status: status})
			}
			if out == nil {
				out = []entry{}
			}
			return out, nil
		},
	}
}

// ---- wm_app ----

func appTool() mcpTool {
	return mcpTool{
		Name:        "wm_app",
		Description: "Manage apps: list, inspect (with config sets and runner status), create, rename/move, delete, reorder.",
		Schema: schema("App management", map[string]any{
			"action":    map[string]any{"type": "string", "description": "list, get, create, update, delete, reorder", "enum": []string{"list", "get", "create", "update", "delete", "reorder"}},
			"app":       strProp(appRefDesc),
			"workspace": strProp(wsRefDesc + " (for list/create/reorder)"),
			"name":      strProp("new app name (create/update)"),
			"path":      strProp("project folder (create: required; update: new path)"),
			"ids":       arrProp("ordered app ids or names (reorder)", "string"),
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			switch argStr(args, "action") {
			case "list":
				if w, ok := argOpt(args, "workspace"); ok {
					r, err := resolveWorkspace(w)
					if err != nil {
						return nil, err
					}
					return daoCall(api.MethodAppsList, map[string]any{"workspaceID": r.ID})
				}
				apps, err := allApps()
				if err != nil {
					return nil, err
				}
				if apps == nil {
					apps = []map[string]any{}
				}
				return apps, nil
			case "get":
				app, err := resolveApp(argStr(args, "app"))
				if err != nil {
					return nil, err
				}
				sets, err := daoCall(api.MethodConfigSetsList, map[string]any{"appID": appIDOf(app)})
				if err != nil {
					return nil, err
				}
				var detail any
				if set, err := resolveConfigSet(app, ""); err == nil {
					detail, _ = daoCall(api.MethodConfigSetsDetail, map[string]any{"id": set.ID})
				}
				status, err := daoCall(api.MethodRunnerStatus, map[string]any{"appID": appIDOf(app)})
				if err != nil {
					return nil, err
				}
				return map[string]any{
					"app": app, "configSets": sets,
					"activeConfigSetDetail": detail, "status": status,
				}, nil
			case "create":
				name, err := need(args, "name", "the app name")
				if err != nil {
					return nil, err
				}
				path, err := need(args, "path", "the project folder")
				if err != nil {
					return nil, err
				}
				if abs, err := filepath.Abs(path); err == nil {
					path = abs
				}
				ws, err := resolveWorkspace(argStr(args, "workspace"))
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodAppsCreate, map[string]any{
					"workspaceID": ws.ID,
					"body":        map[string]any{"name": name, "project_path": path},
				})
			case "update":
				app, err := resolveApp(argStr(args, "app"))
				if err != nil {
					return nil, err
				}
				body := map[string]any{}
				if name, ok := argOpt(args, "name"); ok {
					body["name"] = name
				}
				if path, ok := argOpt(args, "path"); ok {
					if abs, err := filepath.Abs(path); err == nil {
						path = abs
					}
					body["project_path"] = path
				}
				if len(body) == 0 {
					return nil, fmt.Errorf("nothing to update: pass \"name\" and/or \"path\"")
				}
				return daoCall(api.MethodAppsUpdate, map[string]any{"id": appIDOf(app), "body": body})
			case "delete":
				app, err := resolveApp(argStr(args, "app"))
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodAppsDelete, map[string]any{"id": appIDOf(app)})
			case "reorder":
				ids := argStrList(args, "ids")
				if len(ids) == 0 {
					return nil, fmt.Errorf("pass \"ids\": the ordered app ids or names")
				}
				var wsID int64
				if w, ok := argOpt(args, "workspace"); ok {
					r, err := resolveWorkspace(w)
					if err != nil {
						return nil, err
					}
					wsID = r.ID
				}
				var out []int64
				for _, ref := range ids {
					a, err := resolveApp(ref)
					if err != nil {
						return nil, err
					}
					if wsID == 0 {
						if id, _ := a["workspace_id"].(float64); id != 0 {
							wsID = int64(id)
						}
					} else if id, _ := a["workspace_id"].(float64); int64(id) != wsID {
						name, _ := a["name"].(string)
						return nil, fmt.Errorf("app %s is not in workspace %d", name, wsID)
					}
					out = append(out, appIDOf(a))
				}
				return daoCall(api.MethodAppsReorder, map[string]any{"workspaceID": wsID, "ids": out})
			}
			return nil, fmt.Errorf("unknown action %q", argStr(args, "action"))
		},
	}
}

// ---- wm_config ----

func configTool() mcpTool {
	return mcpTool{
		Name:        "wm_config",
		Description: "Manage config sets and an app's run/build/setup command lists.",
		Schema: schema("Config sets and command lists", map[string]any{
			"action":    map[string]any{"type": "string", "description": "list, detail, create, rename, activate, copy, delete, get_commands, save_commands, scripts", "enum": []string{"list", "detail", "create", "rename", "activate", "copy", "delete", "get_commands", "save_commands", "scripts"}},
			"app":       strProp(appRefDesc),
			"set":       strProp(setRefDesc),
			"name":      strProp("config set name (create/rename)"),
			"kind":      strProp("run, build or setup (get_commands/save_commands)"),
			"from":      strProp("source config set id or name (create/copy)"),
			"parts":     strProp("comma list: env,templates,run,build,setup (create/copy, default all)"),
			"noActivate": boolProp("do not make the new set active (create)"),
			"mode":      strProp("parallel or sequential (save_commands)"),
			"commands":  map[string]any{"type": "array", "description": "command lines in order (save_commands)", "items": map[string]any{"type": "object", "properties": map[string]any{"command": map[string]any{"type": "string"}, "label": map[string]any{"type": "string"}}}},
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			action := argStr(args, "action")
			if action == "detail" {
				// Fast path: a numeric set id needs no app lookup.
				if id, ok := parseID(argStr(args, "set")); ok && argStr(args, "app") == "" {
					return daoCall(api.MethodConfigSetsDetail, map[string]any{"id": id})
				}
			}
			app, err := resolveApp(argStr(args, "app"))
			if err != nil {
				return nil, err
			}
			appID := appIDOf(app)
			setRef := argStr(args, "set")
			switch action {
			case "list":
				return daoCall(api.MethodConfigSetsList, map[string]any{"appID": appID})
			case "detail":
				set, err := resolveConfigSet(app, setRef)
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodConfigSetsDetail, map[string]any{"id": set.ID})
			case "create":
				name, err := need(args, "name", "the config set name")
				if err != nil {
					return nil, err
				}
				body := map[string]any{"name": name}
				if argBool(args, "noActivate", false) {
					body["activate"] = false
				}
				if from, ok := argOpt(args, "from"); ok {
					src, err := resolveConfigSet(app, from)
					if err != nil {
						return nil, err
					}
					body["copy_from_id"] = src.ID
					body["parts"] = copyParts(argStr(args, "parts"))
				}
				return daoCall(api.MethodConfigSetsCreate, map[string]any{"appID": appID, "body": body})
			case "rename":
				name, err := need(args, "name", "the new name")
				if err != nil {
					return nil, err
				}
				set, err := resolveConfigSet(app, setRef)
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodConfigSetsUpdate, map[string]any{"id": set.ID, "body": map[string]any{"name": name}})
			case "activate":
				set, err := resolveConfigSet(app, setRef)
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodConfigSetsActivate, map[string]any{"id": set.ID})
			case "copy":
				set, err := resolveConfigSet(app, setRef)
				if err != nil {
					return nil, err
				}
				from, err := need(args, "from", "the source config set")
				if err != nil {
					return nil, err
				}
				src, err := resolveConfigSet(app, from)
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodConfigSetsCopyFrom, map[string]any{
					"id": set.ID, "sourceID": src.ID, "parts": copyParts(argStr(args, "parts")),
				})
			case "delete":
				set, err := resolveConfigSet(app, setRef)
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodConfigSetsDelete, map[string]any{"id": set.ID})
			case "get_commands":
				kind, err := need(args, "kind", "the command list kind")
				if err != nil {
					return nil, err
				}
				if err := checkKind(kind); err != nil {
					return nil, err
				}
				return daoCall(api.MethodRunConfigGet, map[string]any{"appID": appID, "kind": kind})
			case "save_commands":
				kind, err := need(args, "kind", "the command list kind")
				if err != nil {
					return nil, err
				}
				if err := checkKind(kind); err != nil {
					return nil, err
				}
				var cmds []any
				for _, c := range argList(args, "commands") {
					m, ok := c.(map[string]any)
					if !ok {
						continue
					}
					cmd, _ := m["command"].(string)
					if cmd == "" {
						continue
					}
					entry := map[string]any{"command": cmd}
					if label, _ := m["label"].(string); label != "" {
						entry["label"] = label
					}
					cmds = append(cmds, entry)
				}
				if len(cmds) == 0 {
					return nil, fmt.Errorf("pass \"commands\": at least one {command, label?}")
				}
				body := map[string]any{"commands": cmds}
				if mode, ok := argOpt(args, "mode"); ok {
					body["mode"] = mode
				}
				return daoCall(api.MethodRunConfigSave, map[string]any{"appID": appID, "kind": kind, "body": body})
			case "scripts":
				return daoCall(api.MethodPackageScriptsList, map[string]any{"appID": appID})
			}
			return nil, fmt.Errorf("unknown action %q", action)
		},
	}
}

func copyParts(list string) map[string]any {
	if list == "" {
		list = "env,templates,run,build,setup"
	}
	parts := map[string]any{}
	for _, p := range splitComma(list) {
		switch p {
		case "env", "templates", "run", "build", "setup":
			parts[p] = true
		}
	}
	return parts
}

func checkKind(kind string) error {
	switch kind {
	case "run", "build", "setup":
		return nil
	}
	return fmt.Errorf("kind must be run, build or setup, got %q", kind)
}

func splitComma(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			for len(part) > 0 && (part[0] == ' ' || part[0] == '\t') {
				part = part[1:]
			}
			for len(part) > 0 && (part[len(part)-1] == ' ' || part[len(part)-1] == '\t') {
				part = part[:len(part)-1]
			}
			if part != "" {
				out = append(out, part)
			}
			start = i + 1
		}
	}
	return out
}

// ---- wm_env ----

func envTool() mcpTool {
	return mcpTool{
		Name:        "wm_env",
		Description: "Manage an app's environment variables on its active config set. Values with include_in_ai=false are hidden from this MCP server.",
		Schema: schema("Environment variables", map[string]any{
			"action":    map[string]any{"type": "string", "description": "list, set (create or update), delete, import", "enum": []string{"list", "set", "delete", "import"}},
			"app":       strProp(appRefDesc),
			"configSet": strProp("config set id or name to list (list only; writes go to the active set)"),
			"key":       strProp("variable name (set/delete)"),
			"value":     strProp("variable value (set; omit to keep the stored value on update)"),
			"hidden":    boolProp("hide the value from AI clients, i.e. include_in_ai=false (set)"),
			"file":      strProp(".env or YAML file to import (import)"),
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			app, err := resolveApp(argStr(args, "app"))
			if err != nil {
				return nil, err
			}
			appID := appIDOf(app)
			switch argStr(args, "action") {
			case "list":
				if cs, ok := argOpt(args, "configSet"); ok {
					set, err := resolveConfigSet(app, cs)
					if err != nil {
						return nil, err
					}
					detail, err := daoCall(api.MethodConfigSetsDetail, map[string]any{"id": set.ID})
					if err != nil {
						return nil, err
					}
					if m, ok := detail.(map[string]any); ok {
						return m["env_vars"], nil
					}
					return detail, nil
				}
				return daoCall(api.MethodEnvVarsList, map[string]any{"appID": appID})
			case "set":
				key, err := need(args, "key", "the variable name")
				if err != nil {
					return nil, err
				}
				include := !argBool(args, "hidden", false)
				vars, err := daoCall(api.MethodEnvVarsList, map[string]any{"appID": appID})
				if err != nil {
					return nil, err
				}
				raw, _ := json.Marshal(vars)
				var list []map[string]any
				_ = json.Unmarshal(raw, &list)
				_, hasValue := args["value"]
				value, _ := args["value"].(string)
				for _, v := range list {
					if v["key"] == key {
						id, _ := v["id"].(float64)
						body := map[string]any{"key": key, "include_in_ai": include}
						if hasValue {
							body["value"] = value
						}
						return daoCall(api.MethodEnvVarsUpdate, map[string]any{"id": int64(id), "body": body})
					}
				}
				body := map[string]any{"key": key, "include_in_ai": include}
				if hasValue {
					body["value"] = value
				}
				return daoCall(api.MethodEnvVarsCreate, map[string]any{"appID": appID, "body": body})
			case "delete":
				key, err := need(args, "key", "the variable name")
				if err != nil {
					return nil, err
				}
				vars, err := daoCall(api.MethodEnvVarsList, map[string]any{"appID": appID})
				if err != nil {
					return nil, err
				}
				raw, _ := json.Marshal(vars)
				var list []map[string]any
				_ = json.Unmarshal(raw, &list)
				for _, v := range list {
					if v["key"] == key {
						id, _ := v["id"].(float64)
						return daoCall(api.MethodEnvVarsDelete, map[string]any{"id": int64(id)})
					}
				}
				return nil, api.Errf(api.KindNotFound, "no env var named %q on this app", key)
			case "import":
				file, err := need(args, "file", "the file to import")
				if err != nil {
					return nil, err
				}
				if abs, err := filepath.Abs(file); err == nil {
					file = abs
				}
				if _, err := os.Stat(file); err != nil {
					return nil, api.Errf(api.KindNotFound, "cannot read %s", file)
				}
				return daoCall(api.MethodEnvVarsImport, map[string]any{"appID": appID, "path": file})
			}
			return nil, fmt.Errorf("unknown action %q", argStr(args, "action"))
		},
	}
}

// ---- wm_template ----

func templateTool() mcpTool {
	return mcpTool{
		Name:        "wm_template",
		Description: "Manage an app's file templates (generated config files) on its active config set.",
		Schema: schema("File templates", map[string]any{
			"action":  map[string]any{"type": "string", "description": "list, add, edit, delete", "enum": []string{"list", "add", "edit", "delete"}},
			"app":     strProp(appRefDesc),
			"file":    strProp("project-relative file path, e.g. .env.local (add/edit/delete)"),
			"content": strProp("template content, with {{placeholders}} (add/edit)"),
			"hidden":  boolProp("hide the content from AI clients (add/edit)"),
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			app, err := resolveApp(argStr(args, "app"))
			if err != nil {
				return nil, err
			}
			appID := appIDOf(app)
			find := func(file string) (map[string]any, error) {
				res, err := daoCall(api.MethodTemplatesList, map[string]any{"appID": appID})
				if err != nil {
					return nil, err
				}
				raw, _ := json.Marshal(res)
				var list []map[string]any
				_ = json.Unmarshal(raw, &list)
				for _, t := range list {
					if p, _ := t["file_path"].(string); equalFold(p, file) {
						return t, nil
					}
				}
				return nil, api.Errf(api.KindNotFound, "no template for %q on this app", file)
			}
			switch argStr(args, "action") {
			case "list":
				return daoCall(api.MethodTemplatesList, map[string]any{"appID": appID})
			case "add":
				file, err := need(args, "file", "the project-relative file path")
				if err != nil {
					return nil, err
				}
				body := map[string]any{"file_path": file}
				if content, ok := args["content"].(string); ok {
					body["content"] = content
				}
				if _, changed := args["hidden"]; changed {
					body["include_in_ai"] = !argBool(args, "hidden", false)
				}
				return daoCall(api.MethodTemplatesCreate, map[string]any{"appID": appID, "body": body})
			case "edit":
				file, err := need(args, "file", "the project-relative file path")
				if err != nil {
					return nil, err
				}
				tpl, err := find(file)
				if err != nil {
					return nil, err
				}
				id, _ := tpl["id"].(float64)
				body := map[string]any{}
				if content, ok := args["content"].(string); ok {
					body["content"] = content
				}
				if _, changed := args["hidden"]; changed {
					body["include_in_ai"] = !argBool(args, "hidden", false)
				}
				if len(body) == 0 {
					return nil, fmt.Errorf("nothing to change: pass \"content\" and/or \"hidden\"")
				}
				return daoCall(api.MethodTemplatesUpdate, map[string]any{"id": int64(id), "body": body})
			case "delete":
				file, err := need(args, "file", "the project-relative file path")
				if err != nil {
					return nil, err
				}
				tpl, err := find(file)
				if err != nil {
					return nil, err
				}
				id, _ := tpl["id"].(float64)
				return daoCall(api.MethodTemplatesDelete, map[string]any{"id": int64(id)})
			}
			return nil, fmt.Errorf("unknown action %q", argStr(args, "action"))
		},
	}
}

// ---- wm_run ----

func runTool() mcpTool {
	return mcpTool{
		Name:        "wm_run",
		Description: "Run, build and stop apps; check runner status; read terminal output. An app runs one session at a time.",
		Schema: schema("Runner", map[string]any{
			"action":    map[string]any{"type": "string", "description": "run, build, setup, stop, reload, status, logs", "enum": []string{"run", "build", "setup", "stop", "reload", "status", "logs"}},
			"app":       strProp(appRefDesc),
			"configSet": strProp("config set id or name (only used with worktree)"),
			"worktree":  strProp("run inside this git worktree path instead of the app folder"),
			"commandId": numProp("only this command id (logs)"),
			"tail":      numProp("only the last N bytes of each command's output (logs)"),
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			app, err := resolveApp(argStr(args, "app"))
			if err != nil {
				return nil, err
			}
			appID := appIDOf(app)
			switch argStr(args, "action") {
			case "run", "build", "setup":
				kind := argStr(args, "action")
				if wt, ok := argOpt(args, "worktree"); ok {
					params := map[string]any{"appID": appID, "path": wt, "kind": kind}
					if cs, ok := argOpt(args, "configSet"); ok {
						set, err := resolveConfigSet(app, cs)
						if err != nil {
							return nil, err
						}
						params["configSetID"] = set.ID
					}
					return daoCall(api.MethodGitWorktreeStart, params)
				}
				method := api.MethodRunnerRun
				switch kind {
				case "build":
					method = api.MethodRunnerBuild
				case "setup":
					method = api.MethodRunnerSetup
				}
				return daoCall(method, map[string]any{"appID": appID})
			case "stop":
				return daoCall(api.MethodRunnerStop, map[string]any{"appID": appID})
			case "reload":
				return daoCall(api.MethodRunnerReload, map[string]any{"appID": appID})
			case "status":
				return daoCall(api.MethodRunnerStatus, map[string]any{"appID": appID})
			case "logs":
				status, err := daoCall(api.MethodRunnerStatus, map[string]any{"appID": appID})
				if err != nil {
					return nil, err
				}
				raw, _ := json.Marshal(status)
				var st map[string]any
				_ = json.Unmarshal(raw, &st)
				procs, _ := st["processes"].([]any)
				if len(procs) == 0 {
					appName, _ := app["name"].(string)
					return nil, api.Errf(api.KindNotFound, "app %s has no active session", appName)
				}
				filter, hasFilter := argNum(args, "commandId")
				tail, _ := argNum(args, "tail")
				var out []map[string]any
				for _, p := range procs {
					pm, _ := p.(map[string]any)
					cid, _ := pm["commandId"].(float64)
					if hasFilter && int64(cid) != filter {
						continue
					}
					res, err := daoCall(api.MethodRunnerOutput, map[string]any{"appID": appID, "commandID": int64(cid)})
					if err != nil {
						return nil, err
					}
					raw, _ := json.Marshal(res)
					var o map[string]any
					_ = json.Unmarshal(raw, &o)
					data, _ := base64.StdEncoding.DecodeString(fmt.Sprint(o["data"]))
					if tail > 0 && int64(len(data)) > tail {
						data = data[int64(len(data))-tail:]
					}
					out = append(out, map[string]any{"commandId": int64(cid), "text": string(data)})
				}
				if hasFilter && len(out) == 0 {
					return nil, api.Errf(api.KindNotFound, "no command %d in the app's session", filter)
				}
				if out == nil {
					out = []map[string]any{}
				}
				return out, nil
			}
			return nil, fmt.Errorf("unknown action %q", argStr(args, "action"))
		},
	}
}

// ---- wm_git ----

func gitTool() mcpTool {
	return mcpTool{
		Name:        "wm_git",
		Description: "Branches, worktrees and shared worktree paths of an app's repository.",
		Schema: schema("Git and worktrees", map[string]any{
			"action":     map[string]any{"type": "string", "description": "info, fetch, branch_add, branch_rename, branch_checkout, branch_pull, branch_push, branch_delete, worktree_add, worktree_remove, worktree_prune, worktree_start, links_get, links_save, links_status", "enum": []string{"info", "fetch", "branch_add", "branch_rename", "branch_checkout", "branch_pull", "branch_push", "branch_delete", "worktree_add", "worktree_remove", "worktree_prune", "worktree_start", "links_get", "links_save", "links_status"}},
			"app":        strProp(appRefDesc),
			"name":       strProp("branch name (branch_*)"),
			"oldName":    strProp("old branch name (branch_rename; defaults to name)"),
			"newName":    strProp("new branch name (branch_rename)"),
			"base":       strProp("create the new branch from this ref (branch_add/worktree_add)"),
			"force":      boolProp("force push / delete / remove"),
			"branch":     strProp("branch for the worktree (worktree_add)"),
			"newBranch":  boolProp("create the branch instead of checking out an existing one (worktree_add)"),
			"path":       strProp("worktree path (worktree_add/remove/start, links_status)"),
			"linkShared": boolProp("link the app's shared paths into the worktree (worktree_add, default true)"),
			"links":      arrProp("shared paths: per-worktree override (worktree_add) or the app's new set (links_save)", "string"),
			"kind":       strProp("run, build or setup (worktree_start, default setup)"),
			"configSet":  strProp(setRefDesc + " (worktree_start)"),
			"create":     boolProp("link the missing shared paths first (links_status)"),
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			app, err := resolveApp(argStr(args, "app"))
			if err != nil {
				return nil, err
			}
			appID := appIDOf(app)
			base := map[string]any{"appID": appID}
			with := func(extra map[string]any) map[string]any {
				for k, v := range extra {
					base[k] = v
				}
				return base
			}
			switch argStr(args, "action") {
			case "info":
				return daoCall(api.MethodGitInfo, base)
			case "fetch":
				return daoCall(api.MethodGitFetchAll, base)
			case "branch_add":
				name, err := need(args, "name", "the branch name")
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodGitBranchCreate, with(map[string]any{"name": name, "base": argStr(args, "base")}))
			case "branch_rename":
				oldName := argStr(args, "oldName")
				if oldName == "" {
					oldName = argStr(args, "name")
				}
				newName, err := need(args, "newName", "the new branch name")
				if err != nil {
					return nil, err
				}
				if oldName == "" {
					return nil, fmt.Errorf("pass \"oldName\" (or \"name\") and \"newName\"")
				}
				return daoCall(api.MethodGitBranchRename, with(map[string]any{"oldName": oldName, "newName": newName}))
			case "branch_checkout":
				name, err := need(args, "name", "the branch name")
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodGitBranchCheckout, with(map[string]any{"name": name}))
			case "branch_pull":
				name, err := need(args, "name", "the branch name")
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodGitBranchPull, with(map[string]any{"name": name}))
			case "branch_push":
				name, err := need(args, "name", "the branch name")
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodGitBranchPush, with(map[string]any{"name": name, "force": argBool(args, "force", false)}))
			case "branch_delete":
				name, err := need(args, "name", "the branch name")
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodGitBranchDelete, with(map[string]any{"name": name, "force": argBool(args, "force", false)}))
			case "worktree_add":
				branch, err := need(args, "branch", "the branch for the worktree")
				if err != nil {
					return nil, err
				}
				body := map[string]any{
					"path": argStr(args, "path"), "branch": branch,
					"new_branch": argBool(args, "newBranch", false), "base": argStr(args, "base"),
					"link_shared": argBool(args, "linkShared", true),
				}
				if _, ok := args["links"]; ok {
					body["links"] = argStrList(args, "links")
				}
				return daoCall(api.MethodGitWorktreeAdd, with(map[string]any{"body": body}))
			case "worktree_remove":
				path, err := need(args, "path", "the worktree path")
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodGitWorktreeRemove, with(map[string]any{"path": path, "force": argBool(args, "force", false)}))
			case "worktree_prune":
				return daoCall(api.MethodGitWorktreePrune, base)
			case "worktree_start":
				path, err := need(args, "path", "the worktree path")
				if err != nil {
					return nil, err
				}
				kind := argStr(args, "kind")
				if kind == "" {
					kind = "setup"
				}
				if err := checkKind(kind); err != nil {
					return nil, err
				}
				params := with(map[string]any{"path": path, "kind": kind})
				if cs, ok := argOpt(args, "configSet"); ok {
					set, err := resolveConfigSet(app, cs)
					if err != nil {
						return nil, err
					}
					params["configSetID"] = set.ID
				}
				return daoCall(api.MethodGitWorktreeStart, params)
			case "links_get":
				return daoCall(api.MethodWorktreeLinksGet, base)
			case "links_save":
				return daoCall(api.MethodWorktreeLinksSave, with(map[string]any{"paths": argStrList(args, "links")}))
			case "links_status":
				path, err := need(args, "path", "the worktree path")
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodGitWorktreeLinks, with(map[string]any{"path": path, "create": argBool(args, "create", false)}))
			}
			return nil, fmt.Errorf("unknown action %q", argStr(args, "action"))
		},
	}
}

// ---- wm_blueprint ----

func blueprintTool() mcpTool {
	return mcpTool{
		Name:        "wm_blueprint",
		Description: "Blueprints: the recipes that scaffold new apps. Creating an app runs synchronously and fails fast on the first failing command.",
		Schema: schema("Blueprints", map[string]any{
			"action":      map[string]any{"type": "string", "description": "list, show, create, update, delete, create_app, resolve, cancel", "enum": []string{"list", "show", "create", "update", "delete", "create_app", "resolve", "cancel"}},
			"blueprint":   strProp("blueprint id or name"),
			"name":        strProp("blueprint name (create) or new app name (create_app)"),
			"description": strProp("one sentence about what the blueprint creates (create/update)"),
			"sampleName":  strProp("example app name for the new-app dialog (create/update)"),
			"createFolder": boolProp("commands run in the parent folder and the app folder is created first (create/update, default true)"),
			"commands":    map[string]any{"type": "array", "description": "shell lines in order; {{app_name}}, {{folder_name}} and {{app_dir}} are substituted (create/update)", "items": map[string]any{"type": "object", "properties": map[string]any{"command": map[string]any{"type": "string"}, "label": map[string]any{"type": "string"}}}},
			"workspace":   strProp(wsRefDesc + " (create_app)"),
			"parent":      strProp("parent folder for the new app (create_app, required)"),
			"folder":      strProp("folder name (create_app, default: slug of the app name)"),
			"runID":       strProp("run id to answer or stop (resolve/cancel)"),
			"decision":    map[string]any{"type": "string", "description": "skip or abort (resolve)", "enum": []string{"skip", "abort"}},
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			switch argStr(args, "action") {
			case "list":
				return daoCall(api.MethodBlueprintsList, nil)
			case "resolve":
				runID, err := need(args, "runID", "the run id")
				if err != nil {
					return nil, err
				}
				decision, err := need(args, "decision", "the decision")
				if err != nil {
					return nil, err
				}
				if decision != "skip" && decision != "abort" {
					return nil, fmt.Errorf("decision must be skip or abort, got %q", decision)
				}
				return daoCall(api.MethodBlueprintsResolve, map[string]any{"runID": runID, "action": decision})
			case "cancel":
				runID, err := need(args, "runID", "the run id")
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodBlueprintsCancel, map[string]any{"runID": runID})
			}
			blueprintCmds := func() ([]any, error) {
				var out []any
				for _, c := range argList(args, "commands") {
					m, ok := c.(map[string]any)
					if !ok {
						continue
					}
					cmd, _ := m["command"].(string)
					if cmd == "" {
						continue
					}
					entry := map[string]any{"command": cmd}
					if label, _ := m["label"].(string); label != "" {
						entry["label"] = label
					}
					out = append(out, entry)
				}
				return out, nil
			}
			switch argStr(args, "action") {
			case "create":
				name, err := need(args, "name", "the blueprint name")
				if err != nil {
					return nil, err
				}
				cmds, _ := blueprintCmds()
				return daoCall(api.MethodBlueprintsCreate, map[string]any{
					"name": name, "description": argStr(args, "description"),
					"sample_name": argStr(args, "sampleName"),
					"create_folder": argBool(args, "createFolder", true),
					"commands":      cmds,
				})
			case "show", "update", "delete":
				bp, err := resolveBlueprint(argStr(args, "blueprint"))
				if err != nil {
					return nil, err
				}
				id, _ := bp["id"].(float64)
				switch argStr(args, "action") {
				case "show":
					return bp, nil
				case "delete":
					return daoCall(api.MethodBlueprintsDelete, map[string]any{"id": int64(id)})
				default: // update: omitted fields keep their current values
					body := map[string]any{
						"name": bp["name"], "description": bp["description"],
						"sample_name": bp["sample_name"], "create_folder": bp["create_folder"],
						"commands": bp["commands"],
					}
					if v, ok := args["name"].(string); ok {
						body["name"] = v
					}
					if v, ok := args["description"].(string); ok {
						body["description"] = v
					}
					if v, ok := args["sampleName"].(string); ok {
						body["sample_name"] = v
					}
					if v, ok := args["createFolder"].(bool); ok {
						body["create_folder"] = v
					}
					if _, ok := args["commands"]; ok {
						cmds, _ := blueprintCmds()
						body["commands"] = cmds
					}
					return daoCall(api.MethodBlueprintsUpdate, map[string]any{"id": int64(id), "body": body})
				}
			case "create_app":
				bp, err := resolveBlueprint(argStr(args, "blueprint"))
				if err != nil {
					return nil, err
				}
				ws, err := resolveWorkspace(argStr(args, "workspace"))
				if err != nil {
					return nil, err
				}
				parent, err := need(args, "parent", "the parent folder for the new app")
				if err != nil {
					return nil, err
				}
				if abs, err := filepath.Abs(parent); err == nil {
					parent = abs
				}
				name := argStr(args, "name")
				if name == "" {
					name, _ = bp["sample_name"].(string)
				}
				if name == "" {
					return nil, fmt.Errorf("pass \"name\" (the blueprint has no sample name)")
				}
				folder := argStr(args, "folder")
				if folder == "" {
					folder = slugify(name)
				}
				bpid, _ := bp["id"].(float64)
				createFolder, _ := bp["create_folder"].(bool)
				return daoCall(api.MethodBlueprintsCreateApp, map[string]any{
					"run_id": fmt.Sprintf("mcp-%d-%d", os.Getpid(), time.Now().UnixNano()),
					"blueprint_id": int64(bpid), "workspace_id": ws.ID,
					"name": name, "parent_path": parent, "folder_name": folder,
					"create_folder": createFolder, "ask_on_error": false,
				})
			}
			return nil, fmt.Errorf("unknown action %q", argStr(args, "action"))
		},
	}
}

func resolveBlueprint(arg string) (map[string]any, error) {
	if arg == "" {
		return nil, fmt.Errorf("pass \"blueprint\" (id or name)")
	}
	res, err := daoCall(api.MethodBlueprintsList, nil)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(res)
	var all []map[string]any
	_ = json.Unmarshal(raw, &all)
	if id, ok := parseID(arg); ok {
		for _, bp := range all {
			if bid, _ := bp["id"].(float64); int64(bid) == id {
				return bp, nil
			}
		}
		return nil, api.Errf(api.KindNotFound, "no blueprint with id %d", id)
	}
	for _, bp := range all {
		if name, _ := bp["name"].(string); equalFold(name, arg) {
			return bp, nil
		}
	}
	return nil, api.Errf(api.KindNotFound, "no blueprint named %q", arg)
}

// ---- wm_ports ----

func portsTool() mcpTool {
	return mcpTool{
		Name:        "wm_ports",
		Description: "Find local port conflicts and stop the process behind one.",
		Schema: schema("Ports", map[string]any{
			"action": map[string]any{"type": "string", "description": "list, kill", "enum": []string{"list", "kill"}},
			"port":   numProp("only this port (list)"),
			"pid":    numProp("process id to stop (kill)"),
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			switch argStr(args, "action") {
			case "list":
				res, err := daoCall(api.MethodPortsList, nil)
				if err != nil {
					return nil, err
				}
				if port, ok := argNum(args, "port"); ok {
					raw, _ := json.Marshal(res)
					var m map[string]any
					_ = json.Unmarshal(raw, &m)
					var kept []any
					for _, p := range m["processes"].([]any) {
						if pm, _ := p.(map[string]any); pm != nil {
							if pt, _ := pm["port"].(float64); int64(pt) == port {
								kept = append(kept, p)
							}
						}
					}
					m["processes"] = kept
					return m, nil
				}
				return res, nil
			case "kill":
				pid, ok := argNum(args, "pid")
				if !ok || pid <= 0 {
					return nil, fmt.Errorf("pass \"pid\": the process id to stop")
				}
				return daoCall(api.MethodPortsKill, map[string]any{"pid": pid})
			}
			return nil, fmt.Errorf("unknown action %q", argStr(args, "action"))
		},
	}
}

// ---- wm_system ----

func systemTool() mcpTool {
	return mcpTool{
		Name:        "wm_system",
		Description: "Settings, project file inspection and opening URLs in the browser.",
		Schema: schema("System helpers", map[string]any{
			"action": map[string]any{"type": "string", "description": "settings_get, settings_set, fs_validate, fs_read, open", "enum": []string{"settings_get", "settings_set", "fs_validate", "fs_read", "open"}},
			"key":    strProp("setting key (settings_get: optional filter; settings_set: required)"),
			"value":  strProp("setting value (settings_set)"),
			"path":   strProp("folder to validate (fs_validate) or file inside the app to read (fs_read)"),
			"app":    strProp(appRefDesc + " (fs_read)"),
			"url":    strProp("URL to open in the system browser (open)"),
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			switch argStr(args, "action") {
			case "settings_get":
				res, err := daoCall(api.MethodSettingsGet, nil)
				if err != nil {
					return nil, err
				}
				if key, ok := argOpt(args, "key"); ok {
					if m, ok := res.(map[string]any); ok {
						return map[string]any{key: m[key]}, nil
					}
				}
				return res, nil
			case "settings_set":
				key, err := need(args, "key", "the setting key")
				if err != nil {
					return nil, err
				}
				value, err := need(args, "value", "the setting value")
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodSettingsSet, map[string]any{"key": key, "value": value})
			case "fs_validate":
				path, err := need(args, "path", "the folder to validate")
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodFsValidatePath, map[string]any{"path": path})
			case "fs_read":
				path, err := need(args, "path", "the file to read, relative to the app folder")
				if err != nil {
					return nil, err
				}
				app, err := resolveApp(argStr(args, "app"))
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodFsReadAppFile, map[string]any{"appID": appIDOf(app), "path": path})
			case "open":
				url, err := need(args, "url", "the URL to open")
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodOpenExternal, map[string]any{"url": url})
			}
			return nil, fmt.Errorf("unknown action %q", argStr(args, "action"))
		},
	}
}

// ---- wm_call (escape hatch) ----

func callTool() mcpTool {
	allowed := map[string]bool{}
	for _, m := range api.AgentMethods() {
		allowed[m.Name] = true
	}
	return mcpTool{
		Name:        "wm_call",
		Description: "Call any daemon method available to an agent client directly with raw params. Escape hatch for what the curated tools do not cover; prefer them when they fit.",
		Schema: schema("Raw daemon call", map[string]any{
			"method": strProp("daemon method, e.g. apps.list (see wm_call with method \"\" for the full list)"),
			"params": map[string]any{"type": "object", "description": "method params object"},
		}, "method"),
		Handler: func(args map[string]any) (any, error) {
			method := argStr(args, "method")
			if method == "" {
				names := make([]string, 0, len(allowed))
				for _, m := range api.AgentMethods() {
					names = append(names, m.Name)
				}
				return names, nil
			}
			if !allowed[method] {
				return nil, fmt.Errorf("method %q is not available to MCP clients (GUI-only or unknown)", method)
			}
			var params any
			if p, ok := args["params"]; ok {
				params = p
			}
			return daoCall(method, params)
		},
	}
}
