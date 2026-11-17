package handlers

import (
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

func (r *Registry) notes() {
	r.handle(api.MethodNotesList, typed(func(in struct {
		AppID int64 `json:"appID"`
	}) ([]types.Note, error) { return r.Ops.NotesList(in.AppID) }))
	r.handle(api.MethodNotesSave, typed(func(in struct {
		AppID int64               `json:"appID"`
		ID    int64               `json:"id"`
		Body  types.NoteSaveInput `json:"body"`
	}) (types.Note, error) {
		return r.Ops.NotesSave(in.AppID, in.ID, in.Body)
	}))
	r.handle(api.MethodNotesDelete, typed(func(in struct {
		AppID int64 `json:"appID"`
		ID    int64 `json:"id"`
	}) (types.Ok, error) {
		return r.Ops.NotesDelete(in.AppID, in.ID)
	}))
	r.handle(api.MethodNotesImage, typed(func(in struct {
		AppID int64 `json:"appID"`
		ID    int64 `json:"id"`
	}) (string, error) {
		return r.Ops.NotesImage(in.AppID, in.ID)
	}))
}
