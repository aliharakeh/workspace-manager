package native

import "testing"

func TestOpenInEditorUnknown(t *testing.T) {
	if err := OpenInEditor("nope", "."); err == nil {
		t.Fatal("expected error for unknown editor")
	}
}
