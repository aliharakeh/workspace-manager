package lib

import (
	"fmt"
	"strings"
)

type ParsedEnvEntry struct {
	Key   string
	Value string
}

type EnvParseError struct {
	Line    int
	Message string
}

func (e *EnvParseError) Error() string {
	return fmt.Sprintf("%s (line %d)", e.Message, e.Line)
}

func ParseEnvFile(content string) ([]ParsedEnvEntry, error) {
	var entries []ParsedEnvEntry
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "export ") {
			line = trimmed[len("export "):]
			trimmed = strings.TrimSpace(line)
		}
		eq := strings.Index(trimmed, "=")
		if eq <= 0 {
			return nil, &EnvParseError{Message: "Not a valid env line (expected KEY=VALUE)", Line: i + 1}
		}
		key := strings.TrimSpace(trimmed[:eq])
		if key == "" {
			return nil, &EnvParseError{Message: "Empty variable key", Line: i + 1}
		}
		value := strings.TrimSpace(trimmed[eq+1:])
		if len(value) > 0 && (value[0] == '"' || value[0] == '\'') {
			quote := value[0]
			end := strings.LastIndex(value, string(quote))
			if end <= 0 {
				return nil, &EnvParseError{Message: "Unclosed quote in value", Line: i + 1}
			}
			value = value[1:end]
		} else if hash := strings.Index(value, " #"); hash != -1 {
			value = strings.TrimSpace(value[:hash])
		}
		entries = append(entries, ParsedEnvEntry{Key: key, Value: value})
	}
	return entries, nil
}

func EnvFileToTemplate(content string) string {
	var out []string
	for _, raw := range strings.Split(content, "\n") {
		raw = strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			out = append(out, raw)
			continue
		}
		head := trimmed
		prefix := ""
		if strings.HasPrefix(head, "export ") {
			prefix = "export "
			head = head[len("export "):]
		}
		eq := strings.Index(head, "=")
		if eq <= 0 {
			out = append(out, raw)
			continue
		}
		key := strings.TrimSpace(head[:eq])
		out = append(out, prefix+key+"={{"+key+"}}")
	}
	return strings.Join(out, "\n") + "\n"
}
