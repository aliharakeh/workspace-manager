package types

import (
	"encoding/json"
	"testing"
)

// A bundle exported before include_in_ai became private must keep its secrets
// private, and one that carries neither field stays public.
func TestBundleReadsLegacyIncludeInAI(t *testing.T) {
	var vars []BundleEnvVar
	in := `[{"key":"A","value":"1","include_in_ai":false},
		{"key":"B","value":"2","include_in_ai":true},
		{"key":"C","value":"3","private":true,"include_in_ai":true},
		{"key":"D","value":"4"}]`
	if err := json.Unmarshal([]byte(in), &vars); err != nil {
		t.Fatal(err)
	}
	want := []bool{true, false, true, false}
	for i, v := range vars {
		if v.Private != want[i] {
			t.Errorf("var %s: private = %v, want %v", v.Key, v.Private, want[i])
		}
	}
	if vars[0].Value != "1" || vars[3].Key != "D" {
		t.Errorf("other fields lost: %+v", vars)
	}

	var tpls []BundleTemplate
	in = `[{"file_path":".env","content":"x","include_in_ai":false},{"file_path":"a","content":"y","private":false}]`
	if err := json.Unmarshal([]byte(in), &tpls); err != nil {
		t.Fatal(err)
	}
	if !tpls[0].Private || tpls[1].Private || tpls[0].Content != "x" {
		t.Errorf("templates: %+v", tpls)
	}

	out, _ := json.Marshal(BundleEnvVar{Key: "K", Private: true})
	if string(out) != `{"key":"K","value":"","private":true}` {
		t.Errorf("marshal: %s", out)
	}
}
