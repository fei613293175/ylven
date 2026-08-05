package identity

import (
	"testing"
	"time"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := NormalizeEmail("  User@Example.COM ")
	if err != nil || got != "user@example.com" { t.Fatalf("normalize = %q, %v", got, err) }
	if _, err := NormalizeEmail("bad"); err == nil { t.Fatal("invalid email accepted") }
}

func TestRegistrationFlowAndOneTimeOTP(t *testing.T) {
	s, err := NewStore(""); if err != nil { t.Fatal(err) }
	c, err := s.CreateChallenge("user@example.com", "register"); if err != nil { t.Fatal(err) }
	if err := s.VerifyTurnstile(c.ID, "test-pass", true); err != nil { t.Fatal(err) }
	o, err := s.CreateOTP(c.ID, "123456"); if err != nil { t.Fatal(err) }
	if o.ID == "" { t.Fatal("missing OTP id") }
	if err := s.VerifyOTP(c.ID, "000000"); err == nil { t.Fatal("invalid OTP accepted") }
	if err := s.VerifyOTP(c.ID, "123456"); err != nil { t.Fatal(err) }
	u, err := s.CreateUser(c.ID, "USER@example.com", "correct horse battery"); if err != nil { t.Fatal(err) }
	if u.Email != "user@example.com" || u.PasswordHash == "correct horse battery" { t.Fatalf("bad user: %+v", u) }
	if _, err := s.CreateUser(c.ID, u.Email, "another password"); err == nil { t.Fatal("challenge reused") }
}

func TestSessionRefreshRotatesRefreshToken(t *testing.T) {
	s, _ := NewStore("")
	c, _ := s.CreateChallenge("user@example.com", "register")
	_ = s.VerifyTurnstile(c.ID, "test-pass", true)
	_, _ = s.CreateOTP(c.ID, "123456")
	_ = s.VerifyOTP(c.ID, "123456")
	_, _ = s.CreateUser(c.ID, "user@example.com", "correct horse battery")
	l, _ := s.CreateChallenge("user@example.com", "login")
	_, _ = s.CreateOTP(l.ID, "654321")
	_ = s.VerifyOTP(l.ID, "654321")
	first, access, refresh, err := s.CreateSession(l.ID, "user@example.com")
	if err != nil || first.ID == "" || access == "" || refresh == "" { t.Fatalf("session issue failed: %+v %v", first, err) }
	second, _, nextRefresh, err := s.RotateSession(refresh)
	if err != nil || second.ID == first.ID || nextRefresh == refresh { t.Fatalf("rotation failed: %+v %v", second, err) }
	if _, _, _, err := s.RotateSession(refresh); err == nil { t.Fatal("refresh token was reusable") }
}

func TestSessionLogoutAndRateLimit(t *testing.T) {
	s, _ := NewStore("")
	ok, err := s.AllowAttempt("user@example.com", 1, time.Hour)
	if err != nil || !ok { t.Fatalf("first attempt blocked: %v", err) }
	ok, err = s.AllowAttempt("user@example.com", 1, time.Hour)
	if err != nil || ok { t.Fatalf("second attempt was not limited: %v", err) }
}

func TestAdminSettingsRejectRawSecrets(t *testing.T) {
	s, _ := NewStore("")
	if err := s.PutSetting("turnstile", map[string]string{"secret":"plain-text"}); err == nil { t.Fatal("raw secret accepted") }
	if err := s.PutSetting("turnstile", map[string]string{"site_key_reference":"cf-site","secret_reference":"vault:turnstile"}); err != nil { t.Fatal(err) }
	value, ok := s.GetSetting("turnstile")
	if !ok || value["secret_reference"] != "vault:turnstile" { t.Fatalf("unexpected setting: %#v", value) }
}
