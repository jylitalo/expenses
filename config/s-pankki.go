package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

func ReadSPankkiEvents(ctx context.Context) ([]EventRecord, error) {
	cfg, err := Get(ctx)
	if err != nil {
		return nil, err
	}
	pattern := cfg.Directory + "/S-Pankki/*/*.csv"
	fnames, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Found %d csv files from %s\n", len(fnames), pattern)
	events := []EventRecord{}
	eventsCount := 0
	for _, fname := range fnames {
		// fmt.Println("Reading " + fname)
		content, err := os.ReadFile(fname)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(content), "\n")
		for idx, line := range lines {
			if idx == 0 {
				continue
			}
			if len(line) == 0 {
				continue
			}
			// fmt.Printf("Line: %d Content: %v\n", idx, line)
			rawFields := strings.Split(line, ";")
			fields := []string{}
			for _, field := range rawFields {
				fields = append(fields, strings.Trim(field, "\""))
			}
			eventsCount++
			d, errDate := time.Parse("02.01.2006", fields[1])
			a, errFloat := strconv.ParseFloat(strings.Replace(fields[2], ",", ".", 1), 64)
			if err := errors.Join(errDate, errFloat); err != nil {
				return nil, err
			}
			explanation := fields[4]
			name := fields[5]
			account := fields[6]
			labels := []string{}
			if slices.Contains(cfg.Silent.Explanation, explanation) ||
				slices.Contains(cfg.Silent.Names, name) {
				labels = append(labels, "silent")
			}
			if slices.Contains(cfg.Excluded.Accounts, account) ||
				slices.Contains(cfg.Excluded.Explanation, explanation) ||
				slices.Contains(cfg.Excluded.Names, name) {
				labels = append(labels, "exclude")
			}
			events = append(events, EventRecord{
				Year:        d.Year(),
				Month:       int(d.Month()),
				Day:         d.Day(),
				Explanation: explanation,
				Amount:      a,
				Name:        name,
				Account:     account,
				Labels:      strings.Join(labels, ","),
				Bank:        "S-Pankki",
			})
		}
	}
	fmt.Printf("Found %d S-Pankki events\n", eventsCount)
	return events, err
}
