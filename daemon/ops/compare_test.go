package ops_test

import (
	"encoding/json"
	"strings"
	"testing"

	"workspace-manager/daemon/ops"
	"workspace-manager/shared/api/types"
)

func lbl(s string) *string { return &s }

func cmds(items ...[2]string) *types.RunConfig {
	rc := &types.RunConfig{Mode: "parallel"}
	for _, it := range items {
		c := types.RunCommand{Command: it[1]}
		if it[0] != "" {
			c.Label = lbl(it[0])
		}
		rc.Commands = append(rc.Commands, c)
	}
	return rc
}

func fixture() (types.ConfigSetDetail, types.ConfigSetDetail) {
	l := types.ConfigSetDetail{
		EnvVars: []types.EnvVar{
			{Key: "A", Value: "1", IncludeInAI: true},
			{Key: "B", Value: "left", IncludeInAI: true},
			{Key: "ONLYL", Value: "x", IncludeInAI: true},
			{Key: "SECRET", Value: "hunter2", IncludeInAI: false},
			{Key: "SAMESECRET", Value: "s", IncludeInAI: false},
		},
		Templates: []types.Template{
			{FilePath: ".env", Content: "a", IncludeInAI: true},
			{FilePath: "k.pem", Content: "PRIVATE", IncludeInAI: false},
		},
		DevConfig: cmds([2]string{"web", "npm run dev"}, [2]string{"", "echo 1"}),
	}
	r := types.ConfigSetDetail{
		EnvVars: []types.EnvVar{
			{Key: "A", Value: "1", IncludeInAI: true},
			{Key: "B", Value: "right", IncludeInAI: true},
			{Key: "ONLYR", Value: "y", IncludeInAI: true},
			{Key: "SECRET", Value: "other", IncludeInAI: false},
			{Key: "SAMESECRET", Value: "s", IncludeInAI: false},
		},
		Templates: []types.Template{
			{FilePath: ".env", Content: "b", IncludeInAI: true},
			{FilePath: "k.pem", Content: "OTHERPRIVATE", IncludeInAI: false},
			{FilePath: "new.txt", Content: "n", IncludeInAI: true},
		},
		DevConfig: cmds([2]string{"web", "npm start"}, [2]string{"", "echo 1"}, [2]string{"", "echo 2"}),
	}
	return l, r
}

func TestCompareDetails(t *testing.T) {
	l, r := fixture()
	got := ops.CompareDetails(l, r, false)
	want := map[string]string{
		"A": types.DiffSame, "B": types.DiffDifferent, "ONLYL": types.DiffOnlyLeft,
		"ONLYR": types.DiffOnlyRight, "SECRET": types.DiffDifferent, "SAMESECRET": types.DiffSame,
	}
	for _, e := range got.Env {
		if want[e.Key] != e.Status {
			t.Errorf("env %s: %s, want %s", e.Key, e.Status, want[e.Key])
		}
		if e.Redacted {
			t.Errorf("env %s redacted for the GUI", e.Key)
		}
	}
	if len(got.Env) != len(want) {
		t.Errorf("env entries %d, want %d", len(got.Env), len(want))
	}
	if got.Templates[0].Status != types.DiffDifferent || *got.Templates[0].Left != "a" || *got.Templates[0].Right != "b" {
		t.Errorf("template .env: %+v", got.Templates[0])
	}
	if got.Templates[2].Status != types.DiffOnlyRight || got.Templates[2].Left != nil {
		t.Errorf("template new.txt: %+v", got.Templates[2])
	}

	var dev types.ConfigSetCommandListDiff
	for _, c := range got.Commands {
		if c.Kind == types.KindDev {
			dev = c
		}
	}
	if dev.Status != types.DiffDifferent || len(dev.Commands) != 3 {
		t.Fatalf("dev: %+v", dev)
	}
	if dev.Commands[0].Status != types.DiffChanged || dev.Commands[1].Status != types.DiffSame ||
		dev.Commands[2].Status != types.DiffOnlyRight {
		t.Errorf("dev commands: %+v", dev.Commands)
	}
	for _, c := range got.Commands {
		if c.Kind == types.KindProd && (c.Status != types.DiffSame || len(c.Commands) != 0) {
			t.Errorf("empty prod lists should be same: %+v", c)
		}
	}
}

func TestCompareRedactsForAgents(t *testing.T) {
	l, r := fixture()
	got := ops.CompareDetails(l, r, true)
	raw, _ := json.Marshal(got)
	for _, secret := range []string{"hunter2", "other", "PRIVATE", "OTHERPRIVATE"} {
		if strings.Contains(string(raw), secret) {
			t.Errorf("agent result leaks %q: %s", secret, raw)
		}
	}
	for _, e := range got.Env {
		switch e.Key {
		case "SECRET":
			if !e.Redacted || e.Status != types.DiffDifferent || e.Left != nil || e.Right != nil {
				t.Errorf("SECRET: %+v", e)
			}
		case "SAMESECRET":
			if !e.Redacted || e.Status != types.DiffSame {
				t.Errorf("SAMESECRET: %+v", e)
			}
		case "B":
			if e.Redacted || *e.Left != "left" {
				t.Errorf("B: %+v", e)
			}
		}
	}
	if !got.Templates[1].Redacted || got.Templates[1].Status != types.DiffDifferent {
		t.Errorf("k.pem: %+v", got.Templates[1])
	}
}

func TestCompareRequiresSameApp(t *testing.T) {
	o, _, appID := setup(t)
	sets, err := o.ConfigSetsList(appID)
	if err != nil || len(sets) == 0 {
		t.Fatalf("sets: %v", err)
	}
	if _, err := o.ConfigSetsCompare(sets[0].ID, sets[0].ID, true); err != nil {
		t.Fatalf("self compare: %v", err)
	}
	ws, _ := o.WorkspacesCreate(types.WorkspaceCreateInput{Name: "w2"})
	app2, err := o.AppsCreate(ws.ID, types.AppCreateInput{Name: "app2", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	other, _ := o.ConfigSetsList(app2.ID)
	if _, err := o.ConfigSetsCompare(sets[0].ID, other[0].ID, false); err == nil {
		t.Fatal("cross-app compare should fail")
	}
}
