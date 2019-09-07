package auth

import (
	"fmt"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

type CustomClaims struct {
	User      string `json:"usr"`
	Role      string `json:"eml"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	Issuer    string `json:"iss"`
}

func (c CustomClaims) Valid() error {
	if c.ExpiresAt < time.Now().Unix() {
		return fmt.Errorf("Token Expired")
	}
	return nil
}

func Authorize(jwtTokenString string) error {
	token, err := jwt.Parse(jwtTokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Token invalid")
		}
		return []byte(JWTSECRET), nil
	})

	if err != nil {
		return err
	}

	if !(token.Valid) {
		return fmt.Errorf("Token invalid")
	}

	return token.Claims.Valid()
}

func GenerateJWT(authEntry AuthModel) (string, error) {
	t := time.Now().Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, CustomClaims{
		User:      authEntry.Username,
		Role:      authEntry.Role,
		ExpiresAt: t + TIME_PERIOD,
		IssuedAt:  t,
		Issuer:    ISSUER,
	})

	return token.SignedString([]byte(JWTSECRET))
}
