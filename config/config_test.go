//	go test ./config/ -v

package config

import "testing"

func TestLoad(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.ServerPort != 8000 {
		t.Errorf("port = %d, want 8000", cfg.Server.ServerPort)
	}
}
