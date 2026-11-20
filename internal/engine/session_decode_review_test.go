package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	base64codec "github.com/Zany2/dtoken-go/codec/base64"
	jsoncodec "github.com/Zany2/dtoken-go/codec/json"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
)

// TestManagerSessionDecodeDistinguishesNullFromEmpty preserves legacy objects while rejecting encoded null across bundled codecs. TestManagerSessionDecodeDistinguishesNullFromEmpty 验证内置编解码器拒绝编码空值，同时保留旧版空对象。
func TestManagerSessionDecodeDistinguishesNullFromEmpty(t *testing.T) {
	ctx := context.Background()
	for _, codec := range []adapter.Codec{jsoncodec.NewJSONSerializer(), base64codec.NewBase64Serializer()} {
		for _, null := range []bool{false, true} {
			name := codec.Name() + "/legacy object"
			if null {
				name = codec.Name() + "/null"
			}
			t.Run(name, func(t *testing.T) {
				mgr := newTestManager(t, nil)
				mgr.serializer = codec
				id := "decode-session"
				token, err := mgr.Login(ctx, id, "web")
				if err != nil {
					t.Fatal(err)
				}
				var stored any = Session{Data: map[string]any{"legacy": "kept"}}
				if null {
					stored = (*Session)(nil)
				}
				key := mgr.getSessionKey(id)
				if err = mgr.saveToStorage(ctx, key, stored, time.Hour); err != nil {
					t.Fatal(err)
				}
				before := captureTerminalLifecycle(t, mgr, ctx, []string{key, mgr.getTokenKey(token)})
				sess, err := mgr.GetSession(ctx, id)
				if null {
					if sess != nil || !errors.Is(err, derror.ErrInvalidToken) {
						t.Fatalf("null session = %+v, %v", sess, err)
					}
					if err = mgr.CheckLogin(ctx, token); !errors.Is(err, derror.ErrInvalidToken) {
						t.Fatalf("null session authentication = %v", err)
					}
					if err = mgr.RenewTimeout(ctx, token, time.Minute); !errors.Is(err, derror.ErrInvalidToken) {
						t.Fatalf("null session renewal = %v", err)
					}
				} else {
					if err != nil || sess == nil || sess.LoginID != id || sess.AuthType != mgr.config.AuthType || len(sess.TerminalInfos) != 0 || sess.Data["legacy"] != "kept" {
						t.Fatalf("legacy session = %+v, %v", sess, err)
					}
					if _, _, err = mgr.checkLoginAndGetContextNoRenew(ctx, token); err != nil {
						t.Fatalf("legacy empty-terminal session rejected: %v", err)
					}
				}
				assertTerminalLifecycleUnchanged(t, mgr, ctx, before)
			})
		}
	}
}
