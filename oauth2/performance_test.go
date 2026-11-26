package oauth2

import (
	"context"
	"errors"
	"reflect"
	"testing"

	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	listener "github.com/Zany2/dtoken-go/event"
)

// TestServiceRevokeReusesTokenAndBatchesDelete verifies one read and one delete call supply both revocation and its event. TestServiceRevokeReusesTokenAndBatchesDelete 验证一次读取和一次删除调用完成撤销及其事件。
func TestServiceRevokeReusesTokenAndBatchesDelete(t *testing.T) {
	for _, fail := range []bool{false, true} {
		ctx := context.Background()
		server := newOAuth2TestServer(t)
		client := oauth2TestClient()
		if err := server.RegisterClient(client); err != nil {
			t.Fatal(err)
		}
		token, err := server.ClientCredentialsToken(ctx, client.ClientID, client.ClientSecret, []string{"read"})
		if err != nil {
			t.Fatal(err)
		}
		storage := &revokeCountingStorage{Storage: server.storage, fail: fail}
		server.storage = storage
		calls := 0
		service := NewService(server, revokeEmitter(func(event listener.Event, loginID, credential string, extra map[string]any) {
			calls++
			if event != listener.EventOAuth2TokenRevoke || loginID != token.UserID || credential != token.Token || extra[listener.ExtraKeyClientID] != client.ClientID || !reflect.DeepEqual(extra[listener.ExtraKeyScopes], token.Scopes) {
				t.Errorf("unexpected revoke event: %s, %s, %v", event, loginID, extra)
			}
		}))
		err = service.RevokeOAuth2Token(ctx, token.Token)
		if storage.reads != 1 || storage.deletes != 1 || !reflect.DeepEqual(storage.keys, []string{server.getRefreshKey(token.RefreshToken), server.getTokenKey(token.Token)}) {
			t.Fatalf("reads/deletes/keys = %d/%d/%v", storage.reads, storage.deletes, storage.keys)
		}
		if fail {
			if !errors.Is(err, derror.ErrStorageUnavailable) || calls != 0 {
				t.Fatalf("failed revoke = %v, events = %d", err, calls)
			}
			continue
		}
		if err != nil || calls != 1 {
			t.Fatalf("revoke = %v, events = %d", err, calls)
		}
		if storage.Exists(ctx, server.getTokenKey(token.Token)) || storage.Exists(ctx, server.getRefreshKey(token.RefreshToken)) {
			t.Fatal("token pair survived revocation")
		}
		if err = service.RevokeOAuth2Token(ctx, token.Token); !errors.Is(err, derror.ErrInvalidAccessToken) || calls != 1 {
			t.Fatalf("repeated revoke = %v, events = %d", err, calls)
		}
	}
}

type revokeCountingStorage struct {
	adapter.Storage
	reads, deletes int
	keys           []string
	fail           bool
}

// Get counts token reads. Get 统计令牌读取。
func (s *revokeCountingStorage) Get(ctx context.Context, key string) (any, error) {
	s.reads++
	return s.Storage.Get(ctx, key)
}

// Delete records each batch and injects storage failure. Delete 记录每批删除并注入存储故障。
func (s *revokeCountingStorage) Delete(ctx context.Context, keys ...string) error {
	s.deletes++
	s.keys = append([]string(nil), keys...)
	if s.fail {
		return errors.New("delete unavailable")
	}
	return s.Storage.Delete(ctx, keys...)
}

type revokeEmitter func(listener.Event, string, string, map[string]any)

// Emit records revocation metadata. Emit 记录撤销元数据。
func (f revokeEmitter) Emit(event listener.Event, loginID, device, deviceID, token string, extra map[string]any) {
	f(event, loginID, token, extra)
}
