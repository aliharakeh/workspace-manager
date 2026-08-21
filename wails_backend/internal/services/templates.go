package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"wails_backend/internal/db"
	"wails_backend/internal/lib"
)

var hbRe = regexp.MustCompile(`\{\{\s*\[?([^\]}]+?)\]?\s*\}\}`)

func renderHandlebars(content string, env map[string]string) string {
	return hbRe.ReplaceAllStringFunc(content, func(m string) string {
		sub := hbRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return ""
		}
		if v, ok := env[strings.TrimSpace(sub[1])]; ok {
			return v
		}
		return ""
	})
}

func backupRoot(appID int64, sessionID string) string {
	return filepath.Join(db.DataDir(), "backups", strconv.FormatInt(appID, 10), sessionID)
}

func ApplyTemplates(ctx context.Context, d *db.DB, appID int64, sessionID string) error {
	app, err := d.GetAppT(ctx, appID)
	if err != nil {
		return err
	}
	set, err := d.ResolveActive(ctx, appID)
	if err != nil {
		return err
	}
	templates, err := d.ListTemplatesT(ctx, set.ID)
	if err != nil {
		return err
	}
	env, err := d.EnvToRecord(ctx, set.ID)
	if err != nil {
		return err
	}
	backupDir := backupRoot(appID, sessionID)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}
	for _, template := range templates {
		targetPath, err := lib.ResolveSafePath(app.ProjectPath, template.FilePath)
		if err != nil {
			_ = RestoreTemplates(ctx, d, appID, sessionID)
			return err
		}
		if _, err := os.Stat(targetPath); err != nil {
			_ = RestoreTemplates(ctx, d, appID, sessionID)
			return fmt.Errorf("Target file does not exist: %s", template.FilePath)
		}
		backupPath := filepath.Join(backupDir, filepath.FromSlash(strings.ReplaceAll(template.FilePath, "\\", "/")))
		if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
			_ = RestoreTemplates(ctx, d, appID, sessionID)
			return err
		}
		data, err := os.ReadFile(targetPath)
		if err != nil {
			_ = RestoreTemplates(ctx, d, appID, sessionID)
			return err
		}
		if err := os.WriteFile(backupPath, data, 0o644); err != nil {
			_ = RestoreTemplates(ctx, d, appID, sessionID)
			return err
		}
		rendered := renderHandlebars(template.Content, env)
		if err := os.WriteFile(targetPath, []byte(rendered), 0o644); err != nil {
			_ = RestoreTemplates(ctx, d, appID, sessionID)
			return err
		}
	}
	return nil
}

func RestoreTemplates(ctx context.Context, d *db.DB, appID int64, sessionID string) error {
	app, err := d.GetAppT(ctx, appID)
	if err != nil {
		return nil
	}
	backupDir := backupRoot(appID, sessionID)
	if _, err := os.Stat(backupDir); err != nil {
		return nil
	}
	set, err := d.ResolveActive(ctx, appID)
	if err != nil {
		return err
	}
	templates, err := d.ListTemplatesT(ctx, set.ID)
	if err != nil {
		return err
	}
	for _, template := range templates {
		backupPath := filepath.Join(backupDir, filepath.FromSlash(strings.ReplaceAll(template.FilePath, "\\", "/")))
		if _, err := os.Stat(backupPath); err != nil {
			continue
		}
		targetPath, err := lib.ResolveSafePath(app.ProjectPath, template.FilePath)
		if err != nil {
			continue
		}
		_ = os.MkdirAll(filepath.Dir(targetPath), 0o755)
		data, err := os.ReadFile(backupPath)
		if err != nil {
			continue
		}
		_ = os.WriteFile(targetPath, data, 0o644)
	}
	return os.RemoveAll(backupDir)
}
