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

// TestManagerSessionDataCosts checks full public operations without reading Extra or scheduling maintenance. TestManagerSessionDataCosts 检查完整公开操作的成本，且不读取 Extra 或调度维护。
func TestManagerSessionDataCosts(t *testing.T) {
	ctx := context.Background()
	for _, byToken := range []bool{false, true} {
		for _, active := range []bool{false, true} {
			for _, unlimited := range []bool{false, true} {
				for _, operation := range []string{"set", "set nil", "get nil", "get missing", "delete", "delete missing"} {
					t.Run(fmt.Sprintf("%s/token=%v/active=%v/unlimited=%v", operation, byToken, active, unlimited), func(t *testing.T) {
						mgr := newTestManager(t, func(cfg *config.Config) {
							cfg.Timeout, cfg.AutoRenew = 600, true
							cfg.RenewMaxRefresh = 300
							if active {
								cfg.ActiveTimeout = 60
							}
						})
						const id = "session-data-cost"
						token, err := mgr.LoginWithOptions(ctx, LoginOptions{LoginID: id, Device: "web", Extra: map[string]any{"payload": "detail"}})
						if err != nil {
							t.Fatal(err)
						}
						sess, err := mgr.getSession(ctx, id)
						if err != nil {
							t.Fatal(err)
						}
						sess.Data = map[string]any{"key": nil, "keep": strings.Repeat("x", 32*1024)}
						sess.Permissions, sess.Roles = []string{"read"}, []string{"user"}
						ttl := time.Hour
						if unlimited {
							ttl = 0
						}
						sessionKey := mgr.getSessionKey(id)
						if err := mgr.saveToStorage(ctx, sessionKey, *sess, ttl); err != nil {
							t.Fatal(err)
						}
						beforeTTL, err := mgr.storage.TTL(ctx, sessionKey)
						if err != nil {
							t.Fatal(err)
						}
						storage := &retirementCostStorage{Storage: mgr.storage, gets: make(map[string]int), sessionKey: sessionKey}
						decoding := &retirementCostCodec{Codec: mgr.serializer}
						encoding := &loginSessionWriteCodec{Codec: decoding}
						mgr.storage, mgr.serializer = storage, encoding
						pool := &managerQueuedMaintenancePool{}
						mgr.pool = pool
						t.Cleanup(pool.runAll)
						set, get, remove := mgr.SetSessionValue, mgr.GetSessionValue, mgr.DeleteSessionValue
						subject := id
						if byToken {
							set, get, remove = mgr.SetSessionValueByToken, mgr.GetSessionValueByToken, mgr.DeleteSessionValueByToken
							subject = token
						}
						writes, tokenReads := 0, 2
						switch operation {
						case "set", "set nil":
							var value any
							if operation == "set" {
								value = "changed"
							}
							err = set(ctx, subject, " key ", value)
							sess.Data["key"], writes = value, 1
						case "get nil", "get missing":
							key := " key "
							if operation == "get missing" {
								key = "missing"
							}
							value, found, getErr := get(ctx, subject, key)
							if getErr != nil || value != nil || found != (operation == "get nil") {
								t.Fatalf("read = %v, %v, %v", value, found, getErr)
							}
							tokenReads = 1
						case "delete":
							err = remove(ctx, subject, " key ")
							delete(sess.Data, "key")
							writes = 1
						case "delete missing":
							err = remove(ctx, subject, "missing")
						}
						if err != nil {
							t.Fatal(err)
						}
						wantGets := map[string]int{sessionKey: 1}
						if byToken {
							wantGets[mgr.getTokenKey(token)] = tokenReads
							if active {
								wantGets[mgr.getActiveKey(token)] = 1
							}
						}
						if !reflect.DeepEqual(storage.gets, wantGets) || decoding.decodes != 1 {
							t.Fatalf("GETs/decodes = %v/%d, want %v/1", storage.gets, decoding.decodes, wantGets)
						}
						if storage.sessionWrites != writes || encoding.sessions != writes || len(storage.deletes) != 0 || pool.taskCount() != 0 {
							t.Fatalf("writes/encodes/deletes/tasks = %d/%d/%d/%d, want %d/%d/0/0", storage.sessionWrites, encoding.sessions, len(storage.deletes), pool.taskCount(), writes, writes)
						}
						afterTTL, err := storage.TTL(ctx, sessionKey)
						if err != nil || unlimited && afterTTL != adapter.TTLNoExpire || !unlimited && (afterTTL <= 0 || afterTTL > beforeTTL || beforeTTL-afterTTL > 5*time.Second) {
							t.Fatalf("TTL changed unexpectedly: before=%v, after=%v, error=%v", beforeTTL, afterTTL, err)
						}
						stored, err := mgr.getSession(ctx, id)
						if err != nil || !reflect.DeepEqual(stored, sess) {
							t.Fatalf("session changed unexpectedly: got=%+v, error=%v", stored, err)
						}
					})
				}
			}
		}
	}
}

