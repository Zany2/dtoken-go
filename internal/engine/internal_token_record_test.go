package engine

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Zany2/dtoken-go/config"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
)

// TestCompactTokenRecordFormat checks boundaries independently of storage and codecs. TestCompactTokenRecordFormat 独立于存储和编解码器检查格式边界。
func TestCompactTokenRecordFormat(t *testing.T) {
	want := tokenRecord{
		TokenInfo: TokenInfo{AuthType: "user:体系", LoginID: "id:\x00用户", Device: "web:端", DeviceID: "", CreateTime: 42, Timeout: -1, ActiveTimeout: -1},
		AccessID:  "access:id", TerminalIndex: 7, Revoked: true, HasExtra: true,
	}
	encoded := encodeTokenRecord(&want)
	got, err := decodeTokenRecord(encoded)
	if err != nil || !reflect.DeepEqual(got, &want) {
		t.Fatalf("round trip = %+v, %v, want %+v", got, err, want)
	}
	for i := 0; i < len(encoded); i++ {
		if _, err := decodeTokenRecord(encoded[:i]); !errors.Is(err, derror.ErrInvalidToken) {
			t.Fatalf("truncation at %d accepted: %v", i, err)
		}
	}
	for _, malformed := range []string{
		strings.Replace(encoded, "dtk:2:", "dtk:3:", 1), encoded + "tail",
		"dtk:2:9999999999999999999999999:", "dtk:2:-1:x",
		strings.Replace(encoded, "42:-1:", "9223372036854775808:-1:", 1),
		encoded[:len(encoded)-2] + "4:",
	} {
		if _, err := decodeTokenRecord(malformed); !errors.Is(err, derror.ErrInvalidToken) {
			t.Errorf("malformed %q accepted: %v", malformed, err)
		}
	}
	for _, mutate := range []func(*tokenRecord){
		func(r *tokenRecord) { r.LoginID = "" },
		func(r *tokenRecord) { r.AccessID = "" },
		func(r *tokenRecord) { r.CreateTime = -1 },
		func(r *tokenRecord) { r.TerminalIndex = -1 },
		func(r *tokenRecord) { r.Timeout = -2 },
		func(r *tokenRecord) { r.ActiveTimeout = -2 },
	} {
		invalid := want
		mutate(&invalid)
		if _, err := decodeTokenRecord(encodeTokenRecord(&invalid)); !errors.Is(err, derror.ErrInvalidToken) {
			t.Errorf("invalid fields accepted: %+v, %v", invalid, err)
		}
	}
}

