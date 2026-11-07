package mcp

import (
	"workspace-manager/shared/api"
)

func fileTool() mcpTool {
	return mcpTool{
		Name:        "wm_file",
		Description: "Read a file inside an app's project folder. Settings, git and opening URLs are GUI-only and not available here.",
		Schema: schema("Read an app file", map[string]any{
			"path": strProp("file inside the app, relative to the app folder"),
			"app":  strProp(appRefDesc),
		}, "path"),
		Handler: func(args map[string]any) (any, error) {
			path, err := need(args, "path", "the file to read, relative to the app folder")
			if err != nil {
				return nil, err
			}
			app, err := resolveApp(argStr(args, "app"))
			if err != nil {
				return nil, err
			}
			return daoCall(api.MethodFsReadAppFile, map[string]any{"appID": appIDOf(app), "path": path})
		},
	}
}
