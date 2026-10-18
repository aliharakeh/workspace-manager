package handlers

import (
	"workspace-manager/shared/api"
	"workspace-manager/shared/api/types"
)

// workspaceIO serves workspace export and import. Only the GUI may receive
// secrets in an export; the ops layer downgrades an agent's request.
func (r *Registry) workspaceIO() {
	o := r.Ops

	r.handle(api.MethodWorkspacesExport, kindTyped(func(kind string, in struct {
		ID      int64                         `json:"id"`
		Options types.WorkspaceExportOptions `json:"options"`
	}) (types.WorkspaceBundle, error) {
		return o.WorkspacesExport(in.ID, in.Options, kind == ClientGUI)
	}))
	r.handle(api.MethodWorkspacesImport, typed(func(in struct {
		Bundle  types.WorkspaceBundle        `json:"bundle"`
		Options types.WorkspaceImportOptions `json:"options"`
	}) (types.WorkspaceImportResult, error) {
		return o.WorkspacesImport(in.Bundle, in.Options)
	}))
}
