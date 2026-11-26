package engine

import (
	"context"
	"errors"
	"testing"

	jsoncodec "github.com/Zany2/dtoken-go/codec/json"
	derror "github.com/Zany2/dtoken-go/errors"
)

func TestDisableMigrationIdentityAndKeyFormats(t *testing.T) {
	codec := jsoncodec.NewJSONSerializer()
	mgr := newTestManager(t, nil)
	for _, tc := range []struct {
		name              string
		marker            disableMarker
		owner             string
		kind              string
		legacy, canonical string
	}{
		{"account", disableMarker{}, "account", "account", mgr.getLegacyDisableKey("account"), mgr.getDisableKey("account")},
		{"reserved account", disableMarker{}, "service:account:pay", "account", mgr.getLegacyDisableKey("service:account:pay"), mgr.getDisableKey("service:account:pay")},
		{"service", disableMarker{Service: "pay:invoice", Level: 3}, "team:one", "service", mgr.getLegacyDisableServiceKey("team:one", "pay:invoice"), mgr.getDisableServiceKey("team:one", "pay:invoice")},
		{"device", disableMarker{Device: "web:mobile"}, "team:two", "device", mgr.getLegacyDisableDeviceKey("team:two", "web:mobile"), mgr.getDisableDeviceKey("team:two", "web:mobile")},
		{"concrete device", disableMarker{Device: "web", DeviceID: "phone:one"}, "team:three", "device", mgr.getLegacyDisableDeviceAndDeviceIDKey("team:three", "web", "phone:one"), mgr.getDisableDeviceAndDeviceIDKey("team:three", "web", "phone:one")},
		{"explicit backslash", disableMarker{LoginID: `team\:one`, Kind: "account"}, `team\:one`, "account", mgr.getLegacyDisableKey(`team\:one`), mgr.getDisableKey(`team\:one`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.marker.DisableInfo = DisableInfo{DisableTime: 123, DisableReason: "preserved"}
			source := tc.legacy[len(mgr.storageNamespace()):]
			target := tc.canonical[len(mgr.storageNamespace()):]
			values := map[string]any{source: migrationEncoded(t, tc.marker), "custom:value": "untouched"}
			changes, err := UpgradeDisableSnapshot(context.Background(), codec, values)
			if err != nil {
				t.Fatal(err)
			}
			change, exists := changes[source]
			if !exists || change.Key != target || len(changes) != 1 {
				t.Fatalf("changes = %+v", changes)
			}
			var got disableMarker
			if err := decodeMigrationValue(codec, change.Value, &got); err != nil {
				t.Fatal(err)
			}
			want := tc.marker
			want.LoginID, want.Kind = tc.owner, tc.kind
			if got != want {
				t.Fatalf("marker = %+v, want %+v", got, want)
			}
			if again, err := UpgradeDisableSnapshot(context.Background(), codec, map[string]any{target: change.Value}); err != nil || len(again) != 0 {
				t.Fatalf("canonical conversion = %+v, %v", again, err)
			}
		})
	}
}

func TestDisableMigrationRejectsAmbiguityAndConflicts(t *testing.T) {
	codec := jsoncodec.NewJSONSerializer()
	for name, values := range map[string]map[string]any{
		"ownerless backslash":  {`disable:user\:one`: migrationEncoded(t, DisableInfo{})},
		"kind without owner":   {"disable:account": migrationEncoded(t, disableMarker{Kind: "account"})},
		"different owner":      {"disable:account": migrationEncoded(t, disableMarker{Kind: "account", LoginID: "another"})},
		"different kind":       {"disable:service:account:pay": migrationEncoded(t, disableMarker{Kind: "account", LoginID: "account", Service: "pay"})},
		"unknown kind":         {"disable:account": migrationEncoded(t, disableMarker{Kind: "unknown", LoginID: "account"})},
		"mixed fields":         {"disable:service:account:pay": migrationEncoded(t, disableMarker{Service: "pay", Device: "web"})},
		"missing device":       {"disable:device:id:account:web:id": migrationEncoded(t, disableMarker{DeviceID: "id"})},
		"wrong service suffix": {"disable:service:account:pay": migrationEncoded(t, disableMarker{Service: "billing"})},
		"empty account":        {"disable:": migrationEncoded(t, DisableInfo{})},
		"negative level":       {"disable:service:account:pay": migrationEncoded(t, disableMarker{Service: "pay", Level: -1})},
		"untrimmed device":     {"disable:device:account: web ": migrationEncoded(t, disableMarker{Device: " web "})},
		"null":                 {"disable:account": "null"},
		"corrupt":              {"disable:account": "invalid"},
		"duplicate destination": {
			"disable:user:one":  migrationEncoded(t, DisableInfo{DisableReason: "old"}),
			`disable:user\:one`: migrationEncoded(t, disableMarker{LoginID: "user:one", Kind: "account", DisableInfo: DisableInfo{DisableReason: "current"}}),
		},
	} {
		t.Run(name, func(t *testing.T) {
			if changes, err := UpgradeDisableSnapshot(context.Background(), codec, values); err == nil || changes != nil {
				t.Fatal("invalid restrictions accepted")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := UpgradeDisableSnapshot(ctx, codec, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation = %v", err)
	}
	if _, err := UpgradeDisableSnapshot(context.Background(), nil, nil); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("nil codec = %v", err)
	}
}

func TestDisableMigrationMovesOverlappingSourceKeysIndependently(t *testing.T) {
	codec := jsoncodec.NewJSONSerializer()
	values := map[string]any{
		"disable:user:one":  migrationEncoded(t, disableMarker{LoginID: "user:one", Kind: "account", DisableInfo: DisableInfo{DisableReason: "first"}}),
		`disable:user\:one`: migrationEncoded(t, disableMarker{LoginID: `user\:one`, Kind: "account", DisableInfo: DisableInfo{DisableReason: "second"}}),
	}
	changes, err := UpgradeDisableSnapshot(context.Background(), codec, values)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 || changes["disable:user:one"].Key != `disable:user\:one` || changes[`disable:user\:one`].Key != `disable:user\\\:one` {
		t.Fatalf("overlapping sources = %+v", changes)
	}
}
