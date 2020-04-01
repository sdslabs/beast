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

const (
	ADMIN   int = 1 << 0
	MANAGER int = 1 << 1
	USER    int = 1 << 2
)

var (
	ManagerRoles []string
	AdminRoles   []string
	UserRoles    []string
)

func (c CustomClaims) Valid() error {
	if c.ExpiresAt < time.Now().Unix() {
		return fmt.Errorf("Token Expired")
	}

	return nil
}

func Authorize(jwtTokenString string, roleAccess int) error {
	token, err := jwt.ParseWithClaims(jwtTokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Token invalid")
		}
		return []byte(JWTSECRET), nil
	})

	if err != nil {
		return err
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return fmt.Errorf("Token invalid")
	}

	if !((roleAccess&MANAGER) != 0 && contains(ManagerRoles, claims.Role) ||
		(roleAccess&ADMIN) != 0 && contains(AdminRoles, claims.Role) ||
		(roleAccess&USER) != 0 && contains(UserRoles, claims.Role)) {
		return fmt.Errorf("Role Access Error")
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

func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
