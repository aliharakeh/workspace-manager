// Command wm-mcp is a Model Context Protocol server for workspace-manager.
//
// It speaks MCP over stdio (newline-delimited JSON-RPC) and gives an AI agent
// the user's workspace data and a guide for managing it. It opens the same
// database as the desktop app, edits configuration, and runs nothing: for a
// run it hands the agent the commands (wm_run) and "wm-mcp exec" runs one of
// them with the app's environment. Values excluded from AI stay hidden.
//
// Configure any MCP client with:
//
//	{ "command": "wm-mcp" }
//
// WM_DATA_DIR moves it to another data folder, as for the app.
package main

import (
	"fmt"
	"os"
)

// version is the build version, reported to MCP clients.
var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "exec" {
		os.Exit(runExec(os.Args[2:]))
	}
	if err := openStore(); err != nil {
		fmt.Fprintln(os.Stderr, "wm-mcp:", err)
		os.Exit(1)
	}
	err := serve()
	releaseOwnSessions()
	_ = database.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, "wm-mcp:", err)
		os.Exit(1)
	}
}
