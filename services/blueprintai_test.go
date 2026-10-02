package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"workspace-manager/types"
)

func TestValidateBlueprintProposal(t *testing.T) {
	ok := proposeBlueprintIn{
		Name:         "  Go service ",
		SampleName:   "my-service",
		CreateFolder: true,
		Commands: []proposeCommandIn{
			{Label: " Init ", Command: " go mod init {{folder_name}} "},
			{Label: "", Command: "   "},
			{Label: "", Command: "go mod tidy"},
		},
	}
	bp, problems := validateBlueprintProposal(ok, "linux")
	if len(problems) > 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}
	if bp.Name != "Go service" || len(bp.Commands) != 2 {
		t.Fatalf("not cleaned: %+v", bp)
	}
	if bp.Commands[0].Command != "go mod init {{folder_name}}" || bp.Commands[0].Label == nil || *bp.Commands[0].Label != "Init" {
		t.Fatalf("first command not cleaned: %+v", bp.Commands[0])
	}
	if bp.Commands[1].Label != nil {
		t.Fatalf("blank label should be nil, got %q", *bp.Commands[1].Label)
	}

	cmd := func(c string) []proposeCommandIn { return []proposeCommandIn{{Command: c}} }
	cases := []struct {
		name string
		goos string
		in   proposeBlueprintIn
		want string // substring of one problem
	}{
		{"missing name", "linux", proposeBlueprintIn{CreateFolder: true, Commands: cmd("npm i")}, "name is required"},
		{"long name", "linux", proposeBlueprintIn{Name: strings.Repeat("x", 81), CreateFolder: true, Commands: cmd("npm i")}, "at most 80"},
		{"no commands", "linux", proposeBlueprintIn{Name: "a", CreateFolder: true, Commands: cmd("  ")}, "at least one command"},
		{"unknown placeholder", "linux", proposeBlueprintIn{Name: "a", CreateFolder: true, Commands: cmd("echo {{project}}")}, "{{project}}"},
		{"placeholder in sample", "linux", proposeBlueprintIn{Name: "a", SampleName: "{{app_name}}", CreateFolder: true, Commands: cmd("npm i")}, "sample_name"},
		{"multiline", "linux", proposeBlueprintIn{Name: "a", CreateFolder: true, Commands: cmd("npm i\nnpm run build")}, "single line"},
		{"unix command on windows", "windows", proposeBlueprintIn{Name: "a", CreateFolder: true, Commands: cmd("npm i && rm -rf node_modules")}, `"rm"`},
		{"mkdir -p on windows", "windows", proposeBlueprintIn{Name: "a", CreateFolder: true, Commands: cmd("mkdir -p src/lib")}, "mkdir -p"},
		{"windows command on unix", "linux", proposeBlueprintIn{Name: "a", CreateFolder: true, Commands: cmd("copy a b")}, `"copy"`},
		{"nothing creates the folder", "linux", proposeBlueprintIn{Name: "a", CreateFolder: false, Commands: cmd("npm init -y")}, "nothing creates the app folder"},
		{"too many commands", "linux", proposeBlueprintIn{Name: "a", CreateFolder: true, Commands: func() []proposeCommandIn {
			out := make([]proposeCommandIn, maxBlueprintCommands+1)
			for i := range out {
				out[i].Command = "true"
			}
			return out
		}()}, "at most 30 commands"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, problems := validateBlueprintProposal(tc.in, tc.goos)
			for _, p := range problems {
				if strings.Contains(p, tc.want) {
					return
				}
			}
			t.Fatalf("want a problem containing %q, got %v", tc.want, problems)
		})
	}

	// Valid forms that must not be flagged.
	for _, tc := range []struct {
		name string
		goos string
		in   proposeBlueprintIn
	}{
		{"folder created by command", "linux", proposeBlueprintIn{Name: "a", CreateFolder: false, Commands: cmd("git clone https://x/y.git {{ folder_name }}")}},
		{"app_dir counts", "linux", proposeBlueprintIn{Name: "a", CreateFolder: false, Commands: cmd("mkdir {{app_dir}}")}},
		{"cmd builtin on windows", "windows", proposeBlueprintIn{Name: "a", CreateFolder: true, Commands: cmd("mkdir src && echo hi")}},
		{"cat is fine as an argument", "windows", proposeBlueprintIn{Name: "a", CreateFolder: true, Commands: cmd("npm install cat")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, problems := validateBlueprintProposal(tc.in, tc.goos); len(problems) > 0 {
				t.Fatalf("unexpected problems: %v", problems)
			}
		})
	}
}

