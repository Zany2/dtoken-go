package migration

import (
	"context"
	"errors"
	"testing"
	"time"

	dtoken "github.com/Zany2/dtoken-go"
	base64codec "github.com/Zany2/dtoken-go/codec/base64"
	jsoncodec "github.com/Zany2/dtoken-go/codec/json"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/storage/memory"
)

func TestAuthUpgradeRestrictionLifecycle(t *testing.T) {
	ctx := context.Background()
	for _, codec := range []adapter.Codec{jsoncodec.NewJSONSerializer(), base64codec.NewBase64Serializer()} {
		t.Run(codec.Name(), func(t *testing.T) {
			store := memory.NewStorage()
			cases := []struct {
				source, target string
				value          any
				ttl            time.Duration
				check          func(*dtoken.Auth) error
				untie          func(*dtoken.Auth) error
				blocked        error
			}{
				{"disable:service:account", `disable:service\:account`, adapter.DisableInfo{DisableTime: 123, DisableReason: "account"}, 0,
					func(a *dtoken.Auth) error { return a.CheckDisable(ctx, "service:account") }, func(a *dtoken.Auth) error { return a.Untie(ctx, "service:account") }, derror.ErrAccountDisabled},
				{"disable:service:team:one:pay:invoice", `disable:service:team\:one:pay\:invoice`, adapter.ServiceDisableInfo{Service: "pay:invoice", Level: 3, DisableTime: 123, DisableReason: "service"}, time.Hour,
					func(a *dtoken.Auth) error { return a.CheckDisableServiceLevel(ctx, "team:one", "pay:invoice", 3) }, func(a *dtoken.Auth) error { return a.UntieService(ctx, "team:one", "pay:invoice") }, derror.ErrServiceDisabled},
				{"disable:device:team:two:web:mobile", `disable:device:team\:two:web\:mobile`, adapter.DeviceDisableInfo{Device: "web:mobile", DisableTime: 123, DisableReason: "device"}, time.Hour,
					func(a *dtoken.Auth) error { return a.CheckDisableDevice(ctx, "team:two", "web:mobile") }, func(a *dtoken.Auth) error { return a.UntieDevice(ctx, "team:two", "web:mobile") }, derror.ErrDeviceDisabled},
				{"disable:device:id:team:three:web:phone:one", `disable:device:id:team\:three:web:phone\:one`, adapter.DeviceDisableInfo{Device: "web", DeviceID: "phone:one", DisableTime: 123, DisableReason: "concrete"}, time.Hour,
					func(a *dtoken.Auth) error {
						return a.CheckDisableDeviceAndDeviceID(ctx, "team:three", "web", "phone:one")
					}, func(a *dtoken.Auth) error { return a.UntieDeviceAndDeviceID(ctx, "team:three", "web", "phone:one") }, derror.ErrDeviceDisabled},
			}
			for _, tc := range cases {
				putEncoded(t, store, codec, "old:user:"+tc.source, tc.value, tc.ttl)
			}
			putEncoded(t, store, codec, "old:user:token:access", adapter.TokenInfo{LoginID: "allowed", Timeout: -1, ActiveTimeout: -1}, 0)
			plan, err := PrepareAuthUpgrade(ctx, store, copyOptions(), codec)
			if err != nil {
				t.Fatal(err)
			}
			if summary := plan.Summary(); summary.UpgradedRestrictions != 4 || summary.UpgradedTokens != 1 || summary.Records != 5 || summary.Permanent != 2 || summary.Expiring != 3 {
				t.Fatalf("summary = %+v", summary)
			}
			keys, err := store.Keys(ctx, "*")
			if err != nil || len(keys) != 5 {
				t.Fatal("preflight modified storage")
			}
			if result, err := plan.Apply(ctx); err != nil || result.Written != 5 {
				t.Fatalf("Apply = %+v, %v", result, err)
			}
			auth, err := dtoken.NewBuilder().AuthType("user").KeyPrefix("new:").Storage(store).SetCodec(codec).AutoRenew(false).BuildInstance()
			if err != nil {
				t.Fatal(err)
			}
			defer auth.Close()
			if err := auth.CheckLogin(ctx, "access"); err != nil {
				t.Fatal(err)
			}
			info, err := auth.GetDisableServiceInfo(ctx, "team:one", "pay:invoice")
			if err != nil || info.Level != 3 || info.DisableTime != 123 || info.DisableReason != "service" {
				t.Fatalf("service metadata = %+v, %v", info, err)
			}
			if auth.IsDisable(ctx, "team:one") || auth.IsDisableService(ctx, "service", "account") {
				t.Fatal("restriction applied to a different identity or kind")
			}
			for _, tc := range cases {
				if err := tc.check(auth); !errors.Is(err, tc.blocked) {
					t.Fatalf("%s check = %v", tc.target, err)
				}
				ttl, err := store.TTL(ctx, "new:user:v2:"+tc.target)
				if err != nil || tc.ttl == 0 && ttl != adapter.TTLNoExpire || tc.ttl > 0 && (ttl <= 0 || ttl > tc.ttl) {
					t.Fatalf("TTL = %v, %v", ttl, err)
				}
				if store.Exists(ctx, "new:user:v2:"+tc.source) {
					t.Fatal("legacy key retained in destination")
				}
				if err := tc.untie(auth); err != nil {
					t.Fatal(err)
				}
				if err := tc.check(auth); err != nil {
					t.Fatalf("restriction survived untie: %v", err)
				}
				if !store.Exists(ctx, "old:user:"+tc.source) || store.Exists(ctx, "new:user:v2:"+tc.target) {
					t.Fatal("untie changed the wrong namespace")
				}
			}
		})
	}
}

