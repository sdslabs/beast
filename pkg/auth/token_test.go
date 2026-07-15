package auth

import (
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

func initializeTokenTest() {
	Init(1, 32, 60, "test-issuer", "test-secret", []string{"author"}, []string{"admin"}, []string{"user"})
}

func TestAuthorizeClaimsValidatesToken(t *testing.T) {
	initializeTokenTest()
	token, err := GenerateJWT(AuthModel{Username: "alice", Role: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := AuthorizeClaims(token, ADMIN)
	if err != nil {
		t.Fatal(err)
	}
	if claims.User != "alice" {
		t.Fatalf("unexpected user %q", claims.User)
	}
}

func TestAuthorizeClaimsRejectsOtherHMACMethods(t *testing.T) {
	initializeTokenTest()
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, CustomClaims{
		User: "alice",
		Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			Issuer:    ISSUER,
		},
	})
	signed, err := token.SignedString([]byte(JWTSECRET))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AuthorizeClaims(signed, ADMIN); err == nil {
		t.Fatal("expected signing method rejection")
	}
}

func TestAuthorizeClaimsRejectsWrongIssuer(t *testing.T) {
	initializeTokenTest()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, CustomClaims{
		User: "alice",
		Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			Issuer:    "attacker",
		},
	})
	signed, err := token.SignedString([]byte(JWTSECRET))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AuthorizeClaims(signed, ADMIN); err == nil {
		t.Fatal("expected issuer rejection")
	}
}

func TestAuthorizeClaimsRejectsExpiredToken(t *testing.T) {
	initializeTokenTest()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, CustomClaims{
		User: "alice",
		Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			Issuer:    ISSUER,
		},
	})
	signed, err := token.SignedString([]byte(JWTSECRET))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AuthorizeClaims(signed, ADMIN); err == nil {
		t.Fatal("expected expiration rejection")
	}
}
