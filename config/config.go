package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Directory string `yaml:"directory"`
	Silent    struct {
		Min         float64  `yaml:"min"`
		Max         float64  `yaml:"max"`
		Explanation []string `yaml:"explanation"`
		Names       []string `yaml:"names"`
	} `yaml:"silent"`
	Excluded struct {
		Accounts    []string `yaml:"accounts"`
		Explanation []string `yaml:"explanation"`
		Names       []string `yaml:"names"`
	} `yaml:"exclude"`
}

type EventRecord struct {
	Year        int
	Month       int
	Day         int
	Explanation string
	Name        string
	Account     string
	Amount      float64
	Labels      string
	Bank        string
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

func setLogger() {
	lvl := &slog.LevelVar{}
	// lvl.Set(slog.LevelDebug)
	lvl.Set(slog.LevelInfo)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))
	slog.SetDefault(logger)
}
