package ops_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"workspace-manager/daemon/db"
	"workspace-manager/daemon/ops"
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

func noteImage(t *testing.T) string {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

func requireRemoved(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("image still exists at %s: %v", path, err)
	}
}

func TestNotesPersistenceAndImages(t *testing.T) {
	o, rec, appID := setup(t)
	encoded := noteImage(t)
	input := types.NoteSaveInput{Title: "  Release checklist  ", Content: "Context", Items: []types.NoteItem{{Text: "Ship", Done: true}, {Text: "Verify", Done: false}}, ImageData: &encoded}
	n, err := o.NotesSave(appID, 0, input)
	if err != nil {
		t.Fatal(err)
	}
	if n.Title != "Release checklist" || n.Content != "Context" || !n.Items[0].Done || n.Items[1].Done {
		t.Fatalf("bad note: %+v", n)
	}
	if rec.changed[len(rec.changed)-1] != "note" {
		t.Fatal("write not announced")
	}
	if !strings.HasPrefix(n.ImagePath, filepath.Join(db.DataDir(), "images", "notes")) {
		t.Fatalf("unmanaged path: %s", n.ImagePath)
	}
	first := n.ImagePath
	if _, err := os.Stat(first); err != nil {
		t.Fatal(err)
	}
	url, err := o.NotesImage(appID, n.ID)
	if err != nil || url != "data:image/png;base64,"+encoded {
		t.Fatalf("image read: %s %v", url, err)
	}

	// Closing and opening the database runs startup schema upgrades as well.
	if err := o.DB.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := db.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	o = ops.New(reopened, func(int64, any) {}, func(string, string) {})
	notes, err := o.NotesList(appID)
	if err != nil || len(notes) != 1 || !notes[0].Items[0].Done {
		t.Fatalf("lost notes after restart: %+v %v", notes, err)
	}
	raw, _ := json.Marshal(notes)
	if !bytes.Contains(raw, []byte(`"done":true`)) || !bytes.Contains(raw, []byte(`"done":false`)) {
		t.Fatalf("completion missing from JSON: %s", raw)
	}

	// Omitting image_data keeps the original copy, including during checklist edits.
	input.ImageData = nil
	input.Items[1].Done = true
	n, err = o.NotesSave(appID, n.ID, input)
	if err != nil || n.ImagePath != first {
		t.Fatalf("image not retained: %+v %v", n, err)
	}
	input.ImageData = &encoded
	n, err = o.NotesSave(appID, n.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	requireRemoved(t, first)
	second := n.ImagePath
	empty := ""
	input.ImageData = &empty
	n, err = o.NotesSave(appID, n.ID, input)
	if err != nil || n.ImagePath != "" {
		t.Fatalf("image not removed: %+v %v", n, err)
	}
	requireRemoved(t, second)
	input.ImageData = &encoded
	n, err = o.NotesSave(appID, n.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.NotesDelete(appID, n.ID); err != nil {
		t.Fatal(err)
	}
	requireRemoved(t, n.ImagePath)
	notes, err = o.NotesList(appID)
	if err != nil || len(notes) != 0 {
		t.Fatalf("delete left notes: %+v %v", notes, err)
	}
}

func TestNotesValidationAndAppIsolation(t *testing.T) {
	o, _, appID := setup(t)
	for _, input := range []types.NoteSaveInput{
		{Title: " "},
		{Title: "valid", Items: []types.NoteItem{{Text: " "}}},
		{Title: "valid", Items: make([]types.NoteItem, 501)},
		{Title: "valid", Content: strings.Repeat("a", 100001)},
	} {
		if _, err := o.NotesSave(appID, 0, input); api.KindOf(err) != api.KindInvalid {
			t.Fatalf("invalid note accepted: %v", err)
		}
	}
	for _, encoded := range []string{"not base64", base64.StdEncoding.EncodeToString([]byte("<svg/>")), strings.Repeat("a", base64.StdEncoding.EncodedLen(5*1024*1024)+1)} {
		if _, err := o.NotesSave(appID, 0, types.NoteSaveInput{Title: "image", ImageData: &encoded}); api.KindOf(err) != api.KindInvalid {
			t.Fatalf("invalid image accepted: %v", err)
		}
	}
	n, err := o.NotesSave(appID, 0, types.NoteSaveInput{Title: "Owned"})
	if err != nil {
		t.Fatal(err)
	}
	app, _ := o.AppsGet(appID)
	other, err := o.AppsCreate(app.WorkspaceID, types.AppCreateInput{Name: "other", ProjectPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.NotesSave(other.ID, n.ID, types.NoteSaveInput{Title: "overwrite"}); api.KindOf(err) != api.KindNotFound {
		t.Fatalf("cross-app write: %v", err)
	}
	if _, err := o.NotesDelete(other.ID, n.ID); api.KindOf(err) != api.KindNotFound {
		t.Fatalf("cross-app delete: %v", err)
	}
	if _, err := o.NotesImage(other.ID, n.ID); api.KindOf(err) != api.KindNotFound {
		t.Fatalf("cross-app read: %v", err)
	}
	notes, err := o.NotesList(other.ID)
	if err != nil || len(notes) != 0 {
		t.Fatalf("cross-app list: %+v %v", notes, err)
	}
	if _, err := o.NotesSave(999999, 0, types.NoteSaveInput{Title: "orphan"}); err == nil {
		t.Fatal("saved to nonexistent app")
	}
}

func TestNotesImagesCleanedOnAppAndWorkspaceDeletion(t *testing.T) {
	for _, workspace := range []bool{false, true} {
		name := "app"
		if workspace {
			name = "workspace"
		}
		t.Run(name, func(t *testing.T) {
			o, _, appID := setup(t)
			encoded := noteImage(t)
			n, err := o.NotesSave(appID, 0, types.NoteSaveInput{Title: "image", ImageData: &encoded})
			if err != nil {
				t.Fatal(err)
			}
			app, _ := o.AppsGet(appID)
			if workspace {
				_, err = o.WorkspacesDelete(app.WorkspaceID)
			} else {
				_, err = o.AppsDelete(appID)
			}
			if err != nil {
				t.Fatal(err)
			}
			requireRemoved(t, n.ImagePath)
			var count int
			if err := o.DB.SQL.QueryRow("SELECT count(*) FROM notes").Scan(&count); err != nil || count != 0 {
				t.Fatalf("orphan note rows: %d %v", count, err)
			}
		})
	}
}
