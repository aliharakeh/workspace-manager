// Command wm-mcp is a Model Context Protocol server for workspace-manager.
//
// It speaks MCP over stdio (newline-delimited JSON-RPC) and drives the wmd
// daemon through the same API client the wm CLI uses, starting the daemon on
// demand. It identifies as a "cli" client, so AI-excluded values stay hidden,
// exactly as they are for wm and any other external agent.
//
// Configure any MCP client with:
//
//	{ "command": "wm-mcp" }
//
// WM_DATA_DIR and WM_SOCKET are honored the same way the CLI honors them.
package main

import (
	"fmt"
	"os"
)

// version is the build version, sent in the daemon handshake.
var version = "dev"

func main() {
	if err := serve(); err != nil {
		fmt.Fprintln(os.Stderr, "wm-mcp:", err)
		os.Exit(1)
	}
}
