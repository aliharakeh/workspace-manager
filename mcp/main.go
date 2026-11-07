// Package mcp serves the Workspace Manager agent tools over stdio.
package mcp

import "workspace-manager/shared/api"

var version = api.Version

// Run serves MCP until its client closes stdin. Stdout is reserved for JSON-RPC.
func Run() error { return serve() }
