package main

import (
	"encoding/json"
	"fmt"
	"os"

	"workspace-manager/api"
	"workspace-manager/api/client"
)

// dial connects to the daemon, starting it when it is not running. The MCP
// server is stateless across tool calls, so every call dials fresh: simple,
// and the daemon idles out on its own when nobody uses it.
func dial() (*client.Client, error) {
	opts := client.Options{Kind: "cli", Version: version, Start: client.StartDaemon()}
	if socket := os.Getenv("WM_SOCKET"); socket != "" {
		opts.Socket = socket
	}
	return client.Dial(opts)
}

// daoCall runs one daemon method and returns the decoded result.
func daoCall(method string, params any) (any, error) {
	conn, err := dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	var raw json.RawMessage
	if err := conn.Call(method, params, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return map[string]any{"ok": true}, nil
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ---- argument helpers: tool arguments arrive as map[string]any ----

func argStr(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func argOpt(args map[string]any, key string) (string, bool) {
	v, ok := args[key].(string)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

func argBool(args map[string]any, key string, def bool) bool {
	v, ok := args[key].(bool)
	if !ok {
		return def
	}
	return v
}

func argNum(args map[string]any, key string) (int64, bool) {
	switch v := args[key].(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return n, true
		}
	}
	return 0, false
}

func argList(args map[string]any, key string) []any {
	v, _ := args[key].([]any)
	return v
}

func argStrList(args map[string]any, key string) []string {
	var out []string
	for _, v := range argList(args, key) {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func need(args map[string]any, key, what string) (string, error) {
	v := argStr(args, key)
	if v == "" {
		return "", fmt.Errorf("missing %s: pass %q", what, key)
	}
	return v, nil
}

// ---- id-or-name resolution, mirroring cli/cmd/wm/resolve.go ----

type idName struct {
	ID   int64
	Name string
}

func workspaceID(app map[string]any, args map[string]any) (int64, error) {
	// Explicit workspace argument wins; otherwise the app's own workspace.
	if w, ok := argOpt(args, "workspace"); ok {
		ws, err := resolveWorkspace(w)
		if err != nil {
			return 0, err
		}
		return ws.ID, nil
	}
	if app != nil {
		if id, ok := app["workspace_id"].(float64); ok {
			return int64(id), nil
		}
	}
	return 0, fmt.Errorf("no workspace: pass \"workspace\" (id or name)")
}

func allWorkspaces() ([]map[string]any, error) {
	out, err := daoCall(api.MethodWorkspacesList, nil)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(out)
	var ws []map[string]any
	if err := json.Unmarshal(raw, &ws); err != nil {
		return nil, err
	}
	return ws, nil
}

func resolveWorkspace(arg string) (idName, error) {
	ws, err := allWorkspaces()
	if err != nil {
		return idName{}, err
	}
	for _, w := range ws {
		id, _ := w["id"].(float64)
		name, _ := w["name"].(string)
		if arg == fmt.Sprintf("%.0f", id) || equalFold(name, arg) {
			return idName{ID: int64(id), Name: name}, nil
		}
	}
	return idName{}, api.Errf(api.KindNotFound, "no workspace %q", arg)
}

func allApps() ([]map[string]any, error) {
	ws, err := allWorkspaces()
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for _, w := range ws {
		id, _ := w["id"].(float64)
		res, err := daoCall(api.MethodAppsList, map[string]any{"workspaceID": int64(id)})
		if err != nil {
			return nil, err
		}
		raw, _ := json.Marshal(res)
		var apps []map[string]any
		if err := json.Unmarshal(raw, &apps); err != nil {
			return nil, err
		}
		out = append(out, apps...)
	}
	return out, nil
}

func resolveApp(arg string) (map[string]any, error) {
	if arg == "" {
		apps, err := allApps()
		if err != nil {
			return nil, err
		}
		if len(apps) == 1 {
			return apps[0], nil
		}
		if len(apps) == 0 {
			return nil, api.Errf(api.KindNotFound, "there are no apps yet")
		}
		return nil, api.Errf(api.KindInvalid, "several apps exist; pass \"app\" (id or name)")
	}
	if id, ok := parseID(arg); ok {
		res, err := daoCall(api.MethodAppsGet, map[string]any{"id": id})
		if err != nil {
			return nil, err
		}
		m, _ := res.(map[string]any)
		return m, nil
	}
	apps, err := allApps()
	if err != nil {
		return nil, err
	}
	var matches []map[string]any
	for _, a := range apps {
		if name, _ := a["name"].(string); equalFold(name, arg) {
			matches = append(matches, a)
		}
	}
	switch len(matches) {
	case 0:
		return nil, api.Errf(api.KindNotFound, "no app named %q", arg)
	case 1:
		return matches[0], nil
	default:
		return nil, api.Errf(api.KindInvalid, "%d apps are named %q; pass an id", len(matches), arg)
	}
}

func appIDOf(app map[string]any) int64 {
	id, _ := app["id"].(float64)
	return int64(id)
}

func resolveConfigSet(app map[string]any, arg string) (idName, error) {
	res, err := daoCall(api.MethodConfigSetsList, map[string]any{"appID": appIDOf(app)})
	if err != nil {
		return idName{}, err
	}
	raw, _ := json.Marshal(res)
	var sets []map[string]any
	if err := json.Unmarshal(raw, &sets); err != nil {
		return idName{}, err
	}
	if arg == "" {
		if len(sets) == 1 {
			return setIDName(sets[0]), nil
		}
		if active, _ := app["active_config_set_id"].(float64); active != 0 {
			for _, s := range sets {
				if id, _ := s["id"].(float64); int64(id) == int64(active) {
					return setIDName(s), nil
				}
			}
		}
		if len(sets) > 0 {
			return setIDName(sets[0]), nil
		}
		appName, _ := app["name"].(string)
		return idName{}, api.Errf(api.KindNotFound, "app %s has no config sets", appName)
	}
	if id, ok := parseID(arg); ok {
		for _, s := range sets {
			if sid, _ := s["id"].(float64); int64(sid) == id {
				return setIDName(s), nil
			}
		}
		return idName{}, api.Errf(api.KindNotFound, "no config set with id %d on this app", id)
	}
	var matches []map[string]any
	for _, s := range sets {
		if name, _ := s["name"].(string); equalFold(name, arg) {
			matches = append(matches, s)
		}
	}
	if len(matches) == 1 {
		return setIDName(matches[0]), nil
	}
	if len(matches) > 1 {
		return idName{}, api.Errf(api.KindInvalid, "%d config sets are named %q; pass an id", len(matches), arg)
	}
	return idName{}, api.Errf(api.KindNotFound, "no config set named %q on this app", arg)
}

func setIDName(s map[string]any) idName {
	id, _ := s["id"].(float64)
	name, _ := s["name"].(string)
	return idName{ID: int64(id), Name: name}
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		// still compare case-insensitively without importing strings everywhere
		if len(a) == 0 || len(b) == 0 {
			return false
		}
	}
	la, lb := lowerStr(a), lowerStr(b)
	return la == lb
}

func lowerStr(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

func parseID(s string) (int64, bool) {
	var n int64
	var neg bool
	for i := 0; i < len(s); i++ {
		c := s[i]
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int64(c-'0')
	}
	if neg {
		n = -n
	}
	if n <= 0 {
		return 0, false
	}
	return n, true
}

func slugify(s string) string {
	s = lowerStr(s)
	var b []byte
	lastDash := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			b = append(b, c)
			lastDash = false
		default:
			if !lastDash {
				b = append(b, '-')
				lastDash = true
			}
		}
	}
	for len(b) > 0 && b[0] == '-' {
		b = b[1:]
	}
	for len(b) > 0 && b[len(b)-1] == '-' {
		b = b[:len(b)-1]
	}
	return string(b)
}
