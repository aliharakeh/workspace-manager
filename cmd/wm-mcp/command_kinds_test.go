package main

import "testing"

func TestCommandKinds(t *testing.T) {
	for _, kind := range []string{"setup", "dev", "build", "prod"} {
		if err := checkKind(kind); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
	}
	for _, kind := range []string{"", "run", "other"} {
		if err := checkKind(kind); err == nil {
			t.Fatalf("accepted %q", kind)
		}
	}
	parts := copyParts("")
	for _, part := range []string{"env", "templates", "setup", "dev", "build", "prod"} {
		if parts[part] != true {
			t.Fatalf("default copy missing %s", part)
		}
	}
}
