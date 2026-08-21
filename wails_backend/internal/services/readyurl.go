package services

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"wails_backend/internal/db"
)

type ReadyURLMatch struct {
	URL       string
	PatternID string
	Label     string
}

type compiledPattern struct {
	id      string
	label   string
	pattern *regexp.Regexp
}

var (
	readyMu    sync.Mutex
	readyCache []compiledPattern
)

func InvalidateReadyURLPatternsCache() {
	readyMu.Lock()
	readyCache = nil
	readyMu.Unlock()
}

func jsFlagsToPrefix(flags string) string {
	var chars []byte
	seen := map[rune]bool{}
	for _, c := range flags {
		if seen[c] {
			continue
		}
		switch c {
		case 'i', 'm', 's':
			seen[c] = true
			chars = append(chars, byte(c))
		}
	}
	if len(chars) == 0 {
		return ""
	}
	return "(?" + string(chars) + ")"
}

func jsNamedGroups(src string) string {
	return regexp.MustCompile(`\(\?<(\w+)>`).ReplaceAllString(src, "(?P<$1>")
}

func CompileJSRegexp(source, flags string) (*regexp.Regexp, error) {
	return regexp.Compile(jsFlagsToPrefix(flags) + jsNamedGroups(source))
}

func ValidateReadyURLPattern(source, flags string) error {
	if _, err := CompileJSRegexp(source, flags); err != nil {
		return fmt.Errorf("Invalid regular expression: %s", err.Error())
	}
	if !strings.Contains(source, "(?<url>") && !strings.Contains(source, "(?<port>") {
		return fmt.Errorf("Pattern must include a named group `url` and/or `port`")
	}
	return nil
}

func loadPatterns(ctx context.Context, d *db.DB) []compiledPattern {
	_ = d.EnsureReadyURLPatternsSeeded(ctx)
	rows, err := d.ListReadyUrlPatterns(ctx)
	if err != nil {
		return nil
	}
	var compiled []compiledPattern
	for _, row := range rows {
		re, err := CompileJSRegexp(row.Pattern, row.Flags)
		if err != nil {
			continue
		}
		compiled = append(compiled, compiledPattern{
			id:      strconv.FormatInt(row.ID, 10),
			label:   row.Label,
			pattern: re,
		})
	}
	return compiled
}

func getPatterns(ctx context.Context, d *db.DB) []compiledPattern {
	readyMu.Lock()
	defer readyMu.Unlock()
	if readyCache == nil {
		readyCache = loadPatterns(ctx, d)
	}
	return readyCache
}

func cleanURL(raw string) string {
	return strings.TrimRight(raw, ")].,;:'\">")
}

func fromPort(port string) string {
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return ""
	}
	return fmt.Sprintf("http://localhost:%d", n)
}

func MatchReadyURL(ctx context.Context, d *db.DB, line string) *ReadyURLMatch {
	text := strings.TrimSpace(line)
	if text == "" {
		return nil
	}
	for _, p := range getPatterns(ctx, d) {
		m := p.pattern.FindStringSubmatch(text)
		if m == nil {
			continue
		}
		names := p.pattern.SubexpNames()
		groups := map[string]string{}
		for i, name := range names {
			if i == 0 || name == "" || i >= len(m) {
				continue
			}
			groups[name] = m[i]
		}
		if urlGroup := groups["url"]; urlGroup != "" {
			url := cleanURL(urlGroup)
			if strings.HasPrefix(strings.ToLower(url), "http://") || strings.HasPrefix(strings.ToLower(url), "https://") {
				return &ReadyURLMatch{URL: url, PatternID: p.id, Label: p.label}
			}
			continue
		}
		if portGroup := groups["port"]; portGroup != "" {
			if url := fromPort(portGroup); url != "" {
				return &ReadyURLMatch{URL: url, PatternID: p.id, Label: p.label}
			}
		}
	}
	return nil
}
