package ops

import (
	"fmt"
	"strings"

	"workspace-manager/server/db"

	"workspace-manager/api/types"
)

func (o *Ops) ConfigSetsList(appID int64) ([]types.ConfigSet, error) {
	if _, err := o.DB.GetAppT(o.Ctx(), appID); err != nil {
		return nil, err
	}
	return o.DB.ListConfigSetsByAppT(o.Ctx(), appID)
}

// appForConfigSet returns the app a config set belongs to, 0 when it cannot be
// read: a changed event then carries no id and clients reload the entity.
func (o *Ops) appForConfigSet(setID int64) int64 {
	set, err := o.DB.GetConfigSetT(o.Ctx(), setID)
	if err != nil {
		return 0
	}
	return set.AppID
}

func (o *Ops) ConfigSetsGetDetail(id int64) (types.ConfigSetDetail, error) {
	set, err := o.DB.GetConfigSetT(o.Ctx(), id)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	return o.DB.ConfigSetDetailT(o.Ctx(), set)
}

func (o *Ops) ConfigSetsCreate(appID int64, body types.ConfigSetCreateInput) (types.ConfigSet, error) {
	if _, err := o.DB.GetAppT(o.Ctx(), appID); err != nil {
		return types.ConfigSet{}, err
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return types.ConfigSet{}, fmt.Errorf("name is required")
	}
	if body.CopyFromID != nil && !db.HasAnyPart(body.Parts) {
		return types.ConfigSet{}, fmt.Errorf("Select at least one part to copy")
	}
	if body.CopyFromID != nil {
		source, err := o.DB.GetConfigSetT(o.Ctx(), *body.CopyFromID)
		if err != nil || source.AppID != appID {
			return types.ConfigSet{}, fmt.Errorf("copy_from_id must be a config set on this app")
		}
	}
	set, err := o.DB.CreateConfigSetT(o.Ctx(), appID, name)
	if err != nil {
		return types.ConfigSet{}, err
	}
	if body.CopyFromID != nil {
		if err := o.DB.CopyFrom(o.Ctx(), *body.CopyFromID, set.ID, body.Parts); err != nil {
			return types.ConfigSet{}, err
		}
	}
	if body.Activate == nil || *body.Activate {
		_, _ = o.DB.SetActiveConfigSetT(o.Ctx(), appID, set.ID)
	}
	o.changed("configSet", appID)
	return set, nil
}

func (o *Ops) ConfigSetsUpdate(id int64, body types.ConfigSetUpdateInput) (types.ConfigSet, error) {
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return types.ConfigSet{}, fmt.Errorf("name is required")
	}
	set, err := o.DB.UpdateConfigSetT(o.Ctx(), id, name)
	if err == nil {
		o.changed("configSet", set.AppID)
	}
	return set, err
}

func (o *Ops) ConfigSetsDelete(id int64) (types.Ok, error) {
	set, err := o.DB.GetConfigSetT(o.Ctx(), id)
	if err != nil {
		return types.Ok{}, err
	}
	siblings, err := o.DB.ListConfigSetsByAppT(o.Ctx(), set.AppID)
	if err != nil {
		return types.Ok{}, err
	}
	if len(siblings) <= 1 {
		return types.Ok{}, fmt.Errorf("Cannot delete the last config set")
	}
	app, _ := o.DB.GetAppT(o.Ctx(), set.AppID)
	if _, err := o.DB.DeleteConfigSet(o.Ctx(), id); err != nil {
		return types.Ok{}, err
	}
	if app.ActiveConfigSetID != nil && *app.ActiveConfigSetID == id {
		next, err := o.DB.ListConfigSetsByAppT(o.Ctx(), set.AppID)
		if err == nil && len(next) > 0 {
			_, _ = o.DB.SetActiveConfigSetT(o.Ctx(), set.AppID, next[0].ID)
		}
	}
	o.changed("configSet", set.AppID)
	return types.Ok{Ok: true}, nil
}

