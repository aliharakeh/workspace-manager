package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wails_backend/internal/db"
)

// AI provider settings live in ai.json next to the SQLite database. Multiple
// OpenAI-compatible (or Google) connections are stored keyed by a normalized
// connection name (so one provider can have several connections), with one
// marked active ("default") for chat.
//
// v1 files stored a single flat config ({provider, apiKey, ...}) and v2 files
// keyed connections by provider slug; both are migrated on load (the name
// defaults to the provider slug).

// AIProviderConfig is one saved connection. The API key is stored cleartext.
type AIProviderConfig struct {
	Provider    string   `json:"provider,omitempty"`
	BaseURL     string   `json:"baseURL,omitempty"`
	APIKey      string   `json:"apiKey,omitempty"`
	Model       string   `json:"model,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
}

// AIStore is the persisted multi-provider configuration.
type AIStore struct {
	Providers map[string]AIProviderConfig `json:"providers,omitempty"`
	Active    string                      `json:"active,omitempty"`
}

func aiConfigPath() string {
	return filepath.Join(db.DataDir(), "ai.json")
}

// migrateLegacyStore converts the old flat single-provider file into a store
// with one connection that is also the active one.
func migrateLegacyStore(raw []byte) (AIStore, error) {
	var cfg AIProviderConfig
	var legacy struct {
		Provider string `json:"provider"`
	}
	if err := json.Unmarshal(raw, &legacy); err != nil || legacy.Provider == "" {
		return AIStore{}, err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return AIStore{}, err
	}
	slug := NormalizeProvider(legacy.Provider)
	if slug == "" {
		return AIStore{}, nil
	}
	return AIStore{
		Providers: map[string]AIProviderConfig{slug: {
			Provider: slug,
			BaseURL:  cfg.BaseURL,
			APIKey:   cfg.APIKey,
			Model:    cfg.Model,
		}},
		Active: slug,
	}, nil
}

func LoadAIStore() (AIStore, error) {
	b, err := os.ReadFile(aiConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return AIStore{}, nil
		}
		return AIStore{}, err
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(b, &probe); err != nil {
		return AIStore{}, err
	}
	if _, ok := probe["providers"]; !ok {
		return migrateLegacyStore(b)
	}
	var store AIStore
	if err := json.Unmarshal(b, &store); err != nil {
		return AIStore{}, err
	}
	if store.Providers == nil {
		store.Providers = map[string]AIProviderConfig{}
	}
	// v2 files keyed by provider slug stored no inner provider; backfill it
	// from the key so every entry carries the provider it talks to.
	for name, cfg := range store.Providers {
		if cfg.Provider == "" {
			cfg.Provider = NormalizeProvider(name)
			if cfg.Provider == "" {
				cfg.Provider = name
			}
			store.Providers[name] = cfg
		}
	}
	return store, nil
}

func SaveAIStore(store AIStore) error {
	p := aiConfigPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o600)
}

// UpsertAIConnection stores one connection under its normalized name (one
// provider may have several connections). Empty apiKey keeps the stored key;
// clearApiKey removes it. The first saved connection becomes the active
// default automatically.
func (s *AIStore) UpsertAIConnection(name, provider string, cfg AIProviderConfig, clearAPIKey bool) error {
	name = Slugify(name)
	if name == "" {
		return fmt.Errorf("connection name is required")
	}
	provider = NormalizeProvider(provider)
	if provider == "" {
		return fmt.Errorf("provider is required")
	}
	cfg.Provider = provider
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.Model = strings.TrimSpace(cfg.Model)
	if clearAPIKey {
		cfg.APIKey = ""
	} else if strings.TrimSpace(cfg.APIKey) == "" {
		cfg.APIKey = s.Providers[name].APIKey
	} else {
		cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	}
	if s.Providers == nil {
		s.Providers = map[string]AIProviderConfig{}
	}
	s.Providers[name] = cfg
	if s.Active == "" {
		s.Active = name
	}
	return nil
}

func (s *AIStore) DeleteAIConnection(name string) bool {
	key := Slugify(name)
	if key == "" {
		return false
	}
	if _, ok := s.Providers[key]; !ok {
		return false
	}
	delete(s.Providers, key)
	if s.Active == key {
		s.Active = ""
	}
	return true
}

func (s *AIStore) ActivateAIConnection(name string) error {
	key := Slugify(name)
	if _, ok := s.Providers[key]; !ok {
		return fmt.Errorf("no connection saved for %q", name)
	}
	s.Active = key
	return nil
}

// ActiveAIConnection returns the active connection's config, or an error when
// no default is configured.
func (s *AIStore) ActiveAIConnection() (string, AIProviderConfig, error) {
	if s.Active == "" {
		return "", AIProviderConfig{}, fmt.Errorf("no default AI connection configured")
	}
	conn, ok := s.Providers[s.Active]
	if !ok {
		return "", AIProviderConfig{}, fmt.Errorf("no default AI connection configured")
	}
	return s.Active, conn, nil
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify turns a user-typed connection name into a URL-safe store key.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// NormalizeProvider maps the provider names a user might type onto the
// canonical ids used by the plugins. "google" is served by the googlegenai
// plugin; every other provider goes through the OpenAI-compatible compat_oai
// plugin. Unknown names become URL-safe slugs so they can be used in API paths
// and as store keys.
func NormalizeProvider(p string) string {
	s := strings.ToLower(strings.TrimSpace(p))
	switch s {
	case "gemini", "googleai", "google-genai":
		return "google"
	default:
		return Slugify(s)
	}
}

// oaiDefaults are sensible starting values for well-known OpenAI-compatible
// providers. An empty baseURL means "use the provider's default endpoint" (the
// OpenAI SDK default). An empty model means the model is required from config.
type oaiDefaults struct {
	baseURL string
	model   string
}

var openAICompatDefaults = map[string]oaiDefaults{
	"opencode":   {baseURL: "http://localhost:4096/v1", model: "claude-sonnet-4-20250514"},
	"openai":     {baseURL: "", model: "gpt-4o-mini"},
	"openrouter": {baseURL: "https://openrouter.ai/api/v1", model: ""},
	"anthropic":  {baseURL: "https://api.anthropic.com/v1", model: ""},
	"deepseek":   {baseURL: "https://api.deepseek.com/v1", model: "deepseek-chat"},
	"xai":        {baseURL: "https://api.x.ai/v1", model: ""},
}

// ResolveAIConfig applies defaults to a stored connection's config, returning
// a config that is guaranteed to be buildable (or an actionable error).
func ResolveAIConfig(provider string, cfg AIProviderConfig) (AIProviderConfig, error) {
	if provider == "" {
		return cfg, fmt.Errorf("no AI provider configured")
	}

	if provider == "google" {
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("GEMINI_API_KEY")
		}
		if cfg.APIKey == "" {
			return cfg, fmt.Errorf("google models need an API key (GEMINI_API_KEY)")
		}
		if cfg.Model == "" {
			cfg.Model = "gemini-2.5-flash"
		}
		return cfg, nil
	}

	if provider == "openai" && cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("OPENAI_API_KEY")
	}

	if d, ok := openAICompatDefaults[provider]; ok {
		if cfg.BaseURL == "" {
			cfg.BaseURL = d.baseURL
		}
		if cfg.Model == "" {
			cfg.Model = d.model
		}
	} else if cfg.BaseURL == "" {
		return cfg, fmt.Errorf("provider %q needs a baseURL pointing at an OpenAI-compatible endpoint", provider)
	}
	if cfg.Model == "" {
		return cfg, fmt.Errorf("no model configured for provider %q", provider)
	}
	return cfg, nil
}

// OAIBaseURL normalizes a user-supplied OpenAI-compatible base URL so the SDK
// builds the endpoint the user intended. The openai-go SDK resolves the
// per-request path ("chat/completions") against the base URL, and that
// resolution silently drops the last path segment when the base URL has no
// trailing slash (so ".../v1" would become ".../chat/completions"). Ending the
// root with a slash yields exactly <base>/chat/completions. If the user pasted
// the full endpoint, that suffix is dropped first so it is not doubled.
func OAIBaseURL(base string) string {
	b := strings.TrimSpace(base)
	if b == "" {
		return ""
	}
	b = strings.TrimRight(b, "/")
	if strings.HasSuffix(strings.ToLower(b), "/chat/completions") {
		b = b[:len(b)-len("/chat/completions")]
	}
	return b + "/"
}
