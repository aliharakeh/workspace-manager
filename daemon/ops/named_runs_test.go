package ops_test

import (
	"testing"

	"workspace-manager/shared/api/types"
)

func TestNamedConfigurationOperations(t *testing.T) {
	o, rec, appID := setup(t)
	configs, err := o.RunConfigList(appID, 0)
	if err != nil || len(configs) != 0 {
		t.Fatalf("new app: %+v %v", configs, err)
	}
	// Reading a legacy configuration must not populate a new app.
	if _, err := o.RunConfigGet(appID, 0, "dev"); err != nil {
		t.Fatal(err)
	}
	configs, _ = o.RunConfigList(appID, 0)
	if len(configs) != 0 {
		t.Fatal("read created a configuration")
	}
	name := "My checks"
	mode := "sequential"
	ready := false
	cfg, err := o.RunConfigSave(appID, 0, "custom-checks", types.RunConfigSaveInput{Name: &name, Mode: &mode, DetectReady: &ready, Commands: []types.RunCommandInput{{Command: "echo checks"}}})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Name != name || cfg.Mode != mode || cfg.DetectReady {
		t.Fatalf("saved configuration: %+v", cfg)
	}
	if len(rec.changed) == 0 || rec.changed[len(rec.changed)-1] != "runConfig" {
		t.Fatal("save not announced")
	}
	blank := " "
	if _, err := o.RunConfigSave(appID, 0, "invalid", types.RunConfigSaveInput{Name: &blank}); err == nil {
		t.Fatal("blank name accepted")
	}
	app, _ := o.AppsGet(appID)
	other, err := o.AppsCreate(app.WorkspaceID, types.AppCreateInput{Name: "other", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.RunConfigList(other.ID, cfg.ConfigSetID); err == nil {
		t.Fatal("listed another app's configuration set")
	}
	if _, err := o.RunnerStart(other.ID, cfg.Kind, cfg.ConfigSetID); err == nil {
		t.Fatal("started another app's configuration set")
	}
	if _, err := o.RunConfigDelete(appID, 0, cfg.Kind); err != nil {
		t.Fatal(err)
	}
	configs, _ = o.RunConfigList(appID, 0)
	if len(configs) != 0 {
		t.Fatalf("delete left configurations: %+v", configs)
	}
	if rec.changed[len(rec.changed)-1] != "runConfig" {
		t.Fatal("delete not announced")
	}
}