func (o *Ops) ConfigSetsActivate(id int64) (types.ConfigSetActivateResult, error) {
	set, err := o.DB.GetConfigSetT(o.Ctx(), id)
	if err != nil {
		return types.ConfigSetActivateResult{}, err
	}
	app, err := o.DB.SetActiveConfigSetT(o.Ctx(), set.AppID, set.ID)
	if err != nil {
		return types.ConfigSetActivateResult{}, err
	}
	o.changed("configSet", set.AppID)
	return types.ConfigSetActivateResult{ID: set.ID, AppID: set.AppID, Name: set.Name, App: app}, nil
}

func (o *Ops) ConfigSetsCopyFrom(id, sourceID int64, parts *types.CopyParts) (types.ConfigSet, error) {
	target, err := o.DB.GetConfigSetT(o.Ctx(), id)
	if err != nil {
		return types.ConfigSet{}, err
	}
	if !db.HasAnyPart(parts) {
		return types.ConfigSet{}, fmt.Errorf("Select at least one part to copy")
	}
	source, err := o.DB.GetConfigSetT(o.Ctx(), sourceID)
	if err != nil || source.AppID != target.AppID {
		return types.ConfigSet{}, fmt.Errorf("source_id must be a config set on the same app")
	}
	if err := o.DB.CopyFrom(o.Ctx(), sourceID, id, parts); err != nil {
		return types.ConfigSet{}, err
	}
	o.changed("configSet", target.AppID)
	return o.DB.GetConfigSetT(o.Ctx(), id)
}

func (o *Ops) EnvVarsList(appID int64) ([]types.EnvVar, error) {
	if _, err := o.DB.GetAppT(o.Ctx(), appID); err != nil {
		return nil, err
	}
	set, err := o.DB.ResolveActive(o.Ctx(), appID)
	if err != nil {
		return nil, err
	}
	return o.DB.ListEnvVarsT(o.Ctx(), set.ID)
}

func (o *Ops) EnvVarsCreate(appID int64, body types.EnvVarCreateInput) (types.EnvVar, error) {
	if _, err := o.DB.GetAppT(o.Ctx(), appID); err != nil {
		return types.EnvVar{}, err
	}
	key := strings.TrimSpace(body.Key)
	if key == "" {
		return types.EnvVar{}, fmt.Errorf("key is required")
	}
	set, err := o.DB.ResolveActive(o.Ctx(), appID)
	if err != nil {
		return types.EnvVar{}, err
	}
	value := ""
	if body.Value != nil {
		value = *body.Value
	}
	include := true
	if body.IncludeInAI != nil {
		include = *body.IncludeInAI
	}
	row, err := o.DB.CreateEnvVar(o.Ctx(), db.CreateEnvVarParams{
		ConfigSetID: set.ID, Key: key, Value: value, IncludeInAi: db.BoolInt(include),
	})
	if err != nil {
		return types.EnvVar{}, db.UniqueErr(err, "Env var key already exists for this config set")
	}
	o.changed("envVar", appID)
	return templateEnvVar(row), nil
}

func (o *Ops) EnvVarsUpdate(id int64, body types.EnvVarUpdateInput) (types.EnvVar, error) {
	existing, err := o.DB.GetEnvVar(o.Ctx(), id)
	if err != nil {
		return types.EnvVar{}, fmt.Errorf("Env var not found")
	}
	key, value, include := existing.Key, existing.Value, existing.IncludeInAi != 0
	if body.Key != nil {
		key = strings.TrimSpace(*body.Key)
		if key == "" {
			return types.EnvVar{}, fmt.Errorf("key cannot be empty")
		}
	}
	if body.Value != nil {
		value = *body.Value
	}
	if body.IncludeInAI != nil {
		include = *body.IncludeInAI
	}
	row, err := o.DB.UpdateEnvVar(o.Ctx(), db.UpdateEnvVarParams{
		Key: key, Value: value, IncludeInAi: db.BoolInt(include), ID: id,
	})
	if err != nil {
		return types.EnvVar{}, db.UniqueErr(err, "Env var key already exists for this config set")
	}
	o.changed("envVar", o.appForConfigSet(existing.ConfigSetID))
	return templateEnvVar(row), nil
}

