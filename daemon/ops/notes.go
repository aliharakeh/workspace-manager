package ops

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"workspace-manager/daemon/db"
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

const maxNoteImage = 5 * 1024 * 1024

func noteImagePath(name string) string {
	if name == "" {
		return ""
	}
	return filepath.Join(db.DataDir(), "images", "notes", name)
}

func noteFromRow(row db.Note) (types.Note, error) {
	n := types.Note{ID: row.ID, AppID: row.AppID, Title: row.Title, Content: row.Content,
		ImagePath: noteImagePath(row.Image), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Items: []types.NoteItem{}}
	err := json.Unmarshal([]byte(row.Items), &n.Items)
	return n, err
}

func (o *Ops) NotesList(appID int64) ([]types.Note, error) {
	if _, err := o.AppsGet(appID); err != nil {
		return nil, err
	}
	rows, err := o.DB.ListNotes(o.Ctx(), appID)
	if err != nil {
		return nil, err
	}
	out := make([]types.Note, 0, len(rows))
	for _, row := range rows {
		n, err := noteFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func removeNoteImage(name string) error {
	if name == "" {
		return nil
	}
	// Only daemon-generated basenames may be used, never caller-supplied paths.
	if filepath.Base(name) != name || !strings.HasPrefix(name, "note-") {
		return fmt.Errorf("invalid managed image name")
	}
	err := os.Remove(noteImagePath(name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func storeNoteImage(encoded string) (string, error) {
	if len(encoded) > base64.StdEncoding.EncodedLen(maxNoteImage) {
		return "", api.Errf(api.KindInvalid, "image must be 5 MB or smaller")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(data) == 0 || len(data) > maxNoteImage {
		return "", api.Errf(api.KindInvalid, "invalid base64 image (maximum 5 MB)")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 40000000 {
		return "", api.Errf(api.KindInvalid, "use a PNG, JPEG or GIF image up to 40 megapixels")
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return "", api.Errf(api.KindInvalid, "invalid image: %v", err)
	}
	dir := filepath.Join(db.DataDir(), "images", "notes")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, "note-*."+format)
	if err != nil {
		return "", err
	}
	name := filepath.Base(f.Name())
	_, err = f.Write(data)
	closeErr := f.Close()
	if err = errors.Join(err, closeErr); err != nil {
		return "", errors.Join(err, removeNoteImage(name))
	}
	return name, nil
}

func (o *Ops) NotesSave(appID, id int64, body types.NoteSaveInput) (types.Note, error) {
	o.notesMu.Lock()
	defer o.notesMu.Unlock()
	if _, err := o.AppsGet(appID); err != nil {
		return types.Note{}, err
	}
	body.Title = strings.TrimSpace(body.Title)
	if body.Title == "" || len(body.Title) > 300 {
		return types.Note{}, api.Errf(api.KindInvalid, "title is required (maximum 300 bytes)")
	}
	if len(body.Content) > 100000 || len(body.Items) > 500 {
		return types.Note{}, api.Errf(api.KindInvalid, "note is too large (100,000 bytes or 500 checklist items)")
	}
	if body.Items == nil {
		body.Items = []types.NoteItem{}
	}
	for i := range body.Items {
		body.Items[i].Text = strings.TrimSpace(body.Items[i].Text)
		if body.Items[i].Text == "" || len(body.Items[i].Text) > 2000 {
			return types.Note{}, api.Errf(api.KindInvalid, "checklist items need text (maximum 2,000 bytes)")
		}
	}
	oldImage := ""
	if id != 0 {
		row, err := o.DB.GetNote(o.Ctx(), db.GetNoteParams{ID: id, AppID: appID})
		if err != nil {
			return types.Note{}, api.Errf(api.KindNotFound, "note not found for this app")
		}
		oldImage = row.Image
	}
	nextImage := oldImage
	if body.ImageData != nil {
		nextImage = ""
		if *body.ImageData != "" {
			var err error
			nextImage, err = storeNoteImage(*body.ImageData)
			if err != nil {
				return types.Note{}, err
			}
		}
	}
	items, err := json.Marshal(body.Items)
	if err != nil {
		return types.Note{}, err
	}
	var row db.Note
	if id == 0 {
		row, err = o.DB.CreateNote(o.Ctx(), db.CreateNoteParams{AppID: appID, Title: body.Title, Content: body.Content, Items: string(items), Image: nextImage})
	} else {
		row, err = o.DB.UpdateNote(o.Ctx(), db.UpdateNoteParams{ID: id, AppID: appID, Title: body.Title, Content: body.Content, Items: string(items), Image: nextImage})
	}
	if err != nil {
		if nextImage != oldImage {
			err = errors.Join(err, removeNoteImage(nextImage))
		}
		return types.Note{}, err
	}
	o.changed("note", appID)
	if oldImage != nextImage {
		if err := removeNoteImage(oldImage); err != nil {
			return types.Note{}, fmt.Errorf("note saved but old image cleanup failed: %w", err)
		}
	}
	return noteFromRow(row)
}

func (o *Ops) NotesDelete(appID, id int64) (types.Ok, error) {
	o.notesMu.Lock()
	defer o.notesMu.Unlock()
	row, err := o.DB.GetNote(o.Ctx(), db.GetNoteParams{ID: id, AppID: appID})
	if err != nil {
		return types.Ok{}, api.Errf(api.KindNotFound, "note not found for this app")
	}
	if _, err := o.DB.DeleteNote(o.Ctx(), db.DeleteNoteParams{ID: id, AppID: appID}); err != nil {
		return types.Ok{}, err
	}
	o.changed("note", appID)
	if err := removeNoteImage(row.Image); err != nil {
		return types.Ok{}, fmt.Errorf("note deleted but image cleanup failed: %w", err)
	}
	return types.Ok{Ok: true}, nil
}

// NotesImage reads by note identity, never by an arbitrary file path.
func (o *Ops) NotesImage(appID, id int64) (string, error) {
	o.notesMu.Lock()
	defer o.notesMu.Unlock()
	row, err := o.DB.GetNote(o.Ctx(), db.GetNoteParams{ID: id, AppID: appID})
	if err != nil {
		return "", api.Errf(api.KindNotFound, "note not found for this app")
	}
	if row.Image == "" {
		return "", nil
	}
	data, err := os.ReadFile(noteImagePath(row.Image))
	if err != nil {
		return "", err
	}
	mime := "image/" + strings.TrimPrefix(filepath.Ext(row.Image), ".")
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func (o *Ops) noteImagesForApps(appIDs []int64) ([]string, error) {
	out := []string{}
	for _, id := range appIDs {
		rows, err := o.DB.ListNotes(o.Ctx(), id)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if row.Image != "" {
				out = append(out, row.Image)
			}
		}
	}
	return out, nil
}

func removeNoteImages(names []string) error {
	var err error
	for _, name := range names {
		err = errors.Join(err, removeNoteImage(name))
	}
	return err
}
