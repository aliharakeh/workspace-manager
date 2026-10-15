package db

import (
	"context"
	"testing"

	"workspace-manager/internal/api/types"
)

func TestRunConfigMigrationAndCopy(t *testing.T) {
	t.Setenv("WM_DATA_DIR", t.TempDir())
	d, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	ctx := context.Background()
	ws, err := d.CreateWorkspaceT(ctx, "ws", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	app, err := d.CreateAppT(ctx, ws.ID, "app", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source, err := d.ResolveActive(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	mode := "sequential"
	dev, err := d.UpsertRunConfig(ctx, source.ID, types.KindDev, &mode, []types.RunCommandInput{{Command: "npm run dev"}})
	if err != nil {
		t.Fatal(err)
	}
	// Recreate a legacy run row, including its original command ids.
	if _, err := d.SQL.Exec(`UPDATE run_configs SET kind = 'run' WHERE id = ?;`, dev.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.SQL.Exec(`DELETE FROM run_configs WHERE config_set_id = ? AND kind = 'prod'`, source.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := applySchema(d.SQL); err != nil {
			t.Fatal(err)
		}
	}
	detail, err := d.ConfigSetDetailT(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	if detail.DevConfig == nil || detail.DevConfig.ID != dev.ID || detail.DevConfig.Mode != mode || len(detail.DevConfig.Commands) != 1 || detail.DevConfig.Commands[0].ID != dev.Commands[0].ID {
		t.Fatalf("dev not preserved: %+v", detail.DevConfig)
	}
	if detail.ProdConfig == nil || detail.ProdConfig.Mode != "parallel" || len(detail.ProdConfig.Commands) != 0 {
		t.Fatalf("prod should start empty: %+v", detail.ProdConfig)
	}
	if _, err := d.UpsertRunConfig(ctx, source.ID, types.KindProd, nil, []types.RunCommandInput{{Command: "npm start"}}); err != nil {
		t.Fatal(err)
	}
	if err := applySchema(d.SQL); err != nil {
		t.Fatal(err)
	}
	target, err := d.CreateConfigSetT(ctx, app.ID, "copy")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.CopyFrom(ctx, source.ID, target.ID, &types.CopyParts{Env: false, Templates: false, Dev: false, Prod: true, Build: false, Setup: false}); err != nil {
		t.Fatal(err)
	}
	copy, err := d.ConfigSetDetailT(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(copy.DevConfig.Commands) != 0 || len(copy.ProdConfig.Commands) != 1 || copy.ProdConfig.Commands[0].Command != "npm start" {
		t.Fatalf("prod-only copy: %+v", copy)
	}
	if err := d.CopyFrom(ctx, source.ID, target.ID, nil); err != nil {
		t.Fatal(err)
	}
	copy, err = d.ConfigSetDetailT(ctx, target)
	if err != nil || len(copy.DevConfig.Commands) != 1 || len(copy.ProdConfig.Commands) != 1 {
		t.Fatalf("full copy: %+v, %v", copy, err)
	}
}
