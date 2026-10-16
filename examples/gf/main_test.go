package main

import (
	"testing"

	"github.com/Zany2/dtoken-go/dtoken"
)

func TestInitDTokenRegistersManager(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	initDToken()

	mgr, err := dtoken.GetManager()
	if err != nil {
		t.Fatalf("GetManager() error = %v", err)
	}
	cfg := mgr.GetConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("invalid example configuration: %v", err)
	}
	if !cfg.AutoRenew || cfg.Timeout != 7200 || cfg.RenewMaxRefresh != 3600 || cfg.RefreshTokenTimeout != 30*24*60*60 {
		t.Fatalf("unexpected example configuration: %+v", cfg)
	}
}
