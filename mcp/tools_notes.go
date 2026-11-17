package mcp

import (
	"workspace-manager/shared/api"
)

func notesTool() mcpTool {
	return mcpTool{
		Name:        "wm_notes",
		Description: "Manage titled app notes with text, an image and/or checklist items. Each item has text and a done boolean; done means crossed out. Images are copied beside the database and image_path is the managed absolute path. Save replaces title/content/items; omit image_data to retain the image, use an empty string to remove it. Notes belong to the app, across config sets.",
		Schema: schema("App notes", map[string]any{
			"action":     map[string]any{"type": "string", "enum": []string{"list", "save", "delete", "image"}},
			"app":        strProp(appRefDesc),
			"id":         numProp("note id; omit or 0 to create on save"),
			"title":      strProp("required note title on save"),
			"content":    strProp("note text on save"),
			"items":      map[string]any{"type": "array", "items": schema("Checklist item", map[string]any{"text": strProp("item text"), "done": boolProp("completed/crossed out")}, "text", "done")},
			"image_data": strProp("base64 PNG/JPEG/GIF, maximum 5 MB decoded; empty removes image; omit retains it"),
		}, "action"),
		Handler: func(args map[string]any) (any, error) {
			app, err := resolveApp(argStr(args, "app"))
			if err != nil {
				return nil, err
			}
			params := map[string]any{"appID": appIDOf(app), "id": args["id"]}
			switch argStr(args, "action") {
			case "list":
				return daoCall(api.MethodNotesList, params)
			case "delete":
				return daoCall(api.MethodNotesDelete, params)
			case "image":
				return daoCall(api.MethodNotesImage, params)
			case "save":
				body := map[string]any{}
				for _, key := range []string{"title", "content", "items", "image_data"} {
					if v, ok := args[key]; ok {
						body[key] = v
					}
				}
				params["body"] = body
				return daoCall(api.MethodNotesSave, params)
			}
			return nil, api.Errf(api.KindInvalid, "unknown notes action %q", argStr(args, "action"))
		},
	}
}
