package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aymerick/raymond"

	"workspace-manager/daemon/db"
	"workspace-manager/shared/api/lib"
)

func renderHandlebars(content string, env map[string]string) (string, error) {
	tpl, err := raymond.Parse(content)
	if err != nil {
		return "", err
	}
	ctx := make(map[string]any, len(env))
	for k, v := range env {
		ctx[k] = raymond.SafeString(v)
	}
	return tpl.Exec(ctx)
}

func backupRoot(appID int64, sessionID string) string {
	return filepath.Join(db.DataDir(), "backups", strconv.FormatInt(appID, 10), sessionID)
}

// createdManifest is the file, beside a session's backup folder, that lists the
// files ApplyTemplates created because they did not exist yet.
func createdManifest(backupDir string) string {
	return backupDir + ".created.json"
}

// createdFile is a file a template created. Dir is the outermost folder
// (relative to the target dir, slash-separated) that was created for it, or
// empty when its folder already existed.
type createdFile struct {
	Path string `json:"path"`
	Dir  string `json:"dir,omitempty"`
}

// ApplyTemplates renders config set setID's templates into dir (the app's
// project path when empty), backing up the originals for RestoreTemplates. A
// template whose file does not exist yet creates it (with any missing folders);
// RestoreTemplates removes it again.
func ApplyTemplates(ctx context.Context, d *db.DB, appID, setID int64, dir, sessionID string) error {
	env, err := d.EnvToRecord(ctx, setID)
	if err != nil {
		return err
	}
	return applyTemplatesWithEnv(ctx, d, appID, setID, dir, sessionID, env)
}
func applyTemplatesWithEnv(ctx context.Context, d *db.DB, appID, setID int64, dir, sessionID string, env map[string]string) error {
	app, err := d.GetAppT(ctx, appID)
	if err != nil {
		return err
	}
	if dir == "" {
		dir = app.ProjectPath
	}
	templates, err := d.ListTemplatesT(ctx, setID)
	if err != nil {
		return err
	}
	if len(templates) == 0 {
		return nil
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	backupDir := backupRoot(appID, sessionID)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}
	var created []createdFile
	fail := func(err error) error {
		_ = RestoreTemplates(ctx, d, appID, setID, dir, sessionID)
		return err
	}
	for _, template := range templates {
		targetPath, err := lib.ResolveSafePath(root, template.FilePath)
		if err != nil {
			return fail(err)
		}
		if targetPath == root {
			return fail(fmt.Errorf("template path is empty"))
		}
		rendered, err := renderHandlebars(template.Content, env)
		if err != nil {
			return fail(fmt.Errorf("failed to render template %s: %w", template.FilePath, err))
		}
		rel, _ := lib.ToProjectRelative(root, targetPath)
		info, statErr := os.Stat(targetPath)
		switch {
		case statErr == nil && info.IsDir():
			return fail(fmt.Errorf("template target is a folder: %s", template.FilePath))
		case statErr == nil:
			// An existing file: back it up so RestoreTemplates can put it back.
			backupPath := filepath.Join(backupDir, filepath.FromSlash(rel))
			if containsCreated(created, rel) {
				break // created by an earlier template of this run
			}
			if _, err := os.Stat(backupPath); err == nil {
				break // listed twice; the first backup holds the original
			}
			if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
				return fail(err)
			}
			data, err := os.ReadFile(targetPath)
			if err != nil {
				return fail(err)
			}
			if err := os.WriteFile(backupPath, data, 0o644); err != nil {
				return fail(err)
			}
		case os.IsNotExist(statErr):
			// A new file: create it (and its folders) and record it so
			// RestoreTemplates removes it.
			if containsCreated(created, rel) {
				break
			}
			firstDir, err := missingAncestor(root, filepath.Dir(targetPath))
			if err != nil {
				return fail(err)
			}
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fail(err)
			}
			created = append(created, createdFile{Path: rel, Dir: firstDir})
			if err := writeCreated(backupDir, created); err != nil {
				_ = os.Remove(targetPath)
				return fail(err)
			}
		default:
			return fail(statErr)
		}
		if err := os.WriteFile(targetPath, []byte(rendered), 0o644); err != nil {
			return fail(err)
		}
	}
	return nil
}

