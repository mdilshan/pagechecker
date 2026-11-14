package config

import (
	"fmt"
	"time"

	_ "embed"
	"gopkg.in/ini.v1"
)

//go:embed config.ini
var defaultIni []byte

type Config struct {
	Server struct {
		ServerPort int  `ini:"port"`
		Pprof      bool `ini:"pprof"`
	}
	Downloader struct {
		Timeout   time.Duration `ini:"timeout"`
		MaxBodyMB int           `ini:"max_body_mb"`
	}
	LinkChecker struct {
		Timeout       time.Duration `ini:"timeout"`
		Workers       int           `ini:"workers"`
		MaxConcurrent int           `ini:"max_concurrent"`
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

	if err := iniFile.Section("server").MapTo(&cfg.Server); err != nil {
		return Config{}, fmt.Errorf("section [server] parsing failed: %w", err)
	}

	err = iniFile.Section("downloader").MapTo(&cfg.Downloader)
	if err != nil {
		return Config{},
			fmt.Errorf("section [downloader] parsing failed: %w", err)
	}

	err = iniFile.Section("linkchecker").MapTo(&cfg.LinkChecker)
	if err != nil {
		return Config{},
			fmt.Errorf("section [linkchecker] parsing failed: %w", err)
	}

	return cfg, nil
}
