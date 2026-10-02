package services

import (
	"testing"
)

func TestRenderHandlebars(t *testing.T) {
	t.Run("basic variable substitution", func(t *testing.T) {
		tpl := "PORT={{PORT}}\nHOST={{HOST}}"
		env := map[string]string{
			"PORT": "3000",
			"HOST": "localhost",
		}

		got, err := renderHandlebars(tpl, env)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := "PORT=3000\nHOST=localhost"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("no html escaping for special characters", func(t *testing.T) {
		tpl := "DATABASE_URL={{DATABASE_URL}}"
		env := map[string]string{
			"DATABASE_URL": "postgres://user:p&ss\"w'd@localhost:5432/db?sslmode=disable&foo=bar<baz>",
		}

		got, err := renderHandlebars(tpl, env)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := "DATABASE_URL=postgres://user:p&ss\"w'd@localhost:5432/db?sslmode=disable&foo=bar<baz>"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("conditionals and blocks", func(t *testing.T) {
		tpl := "{{#if ENABLE_FEATURE}}feature is on{{else}}feature is off{{/if}}"
		env := map[string]string{
			"ENABLE_FEATURE": "true",
		}

		got, err := renderHandlebars(tpl, env)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "feature is on" {
			t.Fatalf("got %q, want %q", got, "feature is on")
		}

		// When omitted / empty
		gotOff, err := renderHandlebars(tpl, map[string]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotOff != "feature is off" {
			t.Fatalf("got %q, want %q", gotOff, "feature is off")
		}
	})

	t.Run("syntax error returns error", func(t *testing.T) {
		tpl := "{{#if UNCLOSED"
		_, err := renderHandlebars(tpl, map[string]string{})
		if err == nil {
			t.Fatal("expected syntax error, got nil")
		}
	})
}
