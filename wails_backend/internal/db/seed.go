package db

import (
	"context"
	"fmt"
)

type readyURLSeed struct {
	Key     string
	Label   string
	Pattern string
	Flags   string
}

// Mirrors db/ready-url-defaults.ts
var defaultReadyURLPatterns = []readyURLSeed{
	{Key: "next-local", Label: "Next.js Local", Pattern: `-\s*Local:\s+(?<url>https?:\/\/\S+)`, Flags: "i"},
	{Key: "next-ready", Label: "Next.js Ready", Pattern: `\bReady (?:in .+ )?on\s+(?<url>https?:\/\/\S+)`, Flags: "i"},
	{Key: "dev-local", Label: "Dev server Local", Pattern: `\bLocal:\s+(?<url>https?:\/\/\S+)`, Flags: "i"},
	{Key: "dev-network", Label: "Dev server Network", Pattern: `\bNetwork:\s+(?<url>https?:\/\/\S+)`, Flags: "i"},
	{Key: "spring-tomcat", Label: "Spring Boot Tomcat", Pattern: `Tomcat started on port(?:\(s\))?:\s*(?<port>\d+)`, Flags: "i"},
	{Key: "spring-netty", Label: "Spring Boot Netty", Pattern: `Netty started on port\s+(?<port>\d+)`, Flags: "i"},
	{Key: "spring-tomcat-init", Label: "Spring Boot Tomcat init", Pattern: `Tomcat initialized with port(?:\(s\))?:\s*(?<port>\d+)`, Flags: "i"},
	{Key: "dotnet-listening", Label: ".NET Kestrel", Pattern: `Now listening on:\s+(?<url>https?:\/\/\S+)`, Flags: "i"},
	{Key: "django-dev", Label: "Django", Pattern: `Starting development server at\s+(?<url>https?:\/\/\S+)`, Flags: "i"},
	{Key: "listening-on-url", Label: "Listening on URL", Pattern: `\bListening on\s+(?<url>https?:\/\/\S+)`, Flags: "i"},
	{Key: "listening-on-port", Label: "Listening on port", Pattern: `\blisten(?:ing)? on port\s+(?<port>\d+)`, Flags: "i"},
	{Key: "serving-http-port", Label: "Serving HTTP", Pattern: `\bserving HTTP on\s+(?:\S*?:)?(?<port>\d+)`, Flags: "i"},
	{Key: "generic-url", Label: "Generic URL", Pattern: `\b(?:running|started|available|serving)\s+(?:at|on)\s+(?<url>https?:\/\/\S+)`, Flags: "i"},
}

func (d *DB) EnsureReadyURLPatternsSeeded(ctx context.Context) error {
	rows, err := d.ListReadyUrlPatterns(ctx)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		for i, entry := range defaultReadyURLPatterns {
			key := entry.Key
			if _, err := d.CreateReadyUrlPattern(ctx, CreateReadyUrlPatternParams{
				Key:       &key,
				Label:     entry.Label,
				Pattern:   entry.Pattern,
				Flags:     entry.Flags,
				SortOrder: int64(i),
			}); err != nil {
				return err
			}
		}
		return nil
	}

	byPattern := map[string]ReadyUrlPattern{}
	for _, row := range rows {
		byPattern[row.Pattern] = row
	}
	for i, entry := range defaultReadyURLPatterns {
		legacy, ok := byPattern[entry.Pattern]
		if ok && (legacy.Key == nil || *legacy.Key == "") {
			key := entry.Key
			if err := d.UpdateReadyUrlPatternSeed(ctx, UpdateReadyUrlPatternSeedParams{
				Key:       &key,
				Label:     entry.Label,
				Flags:     entry.Flags,
				SortOrder: int64(i),
				ID:        legacy.ID,
			}); err != nil {
				return err
			}
		}
	}

	rows, err = d.ListReadyUrlPatterns(ctx)
	if err != nil {
		return err
	}
	existingKeys := map[string]struct{}{}
	for _, row := range rows {
		if row.Key != nil && *row.Key != "" {
			existingKeys[*row.Key] = struct{}{}
		}
	}
	maxOrderRaw, err := d.MaxReadyUrlSortOrder(ctx)
	if err != nil {
		return err
	}
	next := toInt64(maxOrderRaw) + 1
	for _, entry := range defaultReadyURLPatterns {
		if _, ok := existingKeys[entry.Key]; ok {
			continue
		}
		key := entry.Key
		if _, err := d.CreateReadyUrlPattern(ctx, CreateReadyUrlPatternParams{
			Key:       &key,
			Label:     entry.Label,
			Pattern:   entry.Pattern,
			Flags:     entry.Flags,
			SortOrder: next,
		}); err != nil {
			return err
		}
		next++
	}
	return nil
}

func (d *DB) CreateReadyURLPatternUser(ctx context.Context, label, pattern, flags string) (ReadyUrlPattern, error) {
	maxOrderRaw, err := d.MaxReadyUrlSortOrder(ctx)
	if err != nil {
		return ReadyUrlPattern{}, err
	}
	return d.CreateReadyUrlPattern(ctx, CreateReadyUrlPatternParams{
		Key:       nil,
		Label:     label,
		Pattern:   pattern,
		Flags:     flags,
		SortOrder: toInt64(maxOrderRaw) + 1,
	})
}

func UniqueErr(err error, message string) error {
	if err == nil {
		return nil
	}
	if isUnique(err) {
		return fmt.Errorf("%s", message)
	}
	return err
}
