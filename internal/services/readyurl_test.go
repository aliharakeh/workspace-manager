package services

import "testing"

func TestCompileJSRegexpNamedGroups(t *testing.T) {
	re, err := CompileJSRegexp(`\bLocal:\s+(?<url>https?:\/\/\S+)`, "i")
	if err != nil {
		t.Fatal(err)
	}
	m := re.FindStringSubmatch("Local: http://localhost:5173/")
	if m == nil {
		t.Fatal("no match")
	}
	names := re.SubexpNames()
	got := ""
	for i, name := range names {
		if name == "url" {
			got = m[i]
		}
	}
	if got != "http://localhost:5173/" {
		t.Fatalf("got %q", got)
	}
}
