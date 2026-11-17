package mcp

import (
	"fmt"

	"workspace-manager/shared/api"
)

func configTool() mcpTool {
	return mcpTool{
		Name:        "wm_config",
		Description: "Manage config sets and custom named run configurations. Configuration keys stay stable across renames.",
		Schema: schema("Config sets and command lists", map[string]any{
			"action":       map[string]any{"type": "string", "description": "list, detail, create, rename, activate, copy, delete, list_commands, get_commands, save_commands, delete_commands, scripts", "enum": []string{"list", "detail", "create", "rename", "activate", "copy", "delete", "get_commands", "save_commands", "list_commands", "delete_commands", "scripts"}},
			"app":          strProp(appRefDesc),
			"set":          strProp(setRefDesc),
			"name":         strProp("name for a config set (create/rename) or run configuration (save_commands)"),
			"detect_ready": boolProp("save_commands: detect ready URLs and idle activity; independent of the configuration name"),
			"kind":         strProp("stable run configuration key (get_commands/save_commands/delete_commands); arbitrary key for new configurations; list_commands returns keys"),
			"from":         strProp("source config set id or name (create/copy)"),
			"parts":        strProp("comma list: env,templates,run_configs (legacy dev,prod,build,setup also supported) (create/copy, default all)"),
			"noActivate":   boolProp("do not make the new set active (create)"),
			"mode":         strProp("parallel or sequential (save_commands)"),
			"commands":     map[string]any{"type": "array", "description": "command lines in order (save_commands)", "items": map[string]any{"type": "object", "properties": map[string]any{"command": map[string]any{"type": "string"}, "label": map[string]any{"type": "string"}}}},
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
			case "list_commands", "delete_commands":
				setID, err := setIDArg(app, args)
				if err != nil {
					return nil, err
				}
				params := map[string]any{"appID": appID, "configSetID": setID}
				if action == "list_commands" {
					return daoCall(api.MethodRunConfigList, params)
				}
				kind, err := need(args, "kind", "the configuration key")
				if err != nil {
					return nil, err
				}
				params["kind"] = kind
				return daoCall(api.MethodRunConfigDelete, params)
			case "get_commands":
				kind, err := need(args, "kind", "the command list kind")
				if err != nil {
					return nil, err
				}
				if err := checkKind(kind); err != nil {
					return nil, err
				}
				setID, err := setIDArg(app, args)
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodRunConfigGet, map[string]any{"appID": appID, "configSetID": setID, "kind": kind})
			case "save_commands":
				kind, err := need(args, "kind", "the command list kind")
				if err != nil {
					return nil, err
				}
				if err := checkKind(kind); err != nil {
					return nil, err
				}
				var cmds []any
				if _, ok := args["commands"]; ok {
					cmds = []any{}
				}
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

				body := map[string]any{"commands": cmds}
				if name, ok := argOpt(args, "name"); ok {
					body["name"] = name
				}
				if _, ok := args["detect_ready"]; ok {
					body["detect_ready"] = argBool(args, "detect_ready", false)
				}
				if mode, ok := argOpt(args, "mode"); ok {
					body["mode"] = mode
				}
				setID, err := setIDArg(app, args)
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodRunConfigSave, map[string]any{"appID": appID, "configSetID": setID, "kind": kind, "body": body})
			case "scripts":
				return daoCall(api.MethodPackageScriptsList, map[string]any{"appID": appID})
			}
			return nil, fmt.Errorf("unknown action %q", action)
		},
	}
}

// setIDArg resolves the optional "set" argument (or the older "configSet"
// alias) to a config set id of the app; 0 means the app's active set. Nothing
// ever has to be activated to work on another set.
func setIDArg(app map[string]any, args map[string]any) (int64, error) {
	ref := argStr(args, "set")
	if ref == "" {
		ref = argStr(args, "configSet")
	}
	if ref == "" {
		return 0, nil
	}
	set, err := resolveConfigSet(app, ref)
	if err != nil {
		return 0, err
	}
	return set.ID, nil
}

func copyParts(list string) map[string]any {
	if list == "" {
		list = "env,templates,run_configs"
	}
	parts := map[string]any{"env": false, "templates": false, "dev": false, "prod": false, "build": false, "setup": false}
	for _, p := range splitComma(list) {
		switch p {
		case "env", "templates", "dev", "prod", "build", "setup", "run_configs":
			parts[p] = true
		}
	}
	return parts
}

func checkKind(kind string) error {
	if kind == "" || len(kind) > 128 {
		return fmt.Errorf("configuration key must be between 1 and 128 characters")
	}
	return nil
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
