package main

import (
	dtglobal "github.com/Zany2/dtoken-go/global"
	"testing"
)

func TestInitDTokenRegistersManager(t *testing.T) {
	dtglobal.DeleteAllManager()
	t.Cleanup(dtglobal.DeleteAllManager)
	initDToken()

	mgr, err := dtglobal.GetManager()
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
