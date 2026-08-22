package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	openaiGo "github.com/openai/openai-go"
)

// This file holds the reusable AI setup: the Genkit runtime construction,
// provider plugin wiring, and the lazily-cached service that hands out a
// runtime for the active connection's config. Configuration storage and
// resolution live in aiconfig.go; the app's AI features are the App methods
// in bind.go.

// oaiChatConfig is the per-request config for OpenAI-compatible models. It
// embeds RequestConfig so it satisfies the full ChatConfig contract
// (RequestAPIKey/RequestExtra) and adds the sampling knobs the app exposes.
type oaiChatConfig struct {
	compat_oai.RequestConfig

	Temperature     *float64 `json:"temperature,omitempty" jsonschema:"minimum=0,maximum=2"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty" jsonschema:"minimum=1"`
}

func (c oaiChatConfig) ApplyToChatCompletion(params *openaiGo.ChatCompletionNewParams) {
	c.ApplyVersion(params)
	if c.Temperature != nil {
		params.Temperature = openaiGo.Float(*c.Temperature)
	}
	if c.MaxOutputTokens > 0 {
		params.MaxCompletionTokens = openaiGo.Int(int64(c.MaxOutputTokens))
	}
}

// oaiPlugin adapts an OpenAI-compatible endpoint into a Genkit plugin. Its
// Init builds the client and returns the model action, which genkit.Init then
// registers under "<provider>/<model>".
type oaiPlugin struct {
	comp    *compat_oai.OpenAICompatible
	modelID string
	model   ai.ModelOptions
}

func (p *oaiPlugin) Name() string { return p.comp.Provider }

func (p *oaiPlugin) Init(ctx context.Context) []api.Action {
	p.comp.Init(ctx)
	return []api.Action{
		compat_oai.NewChatModel[oaiChatConfig](p.comp, p.modelID, p.model),
	}
}

// aiService lazily builds a Genkit runtime for a connection config and reuses
// it until that config changes. provider is part of the cache identity: it is
// both the plugin name and half of the registered model name.
type aiService struct {
	mu        sync.Mutex
	g         *genkit.Genkit
	provider  string
	cfg       AIProviderConfig
	modelName string
}

var aiSvc aiService

// ensure returns a cached Genkit runtime for the connection, building a fresh
// one the first time this provider+config is seen.
func (s *aiService) ensure(ctx context.Context, provider string, cfg AIProviderConfig) (*genkit.Genkit, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.g != nil && s.provider == provider && s.cfg == cfg {
		return s.g, s.modelName, nil
	}
	g, modelName, err := buildAI(ctx, provider, cfg)
	if err != nil {
		return nil, "", err
	}
	s.g = g
	s.provider = provider
	s.cfg = cfg
	s.modelName = modelName
	return g, modelName, nil
}

// buildAI constructs a Genkit runtime with the plugin matching the provider.
// Plugin Init may panic on invalid credentials, so it is wrapped and converted
// to an error to keep the desktop app alive.
func buildAI(ctx context.Context, provider string, cfg AIProviderConfig) (g *genkit.Genkit, modelName string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("AI provider failed to initialize: %v", r)
		}
	}()

	if provider == "google" {
		g = genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{APIKey: cfg.APIKey}))
		return g, "googleai/" + cfg.Model, nil
	}

	g = genkit.Init(ctx, genkit.WithPlugins(&oaiPlugin{
		comp: &compat_oai.OpenAICompatible{
			Provider: provider,
			APIKey:   cfg.APIKey,
			// Normalize here (not in the stored config) so the base URL is
			// kept exactly as the user entered it while the SDK still builds
			// <base>/chat/completions instead of mangling the last path.
			BaseURL: OAIBaseURL(cfg.BaseURL),
		},
		modelID: cfg.Model,
		model: ai.ModelOptions{
			Label: provider + "/" + cfg.Model,
			Supports: &ai.ModelSupports{
				Multiturn:  true,
				SystemRole: true,
				Tools:      true,
				Output:     []string{"text", "json"},
			},
		},
	}))
	return g, provider + "/" + cfg.Model, nil
}

// runGeneration sends one request through the (cached) runtime for the given
// connection. Both ChatAI and TestAI go through here.
func runGeneration(ctx context.Context, provider string, cfg AIProviderConfig, system, prompt string) (string, error) {
	resolved, err := ResolveAIConfig(provider, cfg)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	g, model, err := aiSvc.ensure(ctx, provider, resolved)
	if err != nil {
		return "", err
	}

	opts := []ai.GenerateOption{
		ai.WithModelName(model),
	}
	if strings.TrimSpace(system) != "" {
		opts = append(opts, ai.WithMessages(ai.NewSystemTextMessage(system), ai.NewUserTextMessage(prompt)))
	} else {
		opts = append(opts, ai.WithMessages(ai.NewUserTextMessage(prompt)))
	}
	if provider != "google" && resolved.Temperature != nil {
		opts = append(opts, ai.WithConfig(oaiChatConfig{Temperature: resolved.Temperature}))
	}
	return genkit.GenerateText(ctx, g, opts...)
}

// ChatAI runs one generation against the active connection and returns the
// generated text. It loads the persisted store and (re)builds the runtime when
// the active connection's config changed.
func ChatAI(ctx context.Context, system, prompt string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}
	store, err := LoadAIStore()
	if err != nil {
		return "", err
	}
	_, conn, err := store.ActiveAIConnection()
	if err != nil {
		return "", err
	}
	return runGeneration(ctx, conn.Provider, conn, system, prompt)
}

// TestAI tries one minimal generation against an unsaved connection payload.
// It applies the same defaults and validation as real use, so it fails exactly
// when saving and chatting would. Nothing is persisted.
func TestAI(ctx context.Context, provider string, cfg AIProviderConfig) (string, error) {
	if provider == "" {
		return "", fmt.Errorf("provider is required")
	}
	return runGeneration(ctx, provider, cfg, "", "Reply with exactly: OK")
}
