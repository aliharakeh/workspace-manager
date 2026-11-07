package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"workspace-manager/shared/api"
)

func blueprintTool() mcpTool {
	return mcpTool{
		Name:        "wm_blueprint",
		Description: "Blueprints: the recipes that scaffold new apps. Creating an app runs synchronously and fails fast on the first failing command.",
		Schema: schema("Blueprints", map[string]any{
			"action":       map[string]any{"type": "string", "description": "list, show, create, update, delete, create_app, resolve, cancel", "enum": []string{"list", "show", "create", "update", "delete", "create_app", "resolve", "cancel"}},
			"blueprint":    strProp("blueprint id or name"),
			"name":         strProp("blueprint name (create) or new app name (create_app)"),
			"description":  strProp("one sentence about what the blueprint creates (create/update)"),
			"sampleName":   strProp("example app name for the new-app dialog (create/update)"),
			"createFolder": boolProp("commands run in the parent folder and the app folder is created first (create/update, default true)"),
			"commands":     map[string]any{"type": "array", "description": "shell lines in order; {{app_name}}, {{folder_name}} and {{app_dir}} are substituted (create/update)", "items": map[string]any{"type": "object", "properties": map[string]any{"command": map[string]any{"type": "string"}, "label": map[string]any{"type": "string"}}}},
			"workspace":    strProp(wsRefDesc + " (create_app)"),
			"parent":       strProp("parent folder for the new app (create_app, required)"),
			"folder":       strProp("folder name (create_app, default: slug of the app name)"),
			"runID":        strProp("run id to answer or stop (resolve/cancel)"),
			"decision":     map[string]any{"type": "string", "description": "skip or abort (resolve)", "enum": []string{"skip", "abort"}},
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
					"sample_name":   argStr(args, "sampleName"),
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
					"run_id":       fmt.Sprintf("mcp-%d-%d", os.Getpid(), time.Now().UnixNano()),
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
