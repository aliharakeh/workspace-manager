package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ValidateProjectPath(projectPath string) (ok bool, resolved string, errMsg string) {
	trimmed := strings.TrimSpace(projectPath)
	if trimmed == "" {
		return false, "", "project_path is required"
	}
	resolvedPath, err := filepath.Abs(trimmed)
	if err != nil {
		return false, "", err.Error()
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "", fmt.Sprintf("Path does not exist: %s", resolvedPath)
		}
		return false, "", fmt.Sprintf("Cannot access path: %s", resolvedPath)
	}
	if !info.IsDir() {
		return false, "", fmt.Sprintf("Path is not a directory: %s", resolvedPath)
	}
	return true, resolvedPath, ""
}

func ResolveProjectFile(projectPath, relativePath string) (absolute string, errMsg string) {
	cleaned := strings.TrimLeft(strings.ReplaceAll(strings.TrimSpace(relativePath), "\\", "/"), "/")
	if cleaned == "" {
		return "", "file path is required"
	}
	root, err := filepath.Abs(projectPath)
	if err != nil {
		return "", err.Error()
	}
	full, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(cleaned)))
	if err != nil {
		return "", err.Error()
	}
	if !isInside(root, full) {
		return "", "Path escapes project directory"
	}
	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Sprintf("File not found: %s", cleaned)
		}
		return "", fmt.Sprintf("Cannot access file: %s", cleaned)
	}
	if info.IsDir() {
		return "", fmt.Sprintf("Not a file: %s", cleaned)
	}
	return full, ""
}

func ReadProjectFile(projectPath, relativePath string) (content, rel string, errMsg string) {
	full, errMsg := ResolveProjectFile(projectPath, relativePath)
	if errMsg != "" {
		return "", "", errMsg
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", "", err.Error()
	}
	root, _ := filepath.Abs(projectPath)
	relPath, err := filepath.Rel(root, full)
	if err != nil || !isInside(root, full) {
		return "", "", "Path escapes project directory"
	}
	return string(data), filepath.ToSlash(relPath), ""
}

func ToProjectRelative(rootDir, absolutePath string) (string, bool) {
	root, err := filepath.Abs(rootDir)
	if err != nil {
		return "", false
	}
	full, err := filepath.Abs(absolutePath)
	if err != nil {
		return "", false
	}
	if !isInside(root, full) {
		return "", false
	}
	rel, err := filepath.Rel(root, full)
	if err != nil {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func ResolveSafePath(projectPath, relativePath string) (string, error) {
	cleaned := strings.TrimLeft(strings.ReplaceAll(relativePath, "\\", "/"), "/")
	root, err := filepath.Abs(projectPath)
	if err != nil {
		return "", err
	}
	full, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(cleaned)))
	if err != nil {
		return "", err
	}
	if !isInside(root, full) {
		return "", fmt.Errorf("Template path escapes project directory: %s", relativePath)
	}
	return full, nil
}

func isInside(root, full string) bool {
	root = filepath.Clean(root)
	full = filepath.Clean(full)
	sep := string(os.PathSeparator)
	return full == root || strings.HasPrefix(full, root+sep)
}
