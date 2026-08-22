package services

import (
	"os"
	"path/filepath"
	"testing"

	"wails_backend/internal/types"
)

func TestAppAIStatePatch(t *testing.T) {
	label := "web"
	detail := types.ConfigSetDetail{
		EnvVars:   []types.EnvVar{{Key: "PORT", Value: "3000"}},
		Templates: []types.Template{{FilePath: ".env", Content: "PORT=3000"}},
		RunConfig: &types.RunConfig{
			Mode:     "parallel",
			Commands: []types.RunCommand{{Label: &label, Command: "npm run dev"}},
		},
	}
	s := newAppAIState(detail, ".")
	s.updateVar("PORT", "5173")
	s.updateVar("HOST", "localhost")
	s.deleteVar("HOST")
	s.updateTemplate(".env", "PORT={{PORT}}")
	s.updateRun(updateRunIn{Mode: ptr("sequential")})
	p := s.patch()
	if p.Env == nil || len(p.Env.Upsert) != 1 || p.Env.Upsert[0].Key != "PORT" {
		t.Fatalf("env upsert: %+v", p.Env)
	}
	if len(p.Templates) != 1 || p.Templates[0].Content != "PORT={{PORT}}" {
		t.Fatalf("templates: %+v", p.Templates)
	}
	if p.Run == nil || *p.Run.Mode != "sequential" {
		t.Fatalf("run: %+v", p.Run)
	}
	if err, _ := s.updateTemplate("nope", "x")["error"].(string); err == "" {
		t.Fatal("expected unknown template error")
	}
}

func TestAppAIStateReadFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.ts"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := newAppAIState(types.ConfigSetDetail{}, dir)
	got := s.readFile("keep.ts")
	if got["content"] != "hello" || got["file_path"] != "keep.ts" {
		t.Fatalf("read: %+v", got)
	}
	if _, ok := s.readFile("../x")["error"]; !ok {
		t.Fatal("expected escape error")
	}
}

func ptr(s string) *string { return &s }
