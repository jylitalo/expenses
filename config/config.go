package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Directory string  `yaml:"directory"`
	Large     float64 `yaml:"large"`
}

type EventRecord struct {
	Year   int
	Month  int
	Day    int
	Name   string
	Amount float64
}

type configCtxKey string

const configKey configCtxKey = "expenses.config"

func configFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error in UserHomeDir: %w", err)
	}
	return home + "/.expenses.yaml", nil
}

func Get(ctx context.Context) (*Config, error) {
	cfg := ctx.Value(configKey)
	if cfg == nil {
		return nil, errors.New("config not found from context")
	}
	v, ok := cfg.(*Config)
	if !ok {
		return nil, errors.New("config type conversion failed")
	}
	return v, nil
}

func Read(ctx context.Context) (context.Context, error) {
	setLogger()
	fname, err := configFile()
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(filepath.Clean(fname))
	if err != nil {
		return nil, fmt.Errorf("error in reading .expenses.yaml")
	}
	cfg := Config{}
	if err = yaml.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("error in parsing .expenses.yaml")
	}
	ctx = context.WithValue(ctx, configKey, &cfg)
	return ctx, err
}

func ReadOPEvents(ctx context.Context) ([]EventRecord, error) {
	cfg, err := Get(ctx)
	if err != nil {
		return nil, err
	}
	pattern := cfg.Directory + "/OP/*/*.csv"
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
			d, errDate := time.Parse(time.RFC3339, fields[1]+"T12:00:00Z")
			a, errFloat := strconv.ParseFloat(strings.Replace(fields[2], ",", ".", 1), 64)
			if err := errors.Join(errDate, errFloat); err != nil {
				return nil, err
			}
			events = append(events, EventRecord{
				Year:   d.Year(),
				Month:  int(d.Month()),
				Day:    d.Day(),
				Amount: a,
				Name:   fields[5],
			})
		}
	}
	fmt.Printf("Found %d events\n", eventsCount)
	return events, err
}

func setLogger() {
	lvl := &slog.LevelVar{}
	// lvl.Set(slog.LevelDebug)
	lvl.Set(slog.LevelInfo)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))
	slog.SetDefault(logger)
}
