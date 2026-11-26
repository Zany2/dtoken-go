package dtoken

import (
	"context"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	"testing"
)

// TestDefaultMemoryTerminalScopes checks every default retirement route against sibling terminals and accounts. TestDefaultMemoryTerminalScopes 验证各默认下线入口对其他终端和账号的隔离。
func TestDefaultMemoryTerminalScopes(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		retire  func(string) error
		removed int
	}{
		{"Logout/token", func(token string) error { return LogoutByToken(ctx, token) }, 1},
		{"Logout/terminal", func(token string) error { return LogoutByDeviceAndDeviceID(ctx, "user", "web", "one") }, 1},
		{"Logout/device", func(token string) error { return LogoutByDevice(ctx, "user", "web") }, 2},
		{"Logout/account", func(token string) error { return LogoutByLoginID(ctx, "user") }, 3},
		{"Logout/options", func(token string) error {
			return Logout(ctx, LogoutOptions{LoginID: "user", Device: "web", DeviceID: "one"})
		}, 1},
		{"Kickout/token", func(token string) error { return KickoutByToken(ctx, token) }, 1},
		{"Kickout/terminal", func(token string) error { return KickoutByDeviceAndDeviceID(ctx, "user", "web", "one") }, 1},
		{"Kickout/device", func(token string) error { return KickoutByDevice(ctx, "user", "web") }, 2},
		{"Kickout/account", func(token string) error { return KickoutByLoginID(ctx, "user") }, 3},
		{"Kickout/options", func(token string) error {
			return Kickout(ctx, LogoutOptions{LoginID: "user", Device: "web", DeviceID: "one"})
		}, 1},
		{"Replace/token", func(token string) error { return ReplaceByToken(ctx, token) }, 1},
		{"Replace/terminal", func(token string) error { return ReplaceByDeviceAndDeviceID(ctx, "user", "web", "one") }, 1},
		{"Replace/device", func(token string) error { return ReplaceByDevice(ctx, "user", "web") }, 2},
		{"Replace/account", func(token string) error { return ReplaceByLoginID(ctx, "user") }, 3},
		{"Replace/options", func(token string) error {
			return Replace(ctx, LogoutOptions{LoginID: "user", Device: "web", DeviceID: "one"})
		}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			DeleteAllManager()
			t.Cleanup(CloseAll)
			_, err := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).IsShare(false).Build()
			if err != nil {
				t.Fatal(err)
			}
			tokens := []string{}
			for _, opts := range []LoginOptions{{LoginID: "user", Device: "web", DeviceID: "one"}, {LoginID: "user", Device: "web", DeviceID: "two"}, {LoginID: "user", Device: "phone", DeviceID: "three"}, {LoginID: "other"}} {
				token, err := Login(ctx, opts)
				if err != nil {
					t.Fatal(err)
				}
				tokens = append(tokens, token)
			}
			if err = tc.retire(tokens[0]); err != nil {
				t.Fatal(err)
			}
			for i, token := range tokens {
				want := i >= tc.removed
				if IsLogin(ctx, token) != want || (CheckLogin(ctx, token) == nil) != want {
					t.Fatalf("token %d validity != %v", i, want)
				}
			}
			if count, err := GetOnlineTerminalCount(ctx, "user"); err != nil || count != 3-tc.removed {
				t.Fatalf("remaining count=%d,%v", count, err)
			}
		})
	}
}

// TestDefaultMemorySessionQueries checks account, device and concrete-terminal selection. TestDefaultMemorySessionQueries 验证账号、设备与具体终端查询的筛选结果。
func TestDefaultMemorySessionQueries(t *testing.T) {
	DeleteAllManager()
	t.Cleanup(CloseAll)
	_, err := NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).IsShare(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, opts := range []LoginOptions{
		{LoginID: "user", Device: "web", DeviceID: "one", Token: "query-one"},
		{LoginID: "user", Device: "phone", DeviceID: "two", Token: "query-two"},
		{LoginID: "other", Device: "web", Token: "other-token"},
	} {
		if _, err = Login(ctx, opts); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := GetOnlineTerminalCountByDevice(ctx, "user", "web"); err != nil || n != 1 {
		t.Fatalf("device count=%d,%v", n, err)
	}
	if n, err := GetOnlineTerminalCountByDeviceAndDeviceID(ctx, "user", "web", "one"); err != nil || n != 1 {
		t.Fatalf("terminal count=%d,%v", n, err)
	}
	for _, get := range []func() (*manager.Session, error){
		func() (*manager.Session, error) { return GetSession(ctx, "user") },
		func() (*manager.Session, error) { return GetSessionByToken(ctx, "query-one") },
	} {
		if session, err := get(); err != nil || session == nil || session.LoginID != "user" {
			t.Fatalf("session=%+v,%v", session, err)
		}
	}
	for _, tc := range []struct {
		call func() ([]string, error)
		want int
	}{
		{func() ([]string, error) { return GetTokenValueListByLoginID(ctx, "user", true) }, 2},
		{func() ([]string, error) { return GetTokenValueListByDevice(ctx, "user", "web", true) }, 1},
		{func() ([]string, error) { return GetTokenValueListByDeviceAndDeviceID(ctx, "user", "web", "one", true) }, 1},
		{func() ([]string, error) { return SearchTokenValue(ctx, "query-", 0, -1) }, 2},
		{func() ([]string, error) { return SearchSessionId(ctx, "user", 0, -1) }, 1},
	} {
		values, err := tc.call()
		if err != nil || len(values) != tc.want {
			t.Fatalf("selection=%v,%v; want %d", values, err, tc.want)
		}
		for _, value := range values {
			if value != "query-one" && value != "query-two" && value != "user" {
				t.Fatalf("unrelated selection=%q", value)
			}
		}
	}
	if values, err := GetTerminalListByLoginID(ctx, "user"); err != nil || len(values) != 2 {
		t.Fatalf("terminals=%v,%v", values, err)
	}
	if values, err := GetTerminalListByLoginIDAndDevice(ctx, "user", "web"); err != nil || len(values) != 1 || values[0].DeviceID != "one" {
		t.Fatalf("web terminals=%v,%v", values, err)
	}
	if terminal, err := GetTerminalInfoByToken(ctx, "query-one"); err != nil || terminal == nil || terminal.DeviceID != "one" {
		t.Fatalf("terminal=%v,%v", terminal, err)
	}
	if token, err := GetTokenValueByLoginID(ctx, "user"); err != nil || (token != "query-one" && token != "query-two") {
		t.Fatalf("latest token=%q,%v", token, err)
	}
	if token, err := GetTokenValueByLoginIDAndDevice(ctx, "user", "web"); err != nil || token != "query-one" {
		t.Fatalf("web token=%q,%v", token, err)
	}
	visits := 0
	if err = ForEachTerminal(ctx, "user", func(terminal manager.TerminalInfo) bool { visits++; return false }); err != nil || visits != 1 {
		t.Fatalf("early stop visits=%d,%v", visits, err)
	}
	visits = 0
	if err = ForEachTerminalByDevice(ctx, "user", "web", func(terminal manager.TerminalInfo) bool {
		visits++
		if terminal.DeviceID != "one" {
			t.Errorf("wrong terminal=%+v", terminal)
		}
		return true
	}); err != nil || visits != 1 {
		t.Fatalf("web visits=%d,%v", visits, err)
	}
}
