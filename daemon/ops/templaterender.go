package ops

import (
	"fmt"

	"workspace-manager/daemon/services"
	"workspace-manager/shared/api/types"
)

// TemplatesRender renders content with the env vars of config set setID, so an
// editor can preview what applying a template would write. It uses the same
// renderer as a run.
func (o *Ops) TemplatesRender(setID int64, content string) (types.TemplateRenderResult, error) {
	if _, err := o.DB.GetConfigSetT(o.Ctx(), setID); err != nil {
		return types.TemplateRenderResult{}, err
	}
	env, err := o.DB.EnvToRecord(o.Ctx(), setID)
	if err != nil {
		return types.TemplateRenderResult{}, err
	}
	out, err := services.RenderTemplate(content, env)
	if err != nil {
		return types.TemplateRenderResult{}, fmt.Errorf("failed to render template: %w", err)
	}
	return types.TemplateRenderResult{Content: out}, nil
}
