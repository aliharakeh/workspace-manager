package lib

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type ImportEntry struct {
	Key   string
	Value string
}

type ImportFormat struct {
	Label     string
	Matches   func(fileName string) bool
	Parse     func(content string) ([]ImportEntry, error)
	ToTemplate func(content string) (string, error)
}

func flattenYAML(value any, prefix string, out *[]ImportEntry) {
	if value == nil {
		return
	}
	switch v := value.(type) {
	case []any:
		for i, item := range v {
			key := fmt.Sprintf("%d", i)
			if prefix != "" {
				key = prefix + "." + key
			}
			flattenYAML(item, key, out)
		}
	case map[string]any:
		for k, item := range v {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			flattenYAML(item, key, out)
		}
	default:
		*out = append(*out, ImportEntry{Key: prefix, Value: fmt.Sprint(v)})
	}
}

func templateifyYAML(value any, prefix string) any {
	switch v := value.(type) {
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			key := fmt.Sprintf("%d", i)
			if prefix != "" {
				key = prefix + "." + key
			}
			out[i] = templateifyYAML(item, key)
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for k, item := range v {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			out[k] = templateifyYAML(item, key)
		}
		return out
	default:
		return "{{[" + prefix + "]}}"
	}
}

func yamlToMap(content string) (any, error) {
	var data any
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		return nil, fmt.Errorf("Invalid YAML: %s", err.Error())
	}
	return convertYAMLMap(data), nil
}

func convertYAMLMap(v any) any {
	switch t := v.(type) {
	case map[any]any:
		out := map[string]any{}
		for k, val := range t {
			out[fmt.Sprint(k)] = convertYAMLMap(val)
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for k, val := range t {
			out[k] = convertYAMLMap(val)
		}
		return out
	case []any:
		for i, val := range t {
			t[i] = convertYAMLMap(val)
		}
		return t
	default:
		return v
	}
}

var quotedPlaceholder = regexp.MustCompile(`"({{[^}]*}})"`)

func envFormat() ImportFormat {
	return ImportFormat{
		Label: ".env",
		Matches: func(fileName string) bool {
			name := strings.ToLower(filepath.Base(strings.ReplaceAll(fileName, "\\", "/")))
			return name == "env" || strings.HasPrefix(name, ".env")
		},
		Parse: func(content string) ([]ImportEntry, error) {
			parsed, err := ParseEnvFile(content)
			if err != nil {
				return nil, err
			}
			out := make([]ImportEntry, len(parsed))
			for i, e := range parsed {
				out[i] = ImportEntry{Key: e.Key, Value: e.Value}
			}
			return out, nil
		},
		ToTemplate: func(content string) (string, error) {
			return EnvFileToTemplate(content), nil
		},
	}
}

func yamlFormat() ImportFormat {
	return ImportFormat{
		Label: "YAML",
		Matches: func(fileName string) bool {
			name := strings.ToLower(fileName)
			return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
		},
		Parse: func(content string) ([]ImportEntry, error) {
			data, err := yamlToMap(content)
			if err != nil {
				return nil, err
			}
			var entries []ImportEntry
			flattenYAML(data, "", &entries)
			return entries, nil
		},
		ToTemplate: func(content string) (string, error) {
			data, err := yamlToMap(content)
			if err != nil {
				return "", err
			}
			tpl, err := yaml.Marshal(templateifyYAML(data, ""))
			if err != nil {
				return "", err
			}
			return quotedPlaceholder.ReplaceAllString(string(tpl), "$1") + "\n", nil
		},
	}
}

func DetectImportFormat(fileName string) *ImportFormat {
	formats := []ImportFormat{envFormat(), yamlFormat()}
	for i := range formats {
		if formats[i].Matches(fileName) {
			return &formats[i]
		}
	}
	return nil
}