// TestCompactTokenValidationSkipsExtra checks actual read and decode costs for varying extension sizes. TestCompactTokenValidationSkipsExtra 检查不同扩展大小下的实际读取和解码次数。
func TestCompactTokenValidationSkipsExtra(t *testing.T) {
	ctx := context.Background()
	for _, input := range []struct {
		name  string
		extra map[string]any
	}{
		{"nil", nil}, {"empty", map[string]any{}},
		{"large", map[string]any{"payload": strings.Repeat("secret-payload", 8192)}},
	} {
		t.Run(input.name, func(t *testing.T) {
			mgr := newTestManager(t, nil)
			token, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: "compact", Device: "web", Extra: input.extra})
			if err != nil {
				t.Fatal(err)
			}
			record, err := mgr.getTokenRecord(ctx, token)
			if err != nil {
				t.Fatal(err)
			}
			if record.Extra != nil || record.HasExtra != (input.extra != nil) {
				t.Fatalf("unexpected compact metadata: %+v", record)
			}
			raw, err := mgr.storage.Get(ctx, mgr.getTokenKey(token))
			if err != nil {
				t.Fatal(err)
			}
			value, ok := raw.(string)
			if !ok || !strings.HasPrefix(value, tokenRecordPrefix) || len(value) > 256 || strings.Contains(value, "secret-payload") {
				t.Fatalf("primary value is not compact: %T", raw)
			}
			extraKey := mgr.getTokenExtraKey(record.AccessID)
			if mgr.storage.Exists(ctx, extraKey) != (input.extra != nil) {
				t.Fatal("unexpected extension key presence")
			}
			storage := &managerGetCountingStorage{Storage: mgr.storage}
			codec := &managerValidationCountingCodec{Codec: mgr.serializer}
			mgr.storage, mgr.serializer = storage, codec

			if !mgr.IsLogin(ctx, token) {
				t.Fatal("IsLogin rejected valid token")
			}
			if err := mgr.CheckLogin(ctx, token); err != nil {
				t.Fatal(err)
			}
			if id, err := mgr.GetLoginID(ctx, token); err != nil || id != "compact" {
				t.Fatalf("GetLoginID = %q, %v", id, err)
			}
			if storage.getCount(mgr.getTokenKey(token)) != 3 || storage.getCount(extraKey) != 0 || storage.getCount(mgr.getSessionKey("compact")) != 0 || codec.tokens.Load() != 0 || codec.extras.Load() != 0 || codec.sessions.Load() != 0 {
				t.Fatal("base checks loaded or decoded full objects")
			}

			info, err := mgr.GetTokenInfo(ctx, token)
			if err != nil || !reflect.DeepEqual(info.Extra, input.extra) {
				t.Fatalf("GetTokenInfo = %+v, %v", info, err)
			}
			wantReads := 0
			if input.extra != nil {
				wantReads = 1
			}
			if storage.getCount(extraKey) != wantReads || codec.extras.Load() != int64(wantReads) {
				t.Fatal("details did not load exactly the required extension")
			}
			introspection, err := mgr.IntrospectToken(ctx, token)
			if err != nil || !introspection.Active || !reflect.DeepEqual(introspection.Extra, input.extra) {
				t.Fatalf("IntrospectToken = %+v, %v", introspection, err)
			}
		})
	}
}

// TestCompactTokenExtraFailureDoesNotBlockRevocation isolates optional detail failures from authentication and logout. TestCompactTokenExtraFailureDoesNotBlockRevocation 将可选详情故障与认证及退出隔离。
func TestCompactTokenExtraFailureDoesNotBlockRevocation(t *testing.T) {
	ctx := context.Background()
	for _, broken := range []string{"missing", "corrupt", "null", "intact"} {
		for _, action := range []string{"logout", "kickout", "replace", "disable"} {
			t.Run(broken+"/"+action, func(t *testing.T) {
				mgr := newTestManager(t, func(cfg *config.Config) { cfg.Timeout = config.NoLimit })
				token, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: "extra-failure", Device: "web", Extra: map[string]any{"kind": "detail"}})
				if err != nil {
					t.Fatal(err)
				}
				record, err := mgr.getTokenRecord(ctx, token)
				if err != nil {
					t.Fatal(err)
				}
				key := mgr.getTokenExtraKey(record.AccessID)
				switch broken {
				case "missing":
					err = mgr.storage.Delete(ctx, key)
				case "corrupt":
					err = mgr.storage.Set(ctx, key, "{", 0)
				case "null":
					err = mgr.storage.Set(ctx, key, "null", 0)
				}
				if err != nil {
					t.Fatal(err)
				}
				if err = mgr.CheckLogin(ctx, token); err != nil {
					t.Fatal(err)
				}
				if _, err = mgr.GetTokenInfo(ctx, token); (err != nil) != (broken != "intact") {
					t.Fatalf("details error = %v", err)
				}
				switch action {
				case "logout":
					err = mgr.Logout(ctx, token)
				case "kickout":
					err = mgr.Kickout(ctx, token)
				case "replace":
					err = mgr.Replace(ctx, token)
				case "disable":
					err = mgr.Disable(ctx, "extra-failure", time.Minute)
				}
				if err != nil {
					t.Fatal(err)
				}
				if mgr.IsLogin(ctx, token) {
					t.Fatal("revoked token is still active")
				}
				// Account disable retains metadata for inspection; explicit terminal retirement removes it. 账号封禁保留可查元数据；显式终端下线删除扩展。
				if action != "disable" && mgr.storage.Exists(ctx, key) {
					t.Fatal("revocation left extension data")
				}
			})
		}
	}
}

