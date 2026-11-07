package mcp

import (
	"workspace-manager/shared/api"
)

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
