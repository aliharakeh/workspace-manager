package main

import (
	"encoding/json"
	"fmt"

	"workspace-manager/shared/api"
)

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