// TestCompactTokenExtraRenewal checks finite, unlimited, shared and automatic renewal. TestCompactTokenExtraRenewal 检查有限、无限、共享和自动续期。
func TestCompactTokenExtraRenewal(t *testing.T) {
	ctx := context.Background()
	for _, action := range []string{"manual", "unlimited", "shared", "automatic"} {
		t.Run(action, func(t *testing.T) {
			mgr := newTestManager(t, func(cfg *config.Config) {
				cfg.Timeout = 60
				if action == "unlimited" {
					cfg.Timeout = config.NoLimit
				}
			})
			extra := map[string]any{"payload": "preserved"}
			token, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: "extra-renewal", Device: "web", Extra: extra})
			if err != nil {
				t.Fatal(err)
			}
			record, err := mgr.getTokenRecord(ctx, token)
			if err != nil {
				t.Fatal(err)
			}
			key := mgr.getTokenExtraKey(record.AccessID)
			if err = mgr.storage.Expire(ctx, key, 10*time.Second); err != nil {
				t.Fatal(err)
			}
			codec := &managerValidationCountingCodec{Codec: mgr.serializer}
			mgr.serializer = codec
			switch action {
			case "manual":
				err = mgr.RenewTimeout(ctx, token, time.Minute)
			case "unlimited":
				err = mgr.RenewTimeout(ctx, token, 0)
			case "shared":
				var shared string
				shared, err = mgr.Login(ctx, "extra-renewal", "web")
				if err == nil && shared != token {
					t.Fatal("shared login did not reuse token")
				}
			case "automatic":
				mgr.renewFunc(ctx, token, "extra-renewal", false)
			}
			if err != nil {
				t.Fatal(err)
			}
			if codec.tokens.Load() != 0 || codec.extras.Load() != 0 {
				t.Fatal("renewal decoded token or extension objects")
			}
			ttl, err := mgr.storage.TTL(ctx, key)
			if err != nil {
				t.Fatal(err)
			}
			if action == "unlimited" {
				if ttl != adapter.TTLNoExpire {
					t.Fatalf("extension TTL = %v, want unlimited", ttl)
				}
			} else if ttl < 50*time.Second || ttl > time.Minute {
				t.Fatalf("extension TTL = %v", ttl)
			}
			info, err := mgr.GetTokenInfo(ctx, token)
			if err != nil || !reflect.DeepEqual(info.Extra, extra) {
				t.Fatalf("renewed detail = %+v, %v", info, err)
			}
		})
	}
}

// TestCompactTokenCollisionCleansOnlyNewExtra checks ordinary and atomic publication collisions. TestCompactTokenCollisionCleansOnlyNewExtra 检查普通与原子发布冲突只清理本次扩展。
func TestCompactTokenCollisionCleansOnlyNewExtra(t *testing.T) {
	ctx := context.Background()
	for _, basic := range []bool{false, true} {
		t.Run(fmt.Sprint(basic), func(t *testing.T) {
			mgr := newTestManager(t, nil)
			if basic {
				mgr.storage = nonAtomicManagerStorage{Storage: mgr.storage}
			}
			token, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: "existing", Extra: map[string]any{"owner": "existing"}})
			if err != nil {
				t.Fatal(err)
			}
			candidate := tokenRecord{TokenInfo: TokenInfo{AuthType: mgr.config.AuthType, LoginID: "new", Extra: map[string]any{"owner": "new"}}, AccessID: "candidate-lifecycle"}
			if saved, err := mgr.createTokenRecord(ctx, token, &candidate, time.Minute); err != nil || saved {
				t.Fatalf("collision = %v, %v", saved, err)
			}
			if mgr.storage.Exists(ctx, mgr.getTokenExtraKey(candidate.AccessID)) {
				t.Fatal("collision leaked candidate extensions")
			}
			info, err := mgr.GetTokenInfo(ctx, token)
			if err != nil || info.LoginID != "existing" || info.Extra["owner"] != "existing" {
				t.Fatalf("collision changed existing credential: %+v, %v", info, err)
			}
		})
	}
}

