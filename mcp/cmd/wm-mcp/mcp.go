package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// Minimal MCP over stdio: newline-delimited JSON-RPC 2.0. Only what a tool
// server needs: initialize, tools/list, tools/call, ping, plus empty answers
// to resources/list and prompts/list probes. No external dependencies.

type rpcID any

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      rpcID           `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      rpcID     `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

func serve() error {
	in := bufio.NewReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	for {
		line, err := in.ReadBytes('\n')
		if err != nil {
			// EOF: the client went away.
			return nil
		}
		line = trimEOL(line)
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			writeResp(out, rpcResponse{JSONRPC: "2.0", ID: nil,
				Error: &rpcError{Code: -32700, Message: "parse error"}})
			continue
		}
		// Notifications carry no id and get no reply.
		if req.ID == nil {
			handleNotification(req)
			continue
		}
		resp := handle(req)
		writeResp(out, resp)
	}
}

func writeResp(out *bufio.Writer, resp rpcResponse) {
	b, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = out.Write(append(b, '\n'))
	_ = out.Flush()
}

func handleNotification(req rpcRequest) {
	// notifications/initialized, notifications/cancelled: nothing to do.
}

func handle(req rpcRequest) rpcResponse {
	ok := func(result any) rpcResponse {
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
	}
	fail := func(code int, format string, args ...any) rpcResponse {
		return rpcResponse{JSONRPC: "2.0", ID: req.ID,
			Error: &rpcError{Code: code, Message: fmt.Sprintf(format, args...)}}
	}
	switch req.Method {
	case "initialize":
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &params)
		ver := params.ProtocolVersion
		if ver == "" {
			ver = "2025-06-18"
		}
		return ok(map[string]any{
			"protocolVersion": ver,
			"capabilities":    map[string]any{"tools": map[string]any{}, "resources": map[string]any{}},
			"serverInfo":      map[string]any{"name": "workspace-manager", "version": version},
			"instructions":    guideText,
		})
	case "ping":
		return ok(map[string]any{})
	case "tools/list":
		return ok(map[string]any{"tools": toolSchemas()})
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return fail(-32602, "invalid params: %v", err)
		}
		if params.Arguments == nil {
			params.Arguments = map[string]any{}
		}
		tool := findTool(params.Name)
		if tool == nil {
			return fail(-32602, "unknown tool %q", params.Name)
		}
		res, err := tool.Handler(params.Arguments)
		text := toText(res, err)
		out := toolResult{Content: []toolContent{{Type: "text", Text: text}}}
		if err != nil {
			out.IsError = true
		}
		return ok(out)
	case "resources/list":
		return ok(map[string]any{"resources": []any{map[string]any{
			"uri": guideURI, "name": "Workspace Manager guide", "mimeType": "text/markdown",
			"description": "How to manage the user's workspaces with these tools.",
		}}})
	case "resources/read":
		var params struct {
			URI string `json:"uri"`
		}
		_ = json.Unmarshal(req.Params, &params)
		if params.URI != guideURI {
			return fail(-32002, "unknown resource %q", params.URI)
		}
		return ok(map[string]any{"contents": []any{map[string]any{
			"uri": guideURI, "mimeType": "text/markdown", "text": guideText,
		}}})
	case "prompts/list":
		return ok(map[string]any{"prompts": []any{}})
	default:
		return fail(-32601, "method not found: %s", req.Method)
	}
}

func toText(res any, callErr error) string {
	if callErr != nil {
		return callErr.Error()
	}
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", res)
	}
	return string(b)
}

func trimEOL(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}