func (o *Ops) EnvVarsDelete(id int64) (types.Ok, error) {
	existing, err := o.DB.GetEnvVar(o.Ctx(), id)
	if err != nil {
		return types.Ok{}, fmt.Errorf("Env var not found")
	}
	if _, err := o.DB.DeleteEnvVar(o.Ctx(), id); err != nil {
		return types.Ok{}, err
	}
	o.changed("envVar", o.appForConfigSet(existing.ConfigSetID))
	return types.Ok{Ok: true}, nil
}

func (o *Ops) TemplatesList(appID int64) ([]types.Template, error) {
	if _, err := o.DB.GetAppT(o.Ctx(), appID); err != nil {
		return nil, err
	}
	set, err := o.DB.ResolveActive(o.Ctx(), appID)
	if err != nil {
		return nil, err
	}
	return o.DB.ListTemplatesT(o.Ctx(), set.ID)
}

func (o *Ops) TemplatesCreate(appID int64, body types.TemplateCreateInput) (types.Template, error) {
	if _, err := o.DB.GetAppT(o.Ctx(), appID); err != nil {
		return types.Template{}, err
	}
	path := strings.ReplaceAll(strings.TrimSpace(body.FilePath), "\\", "/")
	if path == "" {
		return types.Template{}, fmt.Errorf("file_path is required")
	}
	set, err := o.DB.ResolveActive(o.Ctx(), appID)
	if err != nil {
		return types.Template{}, err
	}
	content := ""
	if body.Content != nil {
		content = *body.Content
	}
	include := true
	if body.IncludeInAI != nil {
		include = *body.IncludeInAI
	}
	row, err := o.DB.CreateTemplate(o.Ctx(), db.CreateTemplateParams{ConfigSetID: set.ID, FilePath: path, Content: content, IncludeInAi: db.BoolInt(include)})
	if err != nil {
		return types.Template{}, db.UniqueErr(err, "Template for this file already exists")
	}
	o.changed("template", appID)
	return templateResult(row), nil
}

func templateResult(row db.Template) types.Template {
	return types.Template{ID: row.ID, ConfigSetID: row.ConfigSetID, FilePath: row.FilePath, Content: row.Content, IncludeInAI: row.IncludeInAi != 0, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func templateEnvVar(row db.EnvVar) types.EnvVar {
	return db.EnvVarFrom(row)
}

func (o *Ops) TemplatesUpdate(id int64, body types.TemplateUpdateInput) (types.Template, error) {
	existing, err := o.DB.GetTemplate(o.Ctx(), id)
	if err != nil {
		return types.Template{}, fmt.Errorf("Template not found")
	}
	path, content, include := existing.FilePath, existing.Content, existing.IncludeInAi != 0
	if body.FilePath != nil {
		path = strings.ReplaceAll(strings.TrimSpace(*body.FilePath), "\\", "/")
		if path == "" {
			return types.Template{}, fmt.Errorf("file_path cannot be empty")
		}
	}
	if body.Content != nil {
		content = *body.Content
	}
	if body.IncludeInAI != nil {
		include = *body.IncludeInAI
	}
	row, err := o.DB.UpdateTemplate(o.Ctx(), db.UpdateTemplateParams{FilePath: path, Content: content, IncludeInAi: db.BoolInt(include), ID: id})
	if err != nil {
		return types.Template{}, db.UniqueErr(err, "Template for this file already exists")
	}
	o.changed("template", o.appForConfigSet(existing.ConfigSetID))
	return templateResult(row), nil
}

func (o *Ops) TemplatesDelete(id int64) (types.Ok, error) {
	existing, err := o.DB.GetTemplate(o.Ctx(), id)
	if err != nil {
		return types.Ok{}, fmt.Errorf("Template not found")
	}
	if _, err := o.DB.DeleteTemplate(o.Ctx(), id); err != nil {
		return types.Ok{}, err
	}
	o.changed("template", o.appForConfigSet(existing.ConfigSetID))
	return types.Ok{Ok: true}, nil
}
