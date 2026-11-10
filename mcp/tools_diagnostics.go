package mcp

import "workspace-manager/shared/api"

func diagnosticsTool() mcpTool {
	return mcpTool{
		Name:        "wm_diagnostics",
		Structured:  true,
		Description: "Read detected errors and warnings from the app's current/latest run, retained independently of terminal scrollback and across daemon restarts. Returns small summaries; use id and sessionId for retained stack/context. Patterns cannot detect every error or establish health; problems are historical, not automatically resolved. Only the latest run per app is kept.",
		Schema: schema("Diagnostics", map[string]any{
			"app":       strProp(appRefDesc),
			"sessionId": strProp("sessionId from the previous result; required with cursor or id to avoid mixing runs"),
			"commandId": numProp("filter by command id"),
			"severity":  map[string]any{"type": "string", "enum": []string{"error", "warning"}},
			"cursor":    numProp("revision from nextCursor; fetch new or updated groups after this revision (repeated problems reappear)"),
			"limit":     numProp("page size, default 10, maximum 20"),
			"id":        numProp("retrieve one diagnostic with its retained context; requires sessionId"),
		}),
		Handler: func(args map[string]any) (any, error) {
			app, err := resolveApp(argStr(args, "app"))
			if err != nil {
				return nil, err
			}
			params := map[string]any{"appID": appIDOf(app)}
			for _, key := range []string{"sessionId", "commandId", "severity", "cursor", "limit", "id"} {
				if value, ok := args[key]; ok {
					params[key] = value
				}
			}
			return daoCall(api.MethodRunnerDiagnostics, params)
		},
	}
}