func containsCreated(created []createdFile, rel string) bool {
	for _, c := range created {
		if c.Path == rel {
			return true
		}
	}
	return false
}

// missingAncestor returns the outermost folder between root and folder that
// does not exist yet (relative to root, slash-separated), or "" when folder
// already exists.
func missingAncestor(root, folder string) (string, error) {
	missing := ""
	for current := folder; current != root; current = filepath.Dir(current) {
		info, err := os.Stat(current)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("not a folder: %s", current)
			}
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		missing = current
		if filepath.Dir(current) == current {
			break
		}
	}
	if missing == "" {
		return "", nil
	}
	rel, _ := lib.ToProjectRelative(root, missing)
	return rel, nil
}

func writeCreated(backupDir string, created []createdFile) error {
	data, err := json.Marshal(created)
	if err != nil {
		return err
	}
	return os.WriteFile(createdManifest(backupDir), data, 0o644)
}

func readCreated(backupDir string) []createdFile {
	data, err := os.ReadFile(createdManifest(backupDir))
	if err != nil {
		return nil
	}
	var created []createdFile
	_ = json.Unmarshal(data, &created)
	return created
}

// RestoreTemplates puts back the files ApplyTemplates replaced in dir (the app's
// project path when empty) and removes the files (and folders) it created.
func RestoreTemplates(ctx context.Context, d *db.DB, appID, setID int64, dir, sessionID string) error {
	app, err := d.GetAppT(ctx, appID)
	if err != nil {
		return nil
	}
	if dir == "" {
		dir = app.ProjectPath
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	backupDir := backupRoot(appID, sessionID)
	created := readCreated(backupDir)
	_, backupErr := os.Stat(backupDir)
	if backupErr != nil && len(created) == 0 {
		return nil
	}
	// Every file in the backup folder is an original, at its path relative to
	// dir. Walking the folder (rather than the current templates) restores a
	// file even when its template was deleted during the run.
	if backupErr == nil {
		_ = filepath.WalkDir(backupDir, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(backupDir, path)
			if err != nil {
				return nil
			}
			targetPath, err := lib.ResolveSafePath(root, filepath.ToSlash(rel))
			if err != nil {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			_ = os.MkdirAll(filepath.Dir(targetPath), 0o755)
			_ = os.WriteFile(targetPath, data, 0o644)
			return nil
		})
	}
	for _, c := range created {
		targetPath, err := lib.ResolveSafePath(root, c.Path)
		if err != nil || targetPath == root {
			continue
		}
		_ = os.Remove(targetPath)
		if c.Dir == "" {
			continue
		}
		stop, err := lib.ResolveSafePath(root, c.Dir)
		if err != nil || stop == root {
			continue
		}
		// Remove the folders created for it, innermost first, while empty.
		for current := filepath.Dir(targetPath); ; current = filepath.Dir(current) {
			if os.Remove(current) != nil || current == stop || len(current) <= len(stop) {
				break
			}
		}
	}
	_ = os.Remove(createdManifest(backupDir))
	// The originals are back in place by now; the backup folder is only
	// clean-up. On Windows a scanner or indexer can hold a just-read file for
	// a moment, so retry briefly and otherwise leave the folder: a stale backup
	// is harmless and must not turn a manual stop into a failed run.
	removeAllRetry(backupDir)
	return nil
}

func removeAllRetry(dir string) {
	delay := 50 * time.Millisecond
	for attempt := 0; attempt < 5; attempt++ {
		if os.RemoveAll(dir) == nil {
			return
		}
		time.Sleep(delay)
		delay *= 2
	}
}

// RenderTemplate renders template content with a config set's env vars, the
// way ApplyTemplates does when it writes a file.
func RenderTemplate(content string, env map[string]string) (string, error) {
	return renderHandlebars(content, env)
}
