package identity

import "testing"

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
