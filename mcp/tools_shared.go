package mcp

import (
	"encoding/json"
	"fmt"

	"workspace-manager/shared/api"
)

// ---- wm_shared ----

// asMaps decodes a daemon result into a list of objects.
func asMaps(v any) []map[string]any {
	raw, _ := json.Marshal(v)
	var out []map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}

// resolveSharedSet finds a workspace shared set by id or name.
func resolveSharedSet(workspaceID int64, ref string) (idName, error) {
	if ref == "" {
		return idName{}, fmt.Errorf("missing shared config set: pass \"shared\" (id or name)")
	}
	res, err := daoCall(api.MethodSharedSetsList, map[string]any{"workspaceID": workspaceID})
	if err != nil {
		return idName{}, err
	}
	sets := asMaps(res)
	if id, ok := parseID(ref); ok {
		for _, s := range sets {
			if sid, _ := s["id"].(float64); int64(sid) == id {
				return setIDName(s), nil
			}
		}
	}
	for _, s := range sets {
		if name, _ := s["name"].(string); equalFold(name, ref) {
			return setIDName(s), nil
		}
	}
	return idName{}, api.Errf(api.KindNotFound, "no shared config set %q in this workspace", ref)
}

func sharedTool() mcpTool {
	return mcpTool{
		Name: "wm_shared",
		Description: "A workspace's shared config: sets of env vars several apps can use (for example a database URL per environment). " +
			"Edit them with set/delete; a change reaches every app config set that uses them at once. " +
			"An app config set takes all of a shared set's variables (also ones added later) with link (any shared set, whatever the names) and stops with unlink; " +
			"linked variables are read-only in the app (wm_env refuses them). Private values are never returned.",
		Schema: schema("Workspace shared config", map[string]any{
			"action":    map[string]any{"type": "string", "description": "list (shared sets), get (a set's variables and where they are used), create_set, rename_set, delete_set, set (create or update a variable), delete (a variable), link, unlink", "enum": []string{"list", "get", "create_set", "rename_set", "delete_set", "set", "delete", "link", "unlink"}},
			"workspace": strProp("workspace id or name; omit when the app or the only workspace tells it"),
			"shared":    strProp("shared config set id or name"),
			"name":      strProp("set name (create_set) or the new name (rename_set)"),
			"key":       strProp("variable name (set/delete)"),
			"value":     strProp("variable value (set; omit to keep the stored value on update)"),
			"options":   map[string]any{"type": "array", "description": "make the variable a choice: its options in order (set); [] turns it back into a plain value", "items": map[string]any{"type": "object", "properties": map[string]any{"label": map[string]any{"type": "string"}, "value": map[string]any{"type": "string"}}, "required": []string{"label"}}},
			"selected":  strProp("label of the option in use (set; needs options)"),
			"private":   boolProp("make the value private: agents never see it (set); omit to keep the current flag; a private variable cannot be made public here"),
			"app":       strProp(appRefDesc + " (link/unlink)"),
			"set":       strProp(setRefDesc + " (link/unlink)"),
			"replace":   boolProp("let a shared variable overwrite an own variable of the same name (link)"),
			"keep":      boolProp("keep the shared variables as the app's own instead of deleting them (unlink)"),
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			action := argStr(args, "action")
			var app map[string]any
			if action == "link" || action == "unlink" || argStr(args, "app") != "" {
				a, err := resolveApp(argStr(args, "app"))
				if err != nil {
					return nil, err
				}
				app = a
			}
			wsID, err := workspaceID(app, args)
			if err != nil {
				// A single workspace needs no argument.
				all, aerr := allWorkspaces()
				if aerr != nil || len(all) != 1 {
					return nil, err
				}
				id, _ := all[0]["id"].(float64)
				wsID = int64(id)
			}

			switch action {
			case "list":
				return daoCall(api.MethodSharedSetsList, map[string]any{"workspaceID": wsID})
			case "create_set":
				name, err := need(args, "name", "the set name")
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodSharedSetsCreate, map[string]any{"workspaceID": wsID, "name": name})
			case "link", "unlink":
				set, err := resolveConfigSet(app, argStr(args, "set"))
				if err != nil {
					return nil, err
				}
				if action == "unlink" {
					return daoCall(api.MethodConfigSetsUnlinkShared, map[string]any{"id": set.ID, "keep": argBool(args, "keep", false)})
				}
				shared, err := resolveSharedSet(wsID, argStr(args, "shared"))
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodConfigSetsLinkShared, map[string]any{
					"id":   set.ID,
					"body": map[string]any{"shared_set_id": shared.ID, "replace": argBool(args, "replace", false)},
				})
			}

			shared, err := resolveSharedSet(wsID, argStr(args, "shared"))
			if err != nil {
				return nil, err
			}
			switch action {
			case "get":
				return daoCall(api.MethodSharedSetsGet, map[string]any{"id": shared.ID})
			case "rename_set":
				name, err := need(args, "name", "the new name")
				if err != nil {
					return nil, err
				}
				return daoCall(api.MethodSharedSetsRename, map[string]any{"id": shared.ID, "name": name})
			case "delete_set":
				return daoCall(api.MethodSharedSetsDelete, map[string]any{"id": shared.ID})
			case "set", "delete":
				key, err := need(args, "key", "the variable name")
				if err != nil {
					return nil, err
				}
				detail, err := daoCall(api.MethodSharedSetsGet, map[string]any{"id": shared.ID})
				if err != nil {
					return nil, err
				}
				var d struct{ Vars []map[string]any }
				raw, _ := json.Marshal(detail)
				_ = json.Unmarshal(raw, &d)
				var existing map[string]any
				for _, v := range d.Vars {
					if k, _ := v["key"].(string); k == key {
						existing = v
					}
				}
				if action == "delete" {
					if existing == nil {
						return nil, api.Errf(api.KindNotFound, "no variable named %q in shared set %q", key, shared.Name)
					}
					id, _ := existing["id"].(float64)
					return daoCall(api.MethodSharedVarsDelete, map[string]any{"id": int64(id)})
				}
				body := map[string]any{"key": key}
				if v, ok := args["value"]; ok {
					body["value"] = v
				}
				if v, ok := args["options"]; ok {
					body["options"] = v
				}
				if v, ok := args["selected"]; ok {
					body["selected"] = v
				}
				if existing != nil {
					if _, specified := args["private"]; specified {
						body["private"] = argBool(args, "private", false)
					}
					id, _ := existing["id"].(float64)
					return daoCall(api.MethodSharedVarsUpdate, map[string]any{"id": int64(id), "body": body})
				}
				body["private"] = argBool(args, "private", false)
				return daoCall(api.MethodSharedVarsCreate, map[string]any{"setID": shared.ID, "body": body})
			}
			return nil, fmt.Errorf("unknown action %q", action)
		},
	}
}

// refuseShared stops wm_env from changing a variable the config set takes from
// the workspace's shared config: it is read-only in the app, and the way to
// change it is wm_shared (which reaches every config set using it) or unlink.
func refuseShared(v map[string]any) error {
	if id, ok := v["shared_var_id"].(float64); ok && id != 0 {
		key, _ := v["key"].(string)
		return api.Errf(api.KindInvalid, "%s is shared from the workspace and read-only here: change it with wm_shared (set), or wm_shared unlink this config set", key)
	}
	return nil
}
