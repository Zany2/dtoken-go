package dtoken

import (
	"context"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/logging/dlog"
	"github.com/Zany2/dtoken-go/pool/ants"
)

// TestBuilderMemoryConfiguration checks config ownership and bundled pool validation at construction. TestBuilderMemoryConfiguration 验证配置所有权和构造时的内置池校验。
func TestBuilderMemoryConfiguration(t *testing.T) {
	cfg := DefaultConfig()
	cfg.IsPrintBanner = false
	poolConfig := ants.DefaultRenewPoolConfig()
	poolConfig.MinSize = 1
	poolConfig.MaxSize = 2
	logConfig := dlog.DefaultLoggerConfig()
	logConfig.StdoutOnly = true
	b := NewBuilder().Config(cfg).RenewPoolConfig(poolConfig).LoggerConfig(logConfig).
		IsReadBody(true).IsReadQuery(true).IsReadHeader(false).LoggerTimeFormat(time.RFC3339).
		RenewPoolScaleUpRate(0.8).RenewPoolScaleDownRate(0.2).RenewPoolCheckInterval(time.Second).
		RenewPoolExpiry(time.Minute).RenewPoolPreAlloc(false).RenewPoolNonBlocking(true)
	// Mutating caller-owned values must not change the configured build. 修改调用方持有的值不得改变已配置的构造器。
	poolConfig.MinSize = 0
	logConfig.TimeFormat = "mutated"
	cfg.IsReadHeader = true
	if b.GetRenewPoolConfig().MinSize != 1 || b.GetLoggerConfig().TimeFormat != time.RFC3339 {
		t.Fatal("builder retained caller configuration pointers")
	}
	auth, err := b.BuildInstance()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(auth.Close)
	b.IsReadHeader(true).IsReadBody(false).IsReadQuery(false)
	got := auth.Manager().GetConfig()
	if got.IsReadHeader || !got.IsReadBody || !got.IsReadQuery {
		t.Fatalf("built instance changed with builder: %+v", got)
	}
	ctx := context.Background()
	token, err := auth.LoginID(ctx, "configured")
	if err != nil || !auth.IsLogin(ctx, token) {
		t.Fatalf("configured instance login=%q,%v", token, err)
	}
	for name, invalid := range map[string]func(*Builder) *Builder{
		"blocking":          func(b *Builder) *Builder { return b.RenewPoolNonBlocking(false) },
		"preallocation":     func(b *Builder) *Builder { return b.RenewPoolPreAlloc(true) },
		"scale overlap":     func(b *Builder) *Builder { return b.RenewPoolScaleDownRate(0.9) },
		"scale upper bound": func(b *Builder) *Builder { return b.RenewPoolScaleUpRate(2) },
		"check interval":    func(b *Builder) *Builder { return b.RenewPoolCheckInterval(0) },
		"expiry":            func(b *Builder) *Builder { return b.RenewPoolExpiry(0) },
	} {
		t.Run(name, func(t *testing.T) {
			candidate, err := invalid(b.Clone()).BuildInstance()
			if candidate != nil {
				candidate.Close()
			}
			if err == nil {
				t.Fatal("invalid bundled pool configuration accepted")
			}
		})
	}
	if b.GetRenewPoolConfig().Validate() != nil {
		t.Fatal("invalid clone mutated parent")
	}
	reset := b.Clone().RenewPoolConfig(nil).LoggerConfig(nil)
	if *reset.GetRenewPoolConfig() != *ants.DefaultRenewPoolConfig() || *reset.GetLoggerConfig() != *dlog.DefaultLoggerConfig() {
		t.Fatal("nil configuration did not restore defaults")
	}
	if DefaultConfig() == cfg {
		t.Fatal("default config reused caller instance")
	}
}
