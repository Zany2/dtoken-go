package sso

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

// TestCredentialDeadlinesSurviveStorage verifies issuance and renewal preserve precise deadlines. TestCredentialDeadlinesSurviveStorage 验证签发和续期的精确截止时间可以正确持久化。
func TestCredentialDeadlinesSurviveStorage(t *testing.T) {
	ctx := context.Background()
	s := newTestServer()
	defer s.Close()
	client := newTestClient()
	client.Modes = []Mode{ModeTicket, ModeSharedToken, ModeRemoteSession, ModeOAuth2}
	if err := s.RegisterClient(client); err != nil {
		t.Fatal(err)
	}
	lifetime := 1500 * time.Millisecond
	check := func(start time.Time, deadline, stored time.Time, seconds int64) {
		t.Helper()
		if deadline.Before(start.Add(lifetime)) || deadline.After(time.Now().Add(lifetime)) || seconds != 2 {
			t.Fatalf("incorrect deadline or lifetime: %v, %d", deadline, seconds)
		}
		if !deadline.Equal(stored) {
			t.Fatalf("stored deadline = %v, want %v", stored, deadline)
		}
	}

	start := time.Now()
	ticket, err := s.GenerateTicketWithTimeout(ctx, client.ClientID, "user", client.RedirectURIs[0], nil, nil, lifetime)
	if err != nil {
		t.Fatal(err)
	}
	storedTicket, err := s.getTicket(ctx, ticket.Ticket)
	if err != nil {
		t.Fatal(err)
	}
	check(start, ticket.ExpiresAt, storedTicket.ExpiresAt, ticket.ExpiresIn)

	start = time.Now()
	token, err := s.GenerateSharedTokenWithTimeout(ctx, client.ClientID, "user", nil, nil, lifetime)
	if err != nil {
		t.Fatal(err)
	}
	storedToken, err := s.getSharedToken(ctx, token.Token)
	if err != nil {
		t.Fatal(err)
	}
	check(start, token.ExpiresAt, storedToken.ExpiresAt, token.ExpiresIn)

	start = time.Now()
	session, err := s.CreateRemoteSessionWithTimeout(ctx, client.ClientID, "user", nil, nil, lifetime)
	if err != nil {
		t.Fatal(err)
	}
	storedSession, err := s.getRemoteSession(ctx, session.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	check(start, session.ExpiresAt, storedSession.ExpiresAt, session.ExpiresIn)
	start = time.Now()
	if err = s.RenewRemoteSession(ctx, session.SessionID, lifetime); err != nil {
		t.Fatal(err)
	}
	storedSession, err = s.ValidateRemoteSession(ctx, session.SessionID, client.ClientID)
	if err != nil {
		t.Fatal(err)
	}
	check(start, storedSession.ExpiresAt, storedSession.ExpiresAt, storedSession.ExpiresIn)

	start = time.Now()
	code, err := s.GenerateOAuth2CodeWithTimeout(ctx, client.ClientID, "user", client.RedirectURIs[0], nil, nil, lifetime)
	if err != nil {
		t.Fatal(err)
	}
	storedCode, err := s.getOAuth2Code(ctx, code.Code)
	if err != nil {
		t.Fatal(err)
	}
	check(start, code.ExpiresAt, storedCode.ExpiresAt, code.ExpiresIn)
}

// TestCredentialExpiryCompatibility verifies exact deadlines take precedence and legacy records still expire. TestCredentialExpiryCompatibility 验证精确截止时间优先且旧记录仍按时过期。
func TestCredentialExpiryCompatibility(t *testing.T) {
	s := newTestServer()
	defer s.Close()
	now := time.Now()
	checks := []struct {
		name  string
		err   error
		check func(int64, int64, time.Time) error
	}{
		{"ticket", ErrTicketExpired, func(created, seconds int64, deadline time.Time) error {
			return s.checkTicketAlive(&Ticket{Ticket: "ticket", CreateTime: created, ExpiresIn: seconds, ExpiresAt: deadline})
		}},
		{"shared token", ErrSharedTokenExpired, func(created, seconds int64, deadline time.Time) error {
			return s.checkSharedTokenAlive(&SharedToken{Token: "token", CreateTime: created, ExpiresIn: seconds, ExpiresAt: deadline})
		}},
		{"remote session", ErrRemoteSessionExpired, func(created, seconds int64, deadline time.Time) error {
			return s.checkRemoteSessionAlive(&RemoteSession{SessionID: "session", CreateTime: created, ExpiresIn: seconds, ExpiresAt: deadline})
		}},
		{"oauth2 code", ErrOAuth2CodeExpired, func(created, seconds int64, deadline time.Time) error {
			return s.checkOAuth2CodeAlive(&OAuth2Code{Code: "code", CreateTime: created, ExpiresIn: seconds, ExpiresAt: deadline})
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			for _, tc := range []struct {
				name             string
				created, seconds int64
				deadline         time.Time
				expired          bool
			}{
				{"precise alive", now.Unix() - 60, 1, now.Add(time.Minute), false},
				{"precise expired", now.Unix(), 60, now.Add(-time.Second), true},
				{"legacy alive", now.Unix(), 60, time.Time{}, false},
				{"legacy expired", now.Unix() - 60, 1, time.Time{}, true},
				{"legacy overflow", math.MaxInt64, 1, time.Time{}, true},
				{"zero lifetime", now.Unix(), 0, time.Time{}, true},
				{"negative lifetime", now.Unix(), -1, time.Time{}, true},
			} {
				err := check.check(tc.created, tc.seconds, tc.deadline)
				if tc.expired && !errors.Is(err, check.err) || !tc.expired && err != nil {
					t.Fatalf("%s: error = %v, expired = %v", tc.name, err, tc.expired)
				}
			}
		})
	}
}

// TestExpiredCredentialsAreNotPersisted verifies expired payloads never become non-expiring storage records. TestExpiredCredentialsAreNotPersisted 验证已过期载荷不会变成永久存储记录。
func TestExpiredCredentialsAreNotPersisted(t *testing.T) {
	ctx := context.Background()
	s := newTestServer()
	defer s.Close()
	expired := time.Now().Add(-time.Second)
	for _, tc := range []struct {
		key  string
		want error
		save func() error
	}{
		{s.getTicketKey("expired"), ErrTicketExpired, func() error {
			return s.saveTicket(ctx, &Ticket{Ticket: "expired", ExpiresIn: 1, ExpiresAt: expired})
		}},
		{s.getSharedTokenKey("expired"), ErrSharedTokenExpired, func() error {
			return s.saveSharedToken(ctx, &SharedToken{Token: "expired", ExpiresIn: 1, ExpiresAt: expired})
		}},
		{s.getRemoteSessionKey("expired"), ErrRemoteSessionExpired, func() error {
			return s.saveRemoteSession(ctx, &RemoteSession{SessionID: "expired", ExpiresIn: 1, ExpiresAt: expired})
		}},
		{s.getOAuth2CodeKey("expired"), ErrOAuth2CodeExpired, func() error {
			return s.saveOAuth2Code(ctx, &OAuth2Code{Code: "expired", ExpiresIn: 1, ExpiresAt: expired})
		}},
	} {
		if err := tc.save(); !errors.Is(err, tc.want) {
			t.Fatalf("save %s = %v, want %v", tc.key, err, tc.want)
		}
		if s.storage.Exists(ctx, tc.key) {
			t.Fatalf("expired credential was persisted: %s", tc.key)
		}
	}
}

// TestReusableCredentialsFollowClientLifecycle verifies removed clients and disabled modes cannot validate or renew credentials. TestReusableCredentialsFollowClientLifecycle 验证客户端注销或模式禁用后不能校验或续期凭证。
func TestReusableCredentialsFollowClientLifecycle(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled mode", true: "unregistered"}[remove], func(t *testing.T) {
			ctx := context.Background()
			s := newTestServer()
			defer s.Close()
			client := newTestClient()
			client.Modes = []Mode{ModeSharedToken, ModeRemoteSession}
			if err := s.RegisterClient(client); err != nil {
				t.Fatal(err)
			}
			token, err := s.GenerateSharedToken(ctx, client.ClientID, "user", nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			session, err := s.CreateRemoteSession(ctx, client.ClientID, "user", nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.ValidateSharedToken(ctx, token.Token, client.ClientID); err != nil {
				t.Fatal(err)
			}
			if _, err = s.ValidateRemoteSession(ctx, session.SessionID, client.ClientID); err != nil {
				t.Fatal(err)
			}

			want := ErrModeUnsupported
			if remove {
				err = s.UnregisterClient(client.ClientID)
				want = ErrClientNotFound
			} else {
				client.Modes = []Mode{ModeTicket}
				err = s.RegisterClient(client)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.ValidateSharedToken(ctx, token.Token, client.ClientID); !errors.Is(err, want) {
				t.Fatalf("ValidateSharedToken = %v, want %v", err, want)
			}
			if _, err = s.ValidateRemoteSession(ctx, session.SessionID, client.ClientID); !errors.Is(err, want) {
				t.Fatalf("ValidateRemoteSession = %v, want %v", err, want)
			}
			if err = s.RenewRemoteSession(ctx, session.SessionID, time.Hour); !errors.Is(err, want) {
				t.Fatalf("RenewRemoteSession = %v, want %v", err, want)
			}
			stored, err := s.getRemoteSession(ctx, session.SessionID)
			if err != nil || !stored.ExpiresAt.Equal(session.ExpiresAt) {
				t.Fatalf("rejected renewal changed session: %+v, %v", stored, err)
			}
			if err = s.RevokeSharedToken(ctx, token.Token); err != nil {
				t.Fatal(err)
			}
			if err = s.RevokeRemoteSession(ctx, session.SessionID); err != nil {
				t.Fatal(err)
			}
		})
	}
}
