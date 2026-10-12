package handlers

import (
	"context"

	"workspace-manager/api"
	"workspace-manager/api/types"
)

// AIOps is the AI surface. Only GUI clients reach these methods.
type AIOps interface {
	ConfigGet() (types.AIConfigInfo, error)
	ConfigSave(body types.AIProviderConfigInput) (types.AIConfigInfo, error)
	ConfigDelete(name string) (types.AIConfigInfo, error)
	ConfigActivate(body types.AIActivateInput) (types.AIConfigInfo, error)
	Chat(ctx context.Context, body types.AIChatInput) (types.AIChatResult, error)
	AppChat(ctx context.Context, body types.AppAIChatInput, emit func(types.AppAIStreamEvent)) (types.AppAIChatResult, error)
	Test(ctx context.Context, body types.AITestInput) (types.AITestResult, error)
	ApplyPatch(appID, configSetID int64, patch types.AppAIPatch) error
	ProposeBlueprint(ctx context.Context, body types.BlueprintAIInput) (types.BlueprintAIResult, error)
}

// AppChat emits its progress as appAIEvent notifications, so the caller does
// not stream through the RPC reply.
func (r *Registry) aiOps() {
	svc := r.AI

	r.handle(api.MethodAIConfigGet, typed(func(struct{}) (types.AIConfigInfo, error) {
		return svc.ConfigGet()
	}))
	r.handle(api.MethodAIConfigSave, typed(svc.ConfigSave))
	r.handle(api.MethodAIConfigDelete, typed(func(in struct {
		Name string `json:"name"`
	}) (types.AIConfigInfo, error) {
		return svc.ConfigDelete(in.Name)
	}))
	r.handle(api.MethodAIConfigActivate, typed(svc.ConfigActivate))
	r.handle(api.MethodAIChat, typedCtx(func(ctx context.Context, in types.AIChatInput) (types.AIChatResult, error) {
		return svc.Chat(ctx, in)
	}))
	r.handle(api.MethodAIAppChat, typedCtx(func(ctx context.Context, in types.AppAIChatInput) (types.AppAIChatResult, error) {
		res, err := svc.AppChat(ctx, in, func(ev types.AppAIStreamEvent) {
			r.publish(api.EventAppAI, in.AppID, ev)
		})
		return res, err
	}))
	r.handle(api.MethodAITest, typedCtx(func(ctx context.Context, in types.AITestInput) (types.AITestResult, error) {
		return svc.Test(ctx, in)
	}))
	r.handle(api.MethodAIApplyPatch, typed(func(in struct {
		AppID       int64            `json:"appID"`
		ConfigSetID int64            `json:"configSetID"`
		Patch       types.AppAIPatch `json:"patch"`
	}) (types.Ok, error) {
		if err := svc.ApplyPatch(in.AppID, in.ConfigSetID, in.Patch); err != nil {
			return types.Ok{}, err
		}
		return types.Ok{Ok: true}, nil
	}))
	r.handle(api.MethodBlueprintsAIPropose, typedCtx(func(ctx context.Context, in types.BlueprintAIInput) (types.BlueprintAIResult, error) {
		return svc.ProposeBlueprint(ctx, in)
	}))
}
