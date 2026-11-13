//	go test ./config/ -v

package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.ServerPort != 8000 {
		t.Errorf("port = %d, want 8000", cfg.Server.ServerPort)
	}
	if cfg.Downloader.Timeout != 30*time.Second {
		t.Errorf("downloader timeout = %v", cfg.Downloader.Timeout)
	}
	if cfg.LinkChecker.Workers != 10 {
		t.Errorf("workers = %d, want 10", cfg.LinkChecker.Workers)
	}
}
