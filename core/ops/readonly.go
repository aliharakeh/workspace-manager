package ops

import (
	"workspace-manager/api/lib"
	"workspace-manager/api/types"
)

// GetEnvVarByID returns one env var, so handlers can check whether it is
// visible to the calling client.
func (o *Ops) GetEnvVarByID(id int64) (types.EnvVar, error) {
	row, err := o.DB.GetEnvVar(o.Ctx(), id)
	if err != nil {
		return types.EnvVar{}, err
	}
	return templateEnvVar(row), nil
}

// GetTemplateByID returns one template, for the same reason.
func (o *Ops) GetTemplateByID(id int64) (types.Template, error) {
	row, err := o.DB.GetTemplate(o.Ctx(), id)
	if err != nil {
		return types.Template{}, err
	}
	return templateResult(row), nil
}

func (o *Ops) PackageScriptsList(appID int64) (types.PackageScripts, error) {
	app, err := o.DB.GetAppT(o.Ctx(), appID)
	if err != nil {
		return types.PackageScripts{}, err
	}
	return lib.ReadPackageScripts(app.ProjectPath), nil
}
