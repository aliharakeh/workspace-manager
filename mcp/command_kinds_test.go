package mcp

import "testing"

func TestCommandKinds(t *testing.T) {
	for _, kind := range []string{"setup", "dev", "build", "prod", "run", "custom-tests"} {
		if err := checkKind(kind); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
	}
	for _, kind := range []string{""} {
		if err := checkKind(kind); err == nil {
			t.Fatalf("accepted %q", kind)
		}
	}
	legacy := copyParts("dev")
	if legacy["dev"] != true || legacy["env"] != false || legacy["run_configs"] != nil {
		t.Fatalf("legacy copy selection changed: %+v", legacy)
	}
	parts := copyParts("")
	for _, part := range []string{"env", "templates", "run_configs"} {
		if parts[part] != true {
			t.Fatalf("default copy missing %s", part)
		}
	}
}
