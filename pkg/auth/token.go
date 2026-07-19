package auth

import (
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

type CustomClaims struct {
	User     string `json:"usr"`
	Role     string `json:"role"`
	TokenUse string `json:"token_use"`
	jwt.RegisteredClaims
}

const (
	AccessTokenUse        = "access"
	PasswordResetTokenUse = "password_reset"
)

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

func AuthorizeClaims(jwtTokenString string, roleAccess int) (*CustomClaims, error) {
	claims, err := parseClaims(jwtTokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenUse != AccessTokenUse {
		return nil, fmt.Errorf("token is not an access token")
	}

	if !((roleAccess&MANAGER) != 0 && contains(ManagerRoles, claims.Role) ||
		(roleAccess&ADMIN) != 0 && contains(AdminRoles, claims.Role) ||
		(roleAccess&USER) != 0 && contains(UserRoles, claims.Role)) {
		return nil, fmt.Errorf("role access error")
	}

	return claims, nil
}

func AuthorizePasswordResetClaims(jwtTokenString string) (*CustomClaims, error) {
	claims, err := parseClaims(jwtTokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenUse != PasswordResetTokenUse {
		return nil, fmt.Errorf("token is not a password reset token")
	}
	return claims, nil
}

func parseClaims(jwtTokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(jwtTokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("token signing method is invalid")
		}
		return []byte(JWTSECRET), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}
	if claims.Issuer != ISSUER {
		return nil, fmt.Errorf("token issuer is invalid")
	}

	return claims, nil
}

func Authorize(jwtTokenString string, roleAccess int) error {
	_, err := AuthorizeClaims(jwtTokenString, roleAccess)
	return err
}

func GenerateJWT(authEntry AuthModel) (string, error) {
	now := time.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, CustomClaims{
		User:     authEntry.Username,
		Role:     authEntry.Role,
		TokenUse: AccessTokenUse,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(TIME_PERIOD) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    ISSUER,
		},
	})

	return token.SignedString([]byte(JWTSECRET))
}

func GeneratePasswordResetJWT(username, role string, lifetime time.Duration) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, CustomClaims{
		User:     username,
		Role:     role,
		TokenUse: PasswordResetTokenUse,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(lifetime)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    ISSUER,
		},
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