func TestAuthUpgradeAmbiguityFailsBeforeWrites(t *testing.T) {
	ctx := context.Background()
	codec := jsoncodec.NewJSONSerializer()
	for _, conflict := range []bool{false, true} {
		store := memory.NewStorage()
		putEncoded(t, store, codec, "old:user:token:access", adapter.TokenInfo{LoginID: "account", Extra: map[string]any{"keep": true}}, time.Hour)
		putEncoded(t, store, codec, `old:user:disable:user\:one`, adapter.DisableInfo{}, time.Hour)
		if conflict {
			putEncoded(t, store, codec, `old:user:disable:user\:one`, map[string]any{"loginId": "user:one", "kind": "account"}, time.Hour)
			putEncoded(t, store, codec, "old:user:disable:user:one", adapter.DisableInfo{}, 0)
		}
		before, err := store.Keys(ctx, "*")
		if err != nil {
			t.Fatal(err)
		}
		plan, err := PrepareAuthUpgrade(ctx, store, copyOptions(), codec)
		if !errors.Is(err, derror.ErrInvalidParam) || plan != nil {
			t.Fatalf("ambiguous plan = %v, %v", plan, err)
		}
		after, err := store.Keys(ctx, "*")
		if err != nil || len(after) != len(before) {
			t.Fatal("failed conversion modified storage")
		}
		for _, key := range before {
			if !store.Exists(ctx, key) {
				t.Fatal("source deleted")
			}
		}
	}
}

func TestAuthUpgradeDoesNotReviveExpiredRestriction(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStorage()
	codec := jsoncodec.NewJSONSerializer()
	putEncoded(t, store, codec, "old:user:disable:team:one", adapter.DisableInfo{}, time.Hour)
	plan, err := PrepareAuthUpgrade(ctx, store, copyOptions(), codec)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.records) != 1 || plan.records[0].deadline.IsZero() {
		t.Fatal("restriction lost its finite deadline")
	}
	plan.records[0].deadline = time.Now().Add(-time.Second)
	if result, err := plan.Apply(ctx); err != nil || result != (Result{Expired: 1}) {
		t.Fatalf("Apply = %+v, %v", result, err)
	}
	if store.Exists(ctx, `new:user:v2:disable:team\:one`) || !store.Exists(ctx, "old:user:disable:team:one") {
		t.Fatal("expired restriction revived or source removed")
	}
}

func TestAuthUpgradeOverlappingSourcesKeepIndependentDeadlines(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStorage()
	codec := jsoncodec.NewJSONSerializer()
	putEncoded(t, store, codec, "old:user:disable:user:one", map[string]any{"loginId": "user:one", "kind": "account", "disableReason": "finite"}, time.Hour)
	putEncoded(t, store, codec, `old:user:disable:user\:one`, map[string]any{"loginId": `user\:one`, "kind": "account", "disableReason": "permanent"}, 0)
	plan, err := PrepareAuthUpgrade(ctx, store, copyOptions(), codec)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := plan.Apply(ctx); err != nil || result.Written != 2 {
		t.Fatalf("Apply = %+v, %v", result, err)
	}
	auth, err := dtoken.NewBuilder().AuthType("user").KeyPrefix("new:").Storage(store).SetCodec(codec).BuildInstance()
	if err != nil {
		t.Fatal(err)
	}
	defer auth.Close()
	for _, tc := range []struct {
		owner, reason string
		permanent     bool
	}{{"user:one", "finite", false}, {`user\:one`, "permanent", true}} {
		info, err := auth.GetDisableInfo(ctx, tc.owner)
		if err != nil || info.DisableReason != tc.reason {
			t.Fatalf("overlapping restriction = %+v, %v", info, err)
		}
		ttl, err := auth.GetDisableTTL(ctx, tc.owner)
		if err != nil || tc.permanent && ttl != -1 || !tc.permanent && (ttl <= 0 || ttl > 3600) {
			t.Fatalf("overlapping TTL = %d, %v", ttl, err)
		}
	}
}
