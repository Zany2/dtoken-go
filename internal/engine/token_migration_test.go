package engine

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	jsoncodec "github.com/Zany2/dtoken-go/codec/json"
	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
)

// upgradeTestTokenPair converts an isolated legacy fixture before runtime operations. upgradeTestTokenPair 在运行时操作前转换隔离的旧令牌对用例。
func upgradeTestTokenPair(t *testing.T, mgr *Manager, pair *RefreshTokenPair) {
	t.Helper()
	ctx := context.Background()
	values := make(map[string]any)
	deadlines := make(map[string]time.Time)
	for _, key := range []string{config.TokenKeyPrefix + pair.AccessToken, RefreshTokenKeyPrefix + pair.RefreshToken} {
		value, err := mgr.storage.Get(ctx, mgr.storageNamespace()+key)
		if err != nil || value == nil {
			t.Fatalf("read migration fixture %q: %v", key, err)
		}
		values[key] = value
		now := time.Now()
		ttl, err := mgr.storage.TTL(ctx, mgr.storageNamespace()+key)
		if err != nil || ttl != adapter.TTLNoExpire && ttl <= 0 {
			t.Fatalf("migration fixture TTL %q: %v, %v", key, ttl, err)
		}
		if ttl > 0 {
			deadlines[key] = now.Add(ttl)
		}
	}
	changes, err := UpgradeTokenSnapshot(ctx, mgr.serializer, mgr.config.AuthType, values)
	if err != nil {
		t.Fatal(err)
	}
	for key, change := range changes {
		var ttl time.Duration
		if deadline := deadlines[change.TTLSource]; !deadline.IsZero() {
			ttl = time.Until(deadline)
			if ttl <= 0 {
				t.Fatal("migration fixture expired")
			}
		}
		if err := mgr.storage.Set(ctx, mgr.storageNamespace()+key, change.Value, ttl); err != nil {
			t.Fatal(err)
		}
	}
}

