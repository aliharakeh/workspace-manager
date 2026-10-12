// Package ai holds the AI connections and agents. It is served to GUI clients
// only: the CLI has no AI commands, because agents drive the CLI directly.
package ai

import (
	"context"
	"sort"

	"workspace-manager/api"
	"workspace-manager/api/types"
	"workspace-manager/server/db"
)

// ErrHidden is what a CLI client sees for a value excluded from AI: the same
// "not found" a missing item gives, so an agent cannot tell what exists.
func ErrHidden() error {
	return api.Errf(api.KindNotFound, "not found")
}

// Service is the AI API used by the handlers and the GUI.
type Service struct {
	DB *db.DB
	// Worktrees applies the git part of a staged patch; the daemon passes the
	// ops layer. It is nil in tests that do not touch worktrees.
	Worktrees Worktrees
}

func NewService(d *db.DB) *Service { return &Service{DB: d} }

// aiConnectionInfo maps a stored connection to what a client sees (no API key).
func aiConnectionInfo(name string, cfg AIProviderConfig) types.AIConnectionInfo {
	return types.AIConnectionInfo{
		Name:            name,
		Provider:        cfg.Provider,
		BaseURL:         cfg.BaseURL,
		Model:           cfg.Model,
		HasAPIKey:       cfg.APIKey != "",
		Temperature:     cfg.Temperature,
		OpenCodeSession: cfg.OpenCodeSession,
	}
}

func aiConfigInfo(store AIStore) types.AIConfigInfo {
	info := types.AIConfigInfo{
		Providers: make([]types.AIConnectionInfo, 0, len(store.Providers)),
		Active:    store.Active,
	}
	names := make([]string, 0, len(store.Providers))
	for name := range store.Providers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		info.Providers = append(info.Providers, aiConnectionInfo(name, store.Providers[name]))
	}
	return info
}

// ConfigGet returns every saved connection, without secrets.
func (s *Service) ConfigGet() (types.AIConfigInfo, error) {
	store, err := LoadAIStore()
	if err != nil {
		return types.AIConfigInfo{}, err
	}
	return aiConfigInfo(store), nil
}

// ConfigSave upserts one connection keyed by its normalized name.
func (s *Service) ConfigSave(body types.AIProviderConfigInput) (types.AIConfigInfo, error) {
	store, err := LoadAIStore()
	if err != nil {
		return types.AIConfigInfo{}, err
	}
	if err := store.UpsertAIConnection(body.Name, body.Provider, AIProviderConfig{
		BaseURL:         body.BaseURL,
		APIKey:          body.APIKey,
		Model:           body.Model,
		Temperature:     body.Temperature,
		OpenCodeSession: body.OpenCodeSession,
	}, body.ClearAPIKey); err != nil {
		return types.AIConfigInfo{}, err
	}
	if err := SaveAIStore(store); err != nil {
		return types.AIConfigInfo{}, err
	}
	return aiConfigInfo(store), nil
}

func (s *Service) ConfigDelete(name string) (types.AIConfigInfo, error) {
	store, err := LoadAIStore()
	if err != nil {
		return types.AIConfigInfo{}, err
	}
	if !store.DeleteAIConnection(name) {
		return types.AIConfigInfo{}, api.Errf(api.KindNotFound, "Connection not found")
	}
	if err := SaveAIStore(store); err != nil {
		return types.AIConfigInfo{}, err
	}
	return aiConfigInfo(store), nil
}

func (s *Service) ConfigActivate(body types.AIActivateInput) (types.AIConfigInfo, error) {
	store, err := LoadAIStore()
	if err != nil {
		return types.AIConfigInfo{}, err
	}
	if err := store.ActivateAIConnection(body.Name); err != nil {
		return types.AIConfigInfo{}, err
	}
	if err := SaveAIStore(store); err != nil {
		return types.AIConfigInfo{}, err
	}
	return aiConfigInfo(store), nil
}

// Chat runs one generation against the active connection.
func (s *Service) Chat(ctx context.Context, body types.AIChatInput) (types.AIChatResult, error) {
	text, err := ChatAI(ctx, body.System, body.Prompt)
	if err != nil {
		return types.AIChatResult{}, err
	}
	return types.AIChatResult{Text: text}, nil
}

// AppChat runs the config-set agent. Its changes come back as a patch for the
// GUI to review; nothing is written until ApplyPatch runs.
func (s *Service) AppChat(ctx context.Context, body types.AppAIChatInput, emit func(types.AppAIStreamEvent)) (types.AppAIChatResult, error) {
	return AppChatAI(ctx, s.DB, body.AppID, body.ConfigSetID, body.History, body.Instruction, emit)
}

// Test runs one minimal generation against an unsaved connection payload.
func (s *Service) Test(ctx context.Context, body types.AITestInput) (types.AITestResult, error) {
	slug := NormalizeProvider(body.Provider)
	text, err := TestAI(ctx, slug, AIProviderConfig{
		BaseURL:         body.BaseURL,
		APIKey:          body.APIKey,
		Model:           body.Model,
		Temperature:     body.Temperature,
		OpenCodeSession: body.OpenCodeSession,
	})
	if err != nil {
		return types.AITestResult{}, err
	}
	return types.AITestResult{Ok: true, Text: text}, nil
}

// ProposeBlueprint asks the active connection for a blueprint. Nothing is
// persisted; the caller shows the result for review.
func (s *Service) ProposeBlueprint(ctx context.Context, body types.BlueprintAIInput) (types.BlueprintAIResult, error) {
	return ProposeBlueprintAI(ctx, body.Instruction, body.Draft)
}
