package main

import (
	_ "embed"
)

// guideText is the guide the server hands to the agent: as the server's
// instructions, as the workspace://guide resource and through wm_guide.
//
//go:embed guide.md
var guideText string

const guideURI = "workspace://guide"

func guideTool() mcpTool {
	return mcpTool{
		Name:        "wm_guide",
		Description: "How to manage the user's Workspace Manager setup with these tools: the data model, how to run an app (this server runs nothing), hidden values, and etiquette. Read it first.",
		Schema:      schema("Guide", map[string]any{}),
		Handler:     func(map[string]any) (any, error) { return guideText, nil },
	}
}
