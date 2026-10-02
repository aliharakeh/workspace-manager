package lib

import "testing"

func TestParseEnvFile(t *testing.T) {
	got, err := ParseEnvFile("FOO=bar\n# skip\nexport BAZ=qux\nQUOTED=\"a b\"\nPLAIN=val # note\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || got[0].Key != "FOO" || got[0].Value != "bar" || got[2].Value != "a b" || got[3].Value != "val" {
		t.Fatalf("%+v", got)
	}
}
