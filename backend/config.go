package main

import (
	"os"
	"strconv"
)

type Config struct{ Port string }

// Settings carries operator-tunable maintenance defaults loaded from the
// environment at startup.
type Settings struct {
	LabelDefaults map[string]string
	Technicians   []string
	MaxBatchSize  int
	NotifyRetry   int
}

// SettingsProvider abstracts where runtime settings come from so callers can
// be tested against static values.
type SettingsProvider interface {
	Settings() Settings
}

type staticSettingsProvider struct{ settings Settings }

func (p staticSettingsProvider) Settings() Settings { return p.settings }

// NewSettingsProvider wraps a concrete Settings value in a provider.
func NewSettingsProvider(settings Settings) SettingsProvider {
	return staticSettingsProvider{settings: settings}
}

func loadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{Port: port}
}

func loadSettings() Settings {
	settings := Settings{
		LabelDefaults: map[string]string{"site": "A-01", "region": "north"},
		Technicians:   []string{"t-01", "t-02", "t-03"},
		MaxBatchSize:  50,
		NotifyRetry:   3,
	}
	if site := os.Getenv("SOLAR_SITE"); site != "" {
		settings.LabelDefaults["site"] = site
	}
	if value := os.Getenv("SOLAR_MAX_BATCH"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			settings.MaxBatchSize = parsed
		}
	}
	return settings
}
