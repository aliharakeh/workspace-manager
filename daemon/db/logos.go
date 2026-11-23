package db

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"workspace-manager/shared/api"
)

// Workspace logos are files in LogoDir; workspaces.icon holds only the
// daemon-generated file name ("logo-*.png"), like notes.image.

// MaxLogo caps a workspace logo file. The GUI scales an upload down to a small
// square first, so this is far above what it sends.
const MaxLogo = 1024 * 1024

const logoPrefix = "logo-"

// LogoDir is the folder that holds the workspace logos, next to the database.
func LogoDir() string {
	return filepath.Join(DataDir(), "logos")
}

// IsLogoName reports whether name is a file name the daemon generated, never a
// caller-supplied path.
func IsLogoName(name string) bool {
	return name != "" && filepath.Base(name) == name && strings.HasPrefix(name, logoPrefix)
}

// LogoPath returns the full path of a stored logo, or "" for none.
func LogoPath(name *string) string {
	if name == nil || !IsLogoName(*name) {
		return ""
	}
	return filepath.Join(LogoDir(), *name)
}

// DecodeLogoData accepts base64 image bytes, bare or as a data URL.
func DecodeLogoData(encoded string) ([]byte, error) {
	if strings.HasPrefix(encoded, "data:") {
		i := strings.Index(encoded, ";base64,")
		if i < 0 {
			return nil, api.Errf(api.KindInvalid, "logo must be base64 image data")
		}
		encoded = encoded[i+len(";base64,"):]
	}
	if len(encoded) > base64.StdEncoding.EncodedLen(MaxLogo) {
		return nil, api.Errf(api.KindInvalid, "logo must be %d KB or smaller", MaxLogo/1024)
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(data) == 0 || len(data) > MaxLogo {
		return nil, api.Errf(api.KindInvalid, "invalid base64 logo (maximum %d KB)", MaxLogo/1024)
	}
	return data, nil
}

// StoreLogo checks that data is a PNG, JPEG or GIF image and writes it to a new
// file in LogoDir, returning the file name to keep in workspaces.icon.
func StoreLogo(data []byte) (string, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 16000000 {
		return "", api.Errf(api.KindInvalid, "logo must be a PNG, JPEG or GIF image")
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return "", api.Errf(api.KindInvalid, "invalid logo image: %v", err)
	}
	if err := os.MkdirAll(LogoDir(), 0o700); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(LogoDir(), logoPrefix+"*."+format)
	if err != nil {
		return "", err
	}
	name := filepath.Base(f.Name())
	_, err = f.Write(data)
	if err = errors.Join(err, f.Close()); err != nil {
		return "", errors.Join(err, RemoveLogo(name))
	}
	return name, nil
}

// RemoveLogo deletes a stored logo file; a missing one is not an error.
func RemoveLogo(name string) error {
	if name == "" {
		return nil
	}
	if !IsLogoName(name) {
		return fmt.Errorf("invalid managed logo name")
	}
	err := os.Remove(filepath.Join(LogoDir(), name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// ReadLogoDataURL returns a stored logo as a data URL, or "" for none.
func ReadLogoDataURL(name *string) (string, error) {
	path := LogoPath(name)
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	mime := "image/" + strings.TrimPrefix(filepath.Ext(path), ".")
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// moveWorkspaceLogosToFiles upgrades workspaces.icon values written before
// logos were files: a base64 data URL is written to LogoDir and replaced by
// its file name; anything else that is not a stored logo's name is cleared.
func moveWorkspaceLogosToFiles(sqlDB *sql.DB) error {
	rows, err := sqlDB.Query(`SELECT id, icon FROM workspaces WHERE icon IS NOT NULL`)
	if err != nil {
		return err
	}
	type legacy struct {
		id   int64
		icon string
	}
	var todo []legacy
	for rows.Next() {
		var l legacy
		if err := rows.Scan(&l.id, &l.icon); err != nil {
			_ = rows.Close()
			return err
		}
		if !IsLogoName(l.icon) {
			todo = append(todo, l)
		}
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return err
	}
	for _, l := range todo {
		var next *string
		if strings.HasPrefix(l.icon, "data:image/") {
			if data, err := DecodeLogoData(l.icon); err == nil {
				if name, err := StoreLogo(data); err == nil {
					next = &name
				}
			}
		}
		if _, err := sqlDB.Exec(`UPDATE workspaces SET icon = ? WHERE id = ?`, next, l.id); err != nil {
			if next != nil {
				err = errors.Join(err, RemoveLogo(*next))
			}
			return err
		}
	}
	return nil
}
