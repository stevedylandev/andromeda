package auth

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestSecureEqual_Equal(t *testing.T) {
	if !SecureEqual("hunter2", "hunter2") {
		t.Fatal("equal strings should match")
	}
}

func TestSecureEqual_Unequal(t *testing.T) {
	if SecureEqual("hunter2", "hunter3") {
		t.Fatal("different strings should not match")
	}
}

func TestSecureEqual_BothEmpty(t *testing.T) {
	if !SecureEqual("", "") {
		t.Fatal("two empty strings should match")
	}
}

func TestSecureEqual_EmptyVsNonempty(t *testing.T) {
	if SecureEqual("", "x") {
		t.Fatal("empty vs nonempty should not match")
	}
	if SecureEqual("x", "") {
		t.Fatal("nonempty vs empty should not match")
	}
}

func TestSecureEqual_LengthMismatch(t *testing.T) {
	if SecureEqual("short", "longer_password") {
		t.Fatal("length mismatch should not match")
	}
}

func TestSecureEqual_Over256SameLengthAndPrefix(t *testing.T) {
	a := strings.Repeat("a", 300)
	b := strings.Repeat("a", 256) + strings.Repeat("b", 44)
	if !SecureEqual(a, b) {
		t.Fatal("same length, identical first 256 bytes should match after pad/truncate")
	}
}

func TestSecureEqual_Over256DifferentPrefix(t *testing.T) {
	a := strings.Repeat("a", 300)
	b := "z" + strings.Repeat("a", 299)
	if SecureEqual(a, b) {
		t.Fatal("differing prefix within first 256 bytes should not match")
	}
}

func TestSecureEqual_Exactly256(t *testing.T) {
	pw := strings.Repeat("x", 256)
	if !SecureEqual(pw, pw) {
		t.Fatal("exact 256-byte identical strings should match")
	}
}

func TestVerifyPassword_PlainHappy(t *testing.T) {
	if !VerifyPassword("hunter2", "hunter2") {
		t.Fatal("plain password should verify")
	}
}

func TestVerifyPassword_PlainSad(t *testing.T) {
	if VerifyPassword("hunter2", "hunter3") {
		t.Fatal("wrong plain password should fail")
	}
}

func TestVerifyPassword_PlainLengthMismatch(t *testing.T) {
	if VerifyPassword("short", "longer_password") {
		t.Fatal("length mismatch should fail")
	}
}

func TestVerifyPassword_BcryptHappy(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("hunter2"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword("hunter2", string(hash)) {
		t.Fatal("bcrypt password should verify")
	}
}

func TestVerifyPassword_BcryptSad(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("hunter2"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if VerifyPassword("nope", string(hash)) {
		t.Fatal("wrong bcrypt password should fail")
	}
}

func TestGenerateSessionToken(t *testing.T) {
	tok, err := GenerateSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) != 64 {
		t.Fatalf("want 64 hex chars, got %d", len(tok))
	}
	for _, c := range tok {
		isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
		if !isHex {
			t.Fatalf("non-hex char %q in token", c)
		}
	}
	tok2, _ := GenerateSessionToken()
	if tok == tok2 {
		t.Fatal("two tokens should differ")
	}
}

func TestSessionCookie_Attrs(t *testing.T) {
	s := &Store{CookieName: "session", CookieSecure: false}
	c := s.SessionCookie("abc123")
	if c.Name != "session" || c.Value != "abc123" {
		t.Fatalf("bad name/value: %+v", c)
	}
	if !c.HttpOnly {
		t.Fatal("HttpOnly should be set")
	}
	if c.SameSite != http.SameSiteStrictMode {
		t.Fatalf("want SameSite=Strict, got %v", c.SameSite)
	}
	if c.Path != "/" {
		t.Fatalf("want Path=/, got %q", c.Path)
	}
	if c.MaxAge != 7*24*3600 {
		t.Fatalf("want MaxAge=604800, got %d", c.MaxAge)
	}
	if c.Secure {
		t.Fatal("Secure should be false")
	}
}

func TestSessionCookie_Secure(t *testing.T) {
	s := &Store{CookieName: "session", CookieSecure: true}
	if !s.SessionCookie("x").Secure {
		t.Fatal("Secure should be true when CookieSecure=true")
	}
}

func TestClearCookie(t *testing.T) {
	s := &Store{CookieName: "session"}
	c := s.ClearCookie()
	if c.Value != "" || c.MaxAge != -1 {
		t.Fatalf("clear cookie should have empty value and MaxAge=-1, got %+v", c)
	}
	if c.SameSite != http.SameSiteStrictMode {
		t.Fatalf("want SameSite=Strict, got %v", c.SameSite)
	}
}

func TestGenerateShortID(t *testing.T) {
	id, err := GenerateShortID(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 10 {
		t.Fatalf("want len 10, got %d", len(id))
	}
	if _, err := GenerateShortID(0); err == nil {
		t.Fatal("zero length should error")
	}
}