// completion builds an OpenAI chat.completion body: a tool call when toolArgs is
// set, otherwise plain text.
func completion(t *testing.T, toolArgs any, text string) string {
	t.Helper()
	msg := map[string]any{"role": "assistant"}
	finish := "stop"
	if toolArgs != nil {
		args, err := json.Marshal(toolArgs)
		if err != nil {
			t.Fatal(err)
		}
		msg["content"] = nil
		msg["tool_calls"] = []any{map[string]any{
			"id": "call_1", "type": "function",
			"function": map[string]any{"name": "propose_blueprint", "arguments": string(args)},
		}}
		finish = "tool_calls"
	} else {
		msg["content"] = text
	}
	body, err := json.Marshal(map[string]any{
		"id": "1", "object": "chat.completion", "created": 1, "model": "m",
		"choices": []any{map[string]any{"index": 0, "finish_reason": finish, "message": msg}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

// fakeLLM serves the given responses in order and records each request body.
func fakeLLM(t *testing.T, responses []string) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		i := len(bodies)
		bodies = append(bodies, string(b))
		mu.Unlock()
		if i >= len(responses) {
			http.Error(w, "unexpected request", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[i]))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), bodies...)
	}
}

func TestProposeBlueprintCorrectsInvalidProposal(t *testing.T) {
	bad := map[string]any{
		"name": "Go service", "description": "", "sample_name": "my-service", "create_folder": true,
		"commands": []any{map[string]any{"label": "Init", "command": "go mod init {{project}}"}},
	}
	good := map[string]any{
		"name": "Go service", "description": "A Go module.", "sample_name": "my-service", "create_folder": true,
		"commands": []any{
			map[string]any{"label": "Init", "command": "go mod init {{folder_name}}"},
			map[string]any{"label": "", "command": "go mod tidy"},
		},
	}
	srv, requests := fakeLLM(t, []string{
		completion(t, bad, ""),
		completion(t, good, ""),
		completion(t, nil, "Needs Go installed."),
	})

	draft := &types.BlueprintInput{Name: "Go service", CreateFolder: true, Commands: []types.BlueprintCommand{{Command: "go version"}}}
	res, err := proposeBlueprint(context.Background(),
		AIProviderConfig{Provider: "custom-bp-loop", BaseURL: srv.URL + "/v1", Model: "m", APIKey: "k"},
		"linux", "make a go service", draft)
	if err != nil {
		t.Fatal(err)
	}
	if res.Blueprint.Name != "Go service" || len(res.Blueprint.Commands) != 2 || !res.Blueprint.CreateFolder {
		t.Fatalf("unexpected blueprint: %+v", res.Blueprint)
	}
	if res.Blueprint.Commands[0].Command != "go mod init {{folder_name}}" {
		t.Fatalf("the corrected proposal should win, got %+v", res.Blueprint.Commands)
	}
	if res.Message != "Needs Go installed." {
		t.Fatalf("message = %q", res.Message)
	}

	reqs := requests()
	if len(reqs) != 3 {
		t.Fatalf("want 3 model requests, got %d", len(reqs))
	}
	if !strings.Contains(reqs[0], "propose_blueprint") {
		t.Fatal("the tool was not offered to the model")
	}
	if !strings.Contains(reqs[0], "go version") {
		t.Fatal("the current draft was not given to the model")
	}
	if !strings.Contains(reqs[1], "unknown placeholder") {
		t.Fatal("validation errors were not returned to the model")
	}
}

func TestProposeBlueprintWithoutProposal(t *testing.T) {
	srv, _ := fakeLLM(t, []string{completion(t, nil, "Which framework do you want?")})
	_, err := proposeBlueprint(context.Background(),
		AIProviderConfig{Provider: "custom-bp-none", BaseURL: srv.URL + "/v1", Model: "m", APIKey: "k"},
		"linux", "make an app", nil)
	if err == nil || !strings.Contains(err.Error(), "Which framework do you want?") {
		t.Fatalf("want the model's text in the error, got %v", err)
	}

	if _, err := proposeBlueprint(context.Background(), AIProviderConfig{}, "linux", "  ", nil); err == nil {
		t.Fatal("blank instruction should be rejected")
	}
}