// TestCompactTokenRejectsLegacyPayload checks runtime validation never decodes or renews legacy payloads. TestCompactTokenRejectsLegacyPayload 检查运行时校验不会解码或续期旧载荷。
func TestCompactTokenRejectsLegacyPayload(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	token, err := mgr.Login(ctx, "legacy", "web")
	if err != nil {
		t.Fatal(err)
	}
	info, err := mgr.GetTokenInfo(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	info.Extra = map[string]any{"legacy": "value"}
	if err = mgr.saveToStorage(ctx, mgr.getTokenKey(token), *info, time.Minute); err != nil {
		t.Fatal(err)
	}
	before, err := mgr.storage.Get(ctx, mgr.getTokenKey(token))
	if err != nil {
		t.Fatal(err)
	}
	codec := &managerValidationCountingCodec{Codec: mgr.serializer}
	mgr.serializer = codec
	if mgr.IsLogin(ctx, token) {
		t.Fatal("legacy payload was accepted")
	}
	if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("CheckLogin = %v, want ErrInvalidToken", err)
	}
	if err = mgr.RenewTimeout(ctx, token, time.Minute); !errors.Is(err, derror.ErrInvalidToken) {
		t.Fatalf("RenewTimeout = %v, want ErrInvalidToken", err)
	}
	after, err := mgr.storage.Get(ctx, mgr.getTokenKey(token))
	if err != nil || !reflect.DeepEqual(before, after) || codec.tokens.Load() != 0 || codec.sessions.Load() != 0 || codec.extras.Load() != 0 {
		t.Fatalf("legacy payload was decoded or rewritten: %v", err)
	}
}

// TestCompactTokenRequiresRuntimeIdentity checks invalid credentials never fall back to Codec decoding. TestCompactTokenRequiresRuntimeIdentity 检查无效凭据不会回退到 Codec 解码。
func TestCompactTokenRequiresRuntimeIdentity(t *testing.T) {
	ctx := context.Background()
	for _, scenario := range []string{"anonymous", "missing namespace", "foreign namespace", "malformed", "unknown version"} {
		t.Run(scenario, func(t *testing.T) {
			mgr := newTestManager(t, nil)
			record := tokenRecord{TokenInfo: TokenInfo{AuthType: mgr.config.AuthType, LoginID: "identity"}, AccessID: "lifecycle"}
			switch scenario {
			case "anonymous":
				record.AccessID = ""
			case "missing namespace":
				record.AuthType = ""
			case "foreign namespace":
				record.AuthType = "foreign:"
			}
			data := encodeTokenRecord(&record)
			if scenario == "malformed" {
				data = "{"
			} else if scenario == "unknown version" {
				data = strings.Replace(data, "dtk:2:", "dtk:3:", 1)
			}
			if err := mgr.storage.Set(ctx, mgr.getTokenKey("invalid"), data, time.Minute); err != nil {
				t.Fatal(err)
			}
			storage := &managerGetCountingStorage{Storage: mgr.storage}
			codec := &managerValidationCountingCodec{Codec: mgr.serializer}
			mgr.storage, mgr.serializer = storage, codec
			if mgr.IsLogin(ctx, "invalid") {
				t.Fatal("invalid credential accepted")
			}
			if err := mgr.CheckLogin(ctx, "invalid"); !errors.Is(err, derror.ErrInvalidToken) {
				t.Fatalf("CheckLogin = %v, want ErrInvalidToken", err)
			}
			if storage.getCount(mgr.getTokenKey("invalid")) != 2 || codec.tokens.Load() != 0 || codec.sessions.Load() != 0 || codec.extras.Load() != 0 {
				t.Fatal("invalid credential used fallback reads or Codec decoding")
			}
		})
	}
}

