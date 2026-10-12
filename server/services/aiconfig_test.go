package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOAIBaseURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://openrouter.ai/api/v1", "https://openrouter.ai/api/v1/"},
		{"http://localhost:4096/v1/", "http://localhost:4096/v1/"},
		{"https://api.deepseek.com/v1/chat/completions", "https://api.deepseek.com/v1/"},
		{"", ""},
		{"   ", ""},
	}
	for _, c := range cases {
		if got := OAIBaseURL(c.in); got != c.want {
			t.Errorf("OAIBaseURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeProvider(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Google", "google"},
		{" gemini ", "google"},
		{"GoogleAI", "google"},
		{"google-genai", "google"},
		{"OpenRouter", "openrouter"},
		{"Local LLM", "local-llm"},
		{"My_Provider!", "my-provider"},
		{"", ""},
	}
	for _, c := range cases {
		if got := NormalizeProvider(c.in); got != c.want {
			t.Errorf("NormalizeProvider(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestResolveAIConfig(t *testing.T) {
	if _, err := ResolveAIConfig("", AIProviderConfig{}); err == nil || err.Error() != "no AI provider configured" {
		t.Errorf("empty provider error = %v", err)
	}

	cfg, err := ResolveAIConfig("google", AIProviderConfig{APIKey: "k"})
	if err != nil {
		t.Fatalf("google resolve: %v", err)
	}
	if cfg.Model != "gemini-2.5-flash" {
		t.Errorf("google defaults = %+v", cfg)
	}

	if _, err := ResolveAIConfig("custom", AIProviderConfig{Model: "m"}); err == nil {
		t.Error("custom provider without baseURL should fail")
	}
	if _, err := ResolveAIConfig("openrouter", AIProviderConfig{}); err == nil {
		t.Error("openrouter without model should fail")
	}

	cfg, err = ResolveAIConfig("deepseek", AIProviderConfig{})
	if err != nil {
		t.Fatalf("deepseek resolve: %v", err)
	}
	if cfg.BaseURL != "https://api.deepseek.com/v1/" && cfg.BaseURL != "https://api.deepseek.com/v1" {
		t.Errorf("deepseek baseURL = %q", cfg.BaseURL)
	}
	if cfg.Model != "deepseek-chat" {
		t.Errorf("deepseek model = %q", cfg.Model)
	}

	cfg, err = ResolveAIConfig("custom", AIProviderConfig{BaseURL: "http://localhost:1234/v1", Model: "m"})
	if err != nil {
		t.Fatalf("custom resolve: %v", err)
	}
	if cfg.Model != "m" || cfg.BaseURL != "http://localhost:1234/v1" {
		t.Errorf("custom passthrough = %+v", cfg)
	}
}

func TestResolveAIConfigEnvFallback(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "env-key")
	cfg, err := ResolveAIConfig("google", AIProviderConfig{})
	if err != nil {
		t.Fatalf("google env fallback: %v", err)
	}
	if cfg.APIKey != "env-key" {
		t.Errorf("google apiKey = %q, want env-key", cfg.APIKey)
	}

	t.Setenv("OPENAI_API_KEY", "env-openai")
	cfg, err = ResolveAIConfig("openai", AIProviderConfig{})
	if err != nil {
		t.Fatalf("openai env fallback: %v", err)
	}
	if cfg.APIKey != "env-openai" {
		t.Errorf("openai apiKey = %q, want env-openai", cfg.APIKey)
	}
}

func TestStoreOperations(t *testing.T) {
	var store AIStore

	if err := store.UpsertAIConnection("Work Router", "openrouter", AIProviderConfig{APIKey: "k1"}, false); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := store.UpsertAIConnection("google", "Google", AIProviderConfig{APIKey: "k2"}, false); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if store.Active != "work-router" {
		t.Errorf("first connection should become active, got %q", store.Active)
	}
	if got := store.Providers["work-router"].Provider; got != "openrouter" {
		t.Errorf("stored provider = %q, want openrouter", got)
	}

	// One provider can back several named connections.
	if err := store.UpsertAIConnection("openrouter-personal", "OpenRouter", AIProviderConfig{APIKey: "k3"}, false); err != nil {
		t.Fatalf("upsert second openrouter: %v", err)
	}
	if len(store.Providers) != 3 {
		t.Errorf("expected 3 connections, got %d: %+v", len(store.Providers), store.Providers)
	}

	if err := store.UpsertAIConnection("", "openai", AIProviderConfig{}, false); err == nil {
		t.Error("empty connection name should fail")
	}
	if err := store.UpsertAIConnection("x", "", AIProviderConfig{}, false); err == nil {
		t.Error("empty provider should fail")
	}

	// Empty key keeps stored; clear removes.
	if err := store.UpsertAIConnection("work-router", "openrouter", AIProviderConfig{BaseURL: "https://x/v1"}, false); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if store.Providers["work-router"].APIKey != "k1" {
		t.Errorf("stored key should be kept, got %q", store.Providers["work-router"].APIKey)
	}
	if store.Providers["work-router"].BaseURL != "https://x/v1" {
		t.Errorf("baseURL not updated: %q", store.Providers["work-router"].BaseURL)
	}
	if err := store.UpsertAIConnection("work-router", "openrouter", AIProviderConfig{}, true); err != nil {
		t.Fatalf("upsert clear: %v", err)
	}
	if store.Providers["work-router"].APIKey != "" {
		t.Errorf("clearApiKey should wipe the key")
	}

	name, conn, err := store.ActiveAIConnection()
	if err != nil || name != "work-router" || conn.Provider != "openrouter" {
		t.Errorf("active connection = %q, %+v, %v", name, conn, err)
	}

	if !store.DeleteAIConnection("work-router") {
		t.Fatal("delete work-router failed")
	}
	if store.Active != "" {
		t.Errorf("deleting the active connection should clear Active, got %q", store.Active)
	}
	if store.DeleteAIConnection("work-router") {
		t.Error("deleting a missing connection should report false")
	}
	if err := store.ActivateAIConnection("nope"); err == nil {
		t.Error("activating a missing connection should fail")
	}
	if err := store.ActivateAIConnection(" google "); err != nil {
		t.Fatalf("activate google: %v", err)
	}
	if _, conn, err := store.ActiveAIConnection(); err != nil || conn.APIKey != "k2" {
		t.Errorf("activate google: %+v, %v", conn, err)
	}
}

func TestLoadAIStoreLegacyMigration(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "ai.json")
	legacy := []byte("{\n  \"provider\": \"OpenRouter\",\n  \"apiKey\": \"legacy-key\",\n  \"model\": \"m\"\n}")
	if err := os.WriteFile(old, legacy, 0o600); err != nil {
		t.Fatal(err)
	}

	// LoadAIStore reads from db.DataDir(); redirect via chdir is not possible,
	// so exercise the migration path directly instead.
	store, err := migrateLegacyStore(legacy)
	if err != nil {
		t.Fatalf("migrateLegacyStore: %v", err)
	}
	conn, ok := store.Providers["openrouter"]
	if !ok || conn.APIKey != "legacy-key" || conn.Model != "m" {
		t.Errorf("migrated providers = %+v", store.Providers)
	}
	if store.Active != "openrouter" {
		t.Errorf("migrated active = %q", store.Active)
	}

	if store, err := migrateLegacyStore([]byte(`{"provider":""}`)); err != nil || len(store.Providers) != 0 {
		t.Errorf("empty legacy provider should yield empty store, got %+v (%v)", store, err)
	}
}