// TestManagerSessionDataWritesValidateCurrentSession checks lock-entry session changes and failed persistence. TestManagerSessionDataWritesValidateCurrentSession 检查入锁前 Session 变化及持久化失败。
func TestManagerSessionDataWritesValidateCurrentSession(t *testing.T) {
	ctx := context.Background()
	for _, operation := range []string{"set", "delete"} {
		for _, state := range []string{"missing", "corrupt", "foreign", "write failure", "TTL failure"} {
			t.Run(operation+"/"+state, func(t *testing.T) {
				mgr := newTestManager(t, nil)
				const id = "session-data-failure"
				token, err := mgr.Login(ctx, id, "web")
				if err != nil {
					t.Fatal(err)
				}
				if err = mgr.SetSessionValue(ctx, id, "key", "original"); err != nil {
					t.Fatal(err)
				}
				base, sessionKey := mgr.storage, mgr.getSessionKey(id)
				original, err := base.Get(ctx, sessionKey)
				if err != nil {
					t.Fatal(err)
				}
				storage := &retirementCostStorage{Storage: base, gets: make(map[string]int), sessionKey: sessionKey}
				wantErr, wantValue := derror.ErrInvalidToken, original
				switch state {
				case "missing":
					wantValue = nil
				case "corrupt":
					wantValue, wantErr = []byte("{"), derror.ErrSerializeFailed
				case "foreign":
					wantValue, err = mgr.serializer.Encode(Session{AuthType: "foreign", LoginID: id})
					if err != nil {
						t.Fatal(err)
					}
				case "write failure":
					storage.failSetKey, wantErr = sessionKey, derror.ErrStorageUnavailable
				case "TTL failure":
					storage.failTTL, wantErr = true, derror.ErrStorageUnavailable
				}
				mgr.storage = &managerAccountSwapStorage{
					Storage: storage, key: mgr.getTokenKey(token),
					afterRead: func() error {
						if wantValue == nil {
							return base.Delete(ctx, sessionKey)
						}
						if err := base.Set(ctx, sessionKey, wantValue, time.Hour); err != nil {
							return err
						}
						// Compare the backend's representation, which may be bytes or a string. 比较存储后端实际返回的表示，兼容字节与字符串。
						var readErr error
						wantValue, readErr = base.Get(ctx, sessionKey)
						return readErr
					},
				}
				if operation == "set" {
					err = mgr.SetSessionValueByToken(ctx, token, "key", "changed")
				} else {
					err = mgr.DeleteSessionValueByToken(ctx, token, "key")
				}
				if !errors.Is(err, wantErr) {
					t.Fatalf("error = %v, want %v", err, wantErr)
				}
				stored, err := base.Get(ctx, sessionKey)
				if err != nil || !reflect.DeepEqual(stored, wantValue) || storage.sessionWrites != 0 {
					t.Fatalf("failed operation changed the session: error=%v, writes=%d", err, storage.sessionWrites)
				}
			})
		}
	}
}
