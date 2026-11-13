package config

import (
	"fmt"

	_ "embed"
	"gopkg.in/ini.v1"
)

//go:embed config.ini
var defaultIni []byte

type Config struct {
	Server struct {
		ServerPort int `ini:"port"`
	}
}

func Load(cfgPath string) (Config, error) {
	sources := []any{defaultIni}
	if cfgPath != "" {
		sources = append(sources, cfgPath)
	}

	iniFile, err := ini.Load(sources[0], sources[1:]...)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := Config{}

	// Server section
	if err := iniFile.Section("server").MapTo(&cfg.Server); err != nil {
		return Config{}, fmt.Errorf("section [server] parsing failed: %w", err)
	}

	// TODO: validate configs before return

	return cfg, nil
}