func migrationEncoded(t *testing.T, value any) []byte {
	t.Helper()
	data, err := jsoncodec.NewJSONSerializer().Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestUpgradeSnapshotPreservesBindingDecisions(t *testing.T) {
	codec := jsoncodec.NewJSONSerializer()
	for _, scenario := range []string{"matched", "replaced", "missing", "identified"} {
		t.Run(scenario, func(t *testing.T) {
			access := tokenRecord{TokenInfo: TokenInfo{LoginID: "account", Device: "web", CreateTime: 20, Timeout: 3600, ActiveTimeout: -1}}
			refresh := refreshTokenRecord{RefreshTokenInfo: RefreshTokenInfo{LoginID: "account", AccessToken: "a", Device: "web", CreateTime: 21, AccessTTL: 3600, ActiveTTL: -1}}
			if scenario == "replaced" {
				access.CreateTime = 22
			}
			if scenario == "identified" {
				access.AccessID, refresh.AccessID = "existing", "existing"
			}
			values := map[string]any{"refresh:r": migrationEncoded(t, refresh), "refresh:token:a": "r"}
			if scenario != "missing" {
				values["token:a"] = migrationEncoded(t, access)
			}
			changes, err := UpgradeTokenSnapshot(context.Background(), codec, "user:", values)
			if err != nil {
				t.Fatal(err)
			}
			var gotRefresh refreshTokenRecord
			if err := decodeMigrationValue(codec, changes["refresh:r"].Value, &gotRefresh); err != nil {
				t.Fatal(err)
			}
			if gotRefresh.AccessID == "" {
				t.Fatal("anonymous refresh retained")
			}
			if scenario == "missing" {
				return
			}
			gotAccess, err := decodeTokenRecord(changes["token:a"].Value.(string))
			if err != nil {
				t.Fatal(err)
			}
			if refreshMatchesAccessRecord(&gotRefresh, gotAccess) != refreshMatchesAccessRecord(&refresh, &access) {
				t.Fatal("migration changed lifecycle ownership")
			}
			if scenario == "identified" && gotAccess.AccessID != "existing" {
				t.Fatal("existing ID changed")
			}
		})
	}
}

func TestUpgradeSnapshotPreservesExtraAndRevocation(t *testing.T) {
	codec := jsoncodec.NewJSONSerializer()
	for _, extra := range []map[string]any{nil, {}, {"value": "preserved"}} {
		legacy := tokenRecord{TokenInfo: TokenInfo{LoginID: "account", Timeout: -1, ActiveTimeout: -1, Extra: extra}, AccessID: "existing", TerminalIndex: 9, Revoked: true}
		// Encode explicitly to retain an empty map despite the legacy omitempty tag. 显式编码以保留旧 omitempty 标签会省略的空 map。
		value := map[string]any{"loginId": legacy.LoginID, "timeout": -1, "activeTimeout": -1, "extra": extra, "accessId": "existing", "terminalIndex": 9, "revoked": true}
		values := map[string]any{"token:a": migrationEncoded(t, value)}
		changes, err := UpgradeTokenSnapshot(context.Background(), codec, "user:", values)
		if err != nil {
			t.Fatal(err)
		}
		got, err := decodeTokenRecord(changes["token:a"].Value.(string))
		if err != nil || !got.Revoked || got.AccessID != "existing" || got.TerminalIndex != 9 || got.HasExtra != (extra != nil) {
			t.Fatalf("converted = %+v, %v", got, err)
		}
		if extra != nil {
			entry := changes["token-extra:existing"]
			var gotExtra map[string]any
			if err := decodeMigrationValue(codec, entry.Value, &gotExtra); err != nil || !reflect.DeepEqual(extra, gotExtra) || entry.TTLSource != "token:a" {
				t.Fatalf("extension = %v, %v", gotExtra, err)
			}
		}
		for key, change := range changes {
			values[key] = change.Value
		}
		if again, err := UpgradeTokenSnapshot(context.Background(), codec, "user:", values); err != nil || len(again) != 0 {
			t.Fatalf("converted snapshot is not stable: %v, %v", again, err)
		}
	}
}

func TestUpgradeSnapshotRejectsAmbiguousOrCorruptData(t *testing.T) {
	codec := jsoncodec.NewJSONSerializer()
	valid := tokenRecord{TokenInfo: TokenInfo{AuthType: "user:", LoginID: "account"}, AccessID: "id"}
	withExtra := valid
	withExtra.HasExtra = true
	for name, values := range map[string]map[string]any{
		"unknown format":        {"token:a": "dtk:99:invalid"},
		"null token":            {"token:a": "null"},
		"null session":          {"session:account": "null"},
		"foreign session":       {"session:account": migrationEncoded(t, Session{AuthType: "admin:", LoginID: "account"})},
		"foreign refresh":       {"refresh:r": migrationEncoded(t, RefreshTokenInfo{AuthType: "admin:", LoginID: "account", AccessToken: "a"})},
		"duplicate identity":    {"token:a": encodeTokenRecord(&valid), "token:b": encodeTokenRecord(&valid)},
		"missing extension":     {"token:a": encodeTokenRecord(&withExtra)},
		"null extension":        {"token:a": encodeTokenRecord(&withExtra), "token-extra:id": "null"},
		"wrong reverse binding": {"refresh:r": migrationEncoded(t, RefreshTokenInfo{LoginID: "account", AccessToken: "b"}), "refresh:token:a": "r"},
	} {
		t.Run(name, func(t *testing.T) {
			if changes, err := UpgradeTokenSnapshot(context.Background(), codec, "user:", values); err == nil || changes != nil {
				t.Fatal("invalid snapshot accepted")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := UpgradeTokenSnapshot(ctx, codec, "user:", map[string]any{"token:a": encodeTokenRecord(&valid)}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
	if _, err := UpgradeTokenSnapshot(context.Background(), nil, "user:", nil); !errors.Is(err, derror.ErrInvalidParam) {
		t.Fatalf("nil codec = %v", err)
	}
}
