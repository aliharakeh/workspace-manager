package ops

import (
	"fmt"
	"path/filepath"
	"strings"
	"workspace-manager/shared/api/types"
)

func (o *Ops) EnvProviderGet(shared bool, id int64) (types.EnvProvider, error) {
	if shared {
		if _, err := o.DB.GetWorkspaceConfigSet(o.Ctx(), id); err != nil {
			return types.EnvProvider{}, err
		}
	} else {
		if _, err := o.DB.GetConfigSet(o.Ctx(), id); err != nil {
			return types.EnvProvider{}, err
		}
	}
	return o.DB.EnvProvider(o.Ctx(), shared, id)
}
func (o *Ops) EnvProviderSave(shared bool, id int64, p types.EnvProvider) (types.Ok, error) {
	if _, err := o.EnvProviderGet(shared, id); err != nil {
		return types.Ok{}, err
	}
	p.Command = strings.TrimSpace(p.Command)
	if p.Timeout == 0 {
		p.Timeout = 30
	}
	if p.Timeout < 1 || p.Timeout > 600 {
		return types.Ok{}, fmt.Errorf("Timeout must be between 1 and 600 seconds")
	}
	if p.Command != "" && shared && !filepath.IsAbs(p.Directory) {
		return types.Ok{}, fmt.Errorf("Shared script requires an absolute working directory")
	}
	if err := o.DB.SaveEnvProvider(o.Ctx(), shared, id, p); err != nil {
		return types.Ok{}, err
	}
	if shared {
		s, _ := o.DB.GetWorkspaceConfigSet(o.Ctx(), id)
		o.changed("sharedSet", s.WorkspaceID)
		o.changed("envVar")
	} else {
		s, _ := o.DB.GetConfigSet(o.Ctx(), id)
		o.changed("envVar", s.AppID)
	}
	return types.Ok{Ok: true}, nil
}

func (o *Ops) EnvProviderRun(shared bool, id int64) (types.Ok, error) {
	if _, err := o.EnvProviderGet(shared, id); err != nil {
		return types.Ok{}, err
	}
	if err := o.Runner.Providers.Run(o.Ctx(), shared, id); err != nil {
		return types.Ok{}, err
	}
	return types.Ok{Ok: true}, nil
}