// TestCompactTokenWritesRejectMissingIdentity checks rejection precedes primary and extension writes. TestCompactTokenWritesRejectMissingIdentity 检查缺失身份在主记录及扩展写入前被拒绝。
func TestCompactTokenWritesRejectMissingIdentity(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t, nil)
	for _, extra := range []map[string]any{nil, {"payload": "value"}} {
		record := tokenRecord{TokenInfo: TokenInfo{AuthType: mgr.config.AuthType, LoginID: "anonymous", Extra: extra}}
		if saved, err := mgr.createTokenRecord(ctx, "anonymous", &record, time.Minute); saved || !errors.Is(err, derror.ErrInvalidToken) {
			t.Fatalf("create anonymous = %v, %v", saved, err)
		}
		if err := mgr.saveTokenRecord(ctx, "anonymous", &record, time.Minute); !errors.Is(err, derror.ErrInvalidToken) {
			t.Fatalf("save anonymous = %v", err)
		}
		if record.AccessID != "" || mgr.storage.Exists(ctx, mgr.getTokenKey("anonymous")) || mgr.storage.Exists(ctx, mgr.getTokenExtraKey("")) {
			t.Fatal("anonymous write assigned identity or published data")
		}
	}
}

// TestCompactTokenRefreshExtraLifecycle checks rotation and independent refresh after access expiration. TestCompactTokenRefreshExtraLifecycle 检查轮换及访问凭据过期后的独立刷新。
func TestCompactTokenRefreshExtraLifecycle(t *testing.T) {
	ctx := context.Background()
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprint(expired), func(t *testing.T) {
			mgr := newTestManager(t, nil)
			extra := map[string]any{"payload": "refresh snapshot"}
			pair, err := mgr.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{LoginOptions: LoginOptions{LoginID: "refresh-extra", Device: "web", Extra: extra}})
			if err != nil {
				t.Fatal(err)
			}
			old, err := mgr.getTokenRecord(ctx, pair.AccessToken)
			if err != nil {
				t.Fatal(err)
			}
			oldKey := mgr.getTokenExtraKey(old.AccessID)
			if expired {
				if err = mgr.storage.Delete(ctx, mgr.getTokenKey(pair.AccessToken), oldKey, mgr.getSessionKey(pair.LoginID)); err != nil {
					t.Fatal(err)
				}
			}
			rotated, err := mgr.RefreshToken(ctx, pair.RefreshToken)
			if err != nil {
				t.Fatal(err)
			}
			if mgr.storage.Exists(ctx, oldKey) {
				t.Fatal("rotation left old extra")
			}
			info, err := mgr.GetTokenInfo(ctx, rotated.AccessToken)
			if err != nil || !reflect.DeepEqual(info.Extra, extra) {
				t.Fatalf("rotated detail = %+v, %v", info, err)
			}
			current, err := mgr.getTokenRecord(ctx, rotated.AccessToken)
			if err != nil {
				t.Fatal(err)
			}
			if current.AccessID == old.AccessID {
				t.Fatal("rotation reused lifecycle identity")
			}
			if err = mgr.RevokeRefreshToken(ctx, rotated.RefreshToken); err != nil {
				t.Fatal(err)
			}
			if mgr.storage.Exists(ctx, mgr.getTokenExtraKey(current.AccessID)) {
				t.Fatal("refresh revocation left extra")
			}
		})
	}
}

// BenchmarkTokenRecordDecode compares credential parsing with full legacy object decoding. BenchmarkTokenRecordDecode 对比凭据解析与旧完整对象解码。
func BenchmarkTokenRecordDecode(b *testing.B) {
	for _, size := range []int{0, 1024, 65536} {
		record := tokenRecord{TokenInfo: TokenInfo{AuthType: "user", LoginID: "benchmark", Device: "web", Timeout: 3600}, AccessID: "lifecycle", TerminalIndex: 1}
		if size > 0 {
			record.Extra = map[string]any{"payload": strings.Repeat("x", size)}
			record.HasExtra = true
		}
		compact := encodeTokenRecord(&record)
		codec := managerTestCodec{}
		legacy, err := codec.Encode(record)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("compact/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ReportMetric(float64(len(compact)), "record-bytes")
			for b.Loop() {
				if _, err := decodeTokenRecord(compact); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("legacy/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ReportMetric(float64(len(legacy)), "record-bytes")
			for b.Loop() {
				var decoded tokenRecord
				if err := codec.Decode(legacy, &decoded); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
