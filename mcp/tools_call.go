package mcp

import (
	"fmt"

	"workspace-manager/shared/api"
)

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
