package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenCodeSessionHeader(t *testing.T) {
	for _, tc := range []struct {
		name string
		on   bool
	}{{"enabled", true}, {"disabled", false}} {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.Header.Get("x-opencode-session")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"1","object":"chat.completion","created":1,"model":"m","choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}]}`))
			}))
			defer srv.Close()

			_, err := TestAI(context.Background(), "custom-"+tc.name, AIProviderConfig{
				BaseURL:         srv.URL + "/v1",
				Model:           "m",
				APIKey:          "k",
				OpenCodeSession: tc.on,
			})
			if err != nil {
				t.Fatal(err)
			}
			if tc.on && got == "" {
				t.Fatal("x-opencode-session header missing")
			}
			if !tc.on && got != "" {
				t.Fatalf("unexpected x-opencode-session header %q", got)
			}
		})
	}
}
