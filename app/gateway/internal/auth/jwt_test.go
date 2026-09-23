package auth

import (
	"testing"
	"time"

	"github.com/tmjwjx/supermarket/app/gateway/internal/conf"

	"github.com/golang-jwt/jwt/v5"
)

func TestVerifyReturnsSubject(t *testing.T) {
	v := NewVerifier(&conf.Auth{JWTSecret: "secret"})
	token := signHS256(t, "secret", jwt.RegisteredClaims{
		Subject:   "user-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	sub, err := v.Verify("Bearer " + token)
	if err != nil {
		t.Fatal(err)
	}
	if sub != "user-1" {
		t.Fatalf("sub = %q", sub)
	}
}

func TestVerifyRejects(t *testing.T) {
	v := NewVerifier(&conf.Auth{JWTSecret: "secret"})
	cases := []string{
		"",
		"Bearer",
		"Bearer " + signHS256(t, "secret", jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}),
		"Bearer " + signHS256(t, "secret", jwt.RegisteredClaims{Subject: "user-1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}),
		"Bearer " + signMethod(t, jwt.SigningMethodHS384, "secret", jwt.RegisteredClaims{Subject: "user-1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}),
	}
	for _, header := range cases {
		if _, err := v.Verify(header); err == nil {
			t.Fatalf("accepted %q", header)
		}
	}
}

func TestAdminVerifyRequiresAudience(t *testing.T) {
	v := NewAdminVerifier(&conf.Auth{AdminJWTSecret: "admin-secret"})
	ok := signAdmin(t, "admin-secret", "admin-1", "order", "admin")
	id, role, err := v.Verify("Bearer " + ok)
	if err != nil {
		t.Fatal(err)
	}
	if id != "admin-1" || role != "order" {
		t.Fatalf("id=%q role=%q", id, role)
	}
	bad := signAdmin(t, "admin-secret", "admin-1", "order", "shop")
	if _, _, err := v.Verify("Bearer " + bad); err == nil {
		t.Fatal("accepted non-admin audience")
	}
}

func signHS256(t *testing.T, secret string, claims jwt.RegisteredClaims) string {
	t.Helper()
	return signMethod(t, jwt.SigningMethodHS256, secret, claims)
}

func signMethod(t *testing.T, method jwt.SigningMethod, secret string, claims jwt.Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(method, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func signAdmin(t *testing.T, secret, sub, role, aud string) string {
	t.Helper()
	return signMethod(t, jwt.SigningMethodHS256, secret, jwt.MapClaims{
		"sub":  sub,
		"aud":  aud,
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
}
