package migration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	dtoken "github.com/Zany2/dtoken-go"
	base64codec "github.com/Zany2/dtoken-go/codec/base64"
	jsoncodec "github.com/Zany2/dtoken-go/codec/json"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/storage/memory"
)

type upgradeCountingCodec struct {
	adapter.Codec
	decodes int
}

func (c *upgradeCountingCodec) Decode(data []byte, target any) error {
	c.decodes++
	return c.Codec.Decode(data, target)
}

func putEncoded(t *testing.T, store adapter.Storage, codec adapter.Codec, key string, value any, ttl time.Duration) {
	t.Helper()
	data, err := codec.Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	put(t, store, key, data, ttl)
}

func TestTokenUpgradeLifecycle(t *testing.T) {
	for _, codec := range []adapter.Codec{jsoncodec.NewJSONSerializer(), base64codec.NewBase64Serializer()} {
		t.Run(codec.Name(), func(t *testing.T) {
			ctx := context.Background()
			store := memory.NewStorage()
			now := time.Now().Unix()
			extra := map[string]any{"trace": "migration"}
			putEncoded(t, store, codec, "old:user:token:access", adapter.TokenInfo{AuthType: "user:", LoginID: "account", Device: "web", CreateTime: now, Timeout: 3600, ActiveTimeout: -1, Extra: extra}, time.Hour)
			putEncoded(t, store, codec, "old:user:refresh:refresh", dtoken.RefreshTokenInfo{AuthType: "user:", LoginID: "account", AccessToken: "access", Device: "web", CreateTime: now, AccessTTL: 3600, ActiveTTL: -1, ExpiresIn: 7200, Extra: extra}, 2*time.Hour)
			put(t, store, "old:user:refresh:token:access", "refresh", 2*time.Hour)
			putEncoded(t, store, codec, "old:user:session:account", adapter.Session{AuthType: "user:", LoginID: "account", TerminalInfos: []adapter.TerminalInfo{{Token: "access", LoginID: "account", Device: "web", CreateTime: now, Index: 1}}, HistoryTerminalCount: 1}, 0)
			put(t, store, "old:user:token:kicked", "KICK_OUT", time.Hour)
			put(t, store, "old:user:custom:other", "unchanged", 0)
			plan, err := PrepareAuthUpgrade(ctx, store, copyOptions(), codec)
			if err != nil {
				t.Fatal(err)
			}
			if got := plan.Summary(); got.UpgradedTokens != 1 || got.UpdatedRefreshTokens != 1 || got.AddedTokenExtras != 1 || got.Records != 7 {
				t.Fatalf("summary = %+v", got)
			}
			if store.Exists(ctx, "new:user:v2:token:access") {
				t.Fatal("preflight wrote destination")
			}
			var accessDeadline, extraDeadline, refreshDeadline time.Time
			for _, entry := range plan.records {
				switch {
				case entry.key == "new:user:v2:token:access":
					accessDeadline = entry.deadline
				case strings.HasPrefix(entry.key, "new:user:v2:token-extra:"):
					extraDeadline = entry.deadline
				case entry.key == "new:user:v2:refresh:refresh":
					refreshDeadline = entry.deadline
				}
			}
			if accessDeadline.IsZero() || !extraDeadline.Equal(accessDeadline) || !refreshDeadline.After(accessDeadline.Add(30*time.Minute)) {
				t.Fatal("conversion changed independent deadlines")
			}
			if result, err := plan.Apply(ctx); err != nil || result.Written != 7 {
				t.Fatalf("Apply = %+v, %v", result, err)
			}
			raw, err := store.Get(ctx, "new:user:v2:token:access")
			if err != nil || !strings.HasPrefix(raw.(string), "dtk:2:") {
				t.Fatalf("primary = %v, %v", raw, err)
			}
			value, _ := store.Get(ctx, "new:user:v2:custom:other")
			if value != "unchanged" {
				t.Fatal("custom data changed")
			}
			counted := &upgradeCountingCodec{Codec: codec}
			auth, err := dtoken.NewBuilder().AuthType("user").KeyPrefix("new:").Storage(store).SetCodec(counted).AutoRenew(false).BuildInstance()
			if err != nil {
				t.Fatal(err)
			}
			defer auth.Close()
			before := counted.decodes
			if err := auth.CheckLogin(ctx, "access"); err != nil {
				t.Fatal(err)
			}
			if counted.decodes != before {
				t.Fatal("migrated login check still decodes through Codec")
			}
			if auth.IsLogin(ctx, "kicked") {
				t.Fatal("kicked credential revived")
			}
			info, err := auth.GetTokenInfo(ctx, "access")
			if err != nil || info.Extra["trace"] != "migration" {
				t.Fatalf("details = %+v, %v", info, err)
			}
			if err := auth.RenewTimeout(ctx, "access", 30*time.Minute); err != nil {
				t.Fatal(err)
			}
			pair, err := auth.RefreshToken(ctx, "refresh")
			if err != nil {
				t.Fatal(err)
			}
			if auth.IsLogin(ctx, "access") || !auth.IsLogin(ctx, pair.AccessToken) {
				t.Fatal("rotation did not retire the original lifecycle")
			}
			if _, err := auth.RefreshToken(ctx, "refresh"); !errors.Is(err, derror.ErrInvalidRefreshToken) {
				t.Fatalf("repeat refresh = %v", err)
			}
			if err := auth.RevokeRefreshToken(ctx, pair.RefreshToken); err != nil {
				t.Fatal(err)
			}
			if auth.IsLogin(ctx, pair.AccessToken) {
				t.Fatal("revoked replacement is still valid")
			}
			if !store.Exists(ctx, "old:user:token:access") || !store.Exists(ctx, "old:user:refresh:refresh") {
				t.Fatal("source modified")
			}
		})
	}
}

