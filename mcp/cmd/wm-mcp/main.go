// Command wm-mcp is a Model Context Protocol server for workspace-manager.
//
// It speaks MCP over stdio (newline-delimited JSON-RPC) and drives the wmd
// daemon through api/client, starting the daemon on demand. It identifies as
// an "agent" client, so AI methods stay refused and AI-excluded values stay
// hidden.
//
// Configure any MCP client with:
//
//	{ "command": "wm-mcp" }
//
// WM_DATA_DIR and WM_SOCKET are honored the same way the daemon's other
// clients honor them.
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
