package native

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Editor is a code editor the app can open a project in.
type Editor struct {
	ID   string
	Name string
}

type editorDef struct {
	Editor
	commands  []string // looked up on PATH, in order
	macApp    string   // application name for `open -a`
	winGlobs  []string // %ENV%-prefixed globs for installs that are not on PATH
	unixPaths []string
}

var editorDefs = []editorDef{
	{
		Editor:   Editor{ID: "vscode", Name: "VS Code"},
		commands: []string{"code"},
		macApp:   "Visual Studio Code",
		winGlobs: []string{`%LOCALAPPDATA%\Programs\Microsoft VS Code\bin\code.cmd`},
	},
	{
		Editor:    Editor{ID: "zed", Name: "Zed"},
		commands:  []string{"zed", "zeditor"},
		macApp:    "Zed",
		winGlobs:  []string{`%LOCALAPPDATA%\Programs\Zed\bin\zed.exe`, `%LOCALAPPDATA%\Programs\Zed\Zed.exe`},
		unixPaths: []string{"~/.local/bin/zed"},
	},
	{
		Editor:   Editor{ID: "intellij", Name: "IntelliJ IDEA"},
		commands: []string{"idea", "idea64", "intellij-idea-ultimate", "intellij-idea-community"},
		macApp:   "IntelliJ IDEA",
		winGlobs: []string{
			`%LOCALAPPDATA%\JetBrains\Toolbox\scripts\idea.cmd`,
			`%LOCALAPPDATA%\JetBrains\Toolbox\apps\IDEA-*\ch-*\*\bin\idea64.exe`,
			`%LOCALAPPDATA%\Programs\IntelliJ IDEA*\bin\idea64.exe`,
			`%ProgramFiles%\JetBrains\IntelliJ IDEA*\bin\idea64.exe`,
		},
		unixPaths: []string{"/snap/bin/intellij-idea-*", "~/.local/share/JetBrains/Toolbox/scripts/idea"},
	},
}

// Editors lists the editors found on this machine.
func Editors() []Editor {
	out := []Editor{}
	for _, d := range editorDefs {
		if _, ok := d.launcher(); ok {
			out = append(out, d.Editor)
		}
	}
	return out
}

// OpenInEditor opens path in the editor with the given id.
func OpenInEditor(id, path string) error {
	for _, d := range editorDefs {
		if d.ID != id {
			continue
		}
		argv, ok := d.launcher()
		if !ok {
			return fmt.Errorf("%s is not installed (or its command line launcher was not found)", d.Name)
		}
		c := exec.Command(argv[0], append(argv[1:], path)...)
		// Only batch launchers need hiding; hiding a GUI exe hides its window.
		if ext := strings.ToLower(filepath.Ext(argv[0])); ext == ".cmd" || ext == ".bat" {
			hideWindow(c)
		}
		return c.Start()
	}
	return fmt.Errorf("unknown editor %q", id)
}

// launcher returns the command (without the path argument) that starts the editor.
func (d editorDef) launcher() ([]string, bool) {
	for _, name := range d.commands {
		if p, err := exec.LookPath(name); err == nil {
			return []string{p}, true
		}
	}
	var patterns []string
	switch runtime.GOOS {
	case "windows":
		for _, g := range d.winGlobs {
			patterns = append(patterns, os.ExpandEnv(strings.NewReplacer("%LOCALAPPDATA%", "${LOCALAPPDATA}", "%ProgramFiles%", "${ProgramFiles}").Replace(g)))
		}
	case "darwin":
		for _, dir := range []string{"/Applications", expandHome("~/Applications")} {
			if _, err := os.Stat(filepath.Join(dir, d.macApp+".app")); err == nil {
				return []string{"open", "-a", d.macApp}, true
			}
		}
	default:
		for _, p := range d.unixPaths {
			patterns = append(patterns, expandHome(p))
		}
	}
	for _, pattern := range patterns {
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			return []string{matches[len(matches)-1]}, true
		}
	}
	return nil, false
}

func expandHome(p string) string {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, rest)
		}
	}
	return p
}