func TestTokenUpgradeExpiredAccessStillRefreshes(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStorage()
	codec := jsoncodec.NewJSONSerializer()
	putEncoded(t, store, codec, "old:user:token:access", adapter.TokenInfo{LoginID: "account", CreateTime: time.Now().Unix(), Timeout: 60, ActiveTimeout: -1}, time.Minute)
	putEncoded(t, store, codec, "old:user:refresh:refresh", dtoken.RefreshTokenInfo{LoginID: "account", AccessToken: "access", CreateTime: time.Now().Unix(), AccessTTL: 60, ActiveTTL: -1, ExpiresIn: 3600}, time.Hour)
	put(t, store, "old:user:refresh:token:access", "refresh", time.Hour)
	plan, err := PrepareAuthUpgrade(ctx, store, copyOptions(), codec)
	if err != nil {
		t.Fatal(err)
	}
	for i := range plan.records {
		if plan.records[i].key == "new:user:v2:token:access" {
			plan.records[i].deadline = time.Now().Add(-time.Second)
		}
	}
	if result, err := plan.Apply(ctx); err != nil || result.Expired != 1 {
		t.Fatalf("Apply = %+v, %v", result, err)
	}
	auth, err := dtoken.NewBuilder().AuthType("user").KeyPrefix("new:").Storage(store).SetCodec(codec).AutoRenew(false).BuildInstance()
	if err != nil {
		t.Fatal(err)
	}
	defer auth.Close()
	if auth.IsLogin(ctx, "access") {
		t.Fatal("expired access revived")
	}
	if pair, err := auth.RefreshToken(ctx, "refresh"); err != nil || pair.AccessToken == "" {
		t.Fatalf("refresh after expiry = %+v, %v", pair, err)
	}
}

func TestTokenUpgradeFailureIsReadOnly(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStorage()
	codec := jsoncodec.NewJSONSerializer()
	putEncoded(t, store, codec, "old:user:token:a", adapter.TokenInfo{AuthType: "foreign:", LoginID: "account"}, time.Hour)
	if plan, err := PrepareAuthUpgrade(ctx, store, copyOptions(), codec); err == nil || plan != nil {
		t.Fatal("foreign credential accepted")
	}
	if _, err := PrepareAuthUpgrade(ctx, store, copyOptions(), nil); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("nil codec = %v", err)
	}
	keys, err := store.Keys(ctx, "*")
	if err != nil || len(keys) != 1 || keys[0] != "old:user:token:a" {
		t.Fatalf("failed preflight changed storage: %v, %v", keys, err)
	}
}

func TestTokenUpgradeStaleRefreshCannotRevokeReusedAccess(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStorage()
	codec := jsoncodec.NewJSONSerializer()
	putEncoded(t, store, codec, "old:user:token:access", adapter.TokenInfo{LoginID: "new-account", CreateTime: 200, Timeout: 3600, ActiveTimeout: -1}, time.Hour)
	putEncoded(t, store, codec, "old:user:refresh:refresh", dtoken.RefreshTokenInfo{LoginID: "old-account", AccessToken: "access", CreateTime: 100, AccessTTL: 3600, ActiveTTL: -1}, time.Hour)
	put(t, store, "old:user:refresh:token:access", "refresh", time.Hour)
	plan, err := PrepareAuthUpgrade(ctx, store, copyOptions(), codec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	auth, err := dtoken.NewBuilder().AuthType("user").KeyPrefix("new:").Storage(store).SetCodec(codec).AutoRenew(false).BuildInstance()
	if err != nil {
		t.Fatal(err)
	}
	defer auth.Close()
	if err := auth.RevokeRefreshToken(ctx, "refresh"); err != nil {
		t.Fatal(err)
	}
	if id, err := auth.GetLoginID(ctx, "access"); err != nil || id != "new-account" {
		t.Fatalf("reused access = %q, %v", id, err)
	}
}
