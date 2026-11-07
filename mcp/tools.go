package mcp

// tool is one MCP tool: a name, a description and a JSON-Schema input.
type mcpTool struct {
	Name        string
	Description string
	Schema      map[string]any
	Handler     func(args map[string]any) (any, error)
}

// schema builds a JSON-Schema object node.
func schema(desc string, props map[string]any, required ...string) map[string]any {
	s := map[string]any{"type": "object", "description": desc, "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func strProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}
func boolProp(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}
func numProp(desc string) map[string]any {
	return map[string]any{"type": "number", "description": desc}
}
func arrProp(desc, item string) map[string]any {
	return map[string]any{"type": "array", "description": desc, "items": map[string]any{"type": item}}
}

const appRefDesc = "app id or name; omit when there is exactly one app"
const wsRefDesc = "workspace id or name; omit to cover all workspaces (or the only one)"
const setRefDesc = "config set id or name; omit for the app's active set"

func allTools() []mcpTool {
	return []mcpTool{
		overviewTool(),
		appTool(),
		configTool(),
		envTool(),
		envScriptTool(),
		sharedTool(),
		templateTool(),
		runTool(),
		blueprintTool(),
		portsTool(),
		fileTool(),
		callTool(),
	}
}

func findTool(name string) *mcpTool {
	for _, t := range allTools() {
		if t.Name == name {
			t := t
			return &t
		}
	}
	return nil
}

func toolSchemas() []any {
	out := make([]any, 0, len(allTools()))
	for _, t := range allTools() {
		out = append(out, map[string]any{
			"name": t.Name, "description": t.Description, "inputSchema": t.Schema,
		})
	}
	return out
}
