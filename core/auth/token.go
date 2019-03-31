package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/database"
)

type CustomClaims struct {
	User      string `json:"usr"`
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
		return []byte(config.Cfg.JWTSecret), nil
	})

	if err != nil {
		return err
	}

	if !(token.Valid) {
		return fmt.Errorf("Token invalid")
	}

	return token.Claims.Valid()
}

func GenerateJWT(username, decrmess string) (string, error) {
	author, err := database.QueryFirstAuthorEntry("name", username)
	if err != nil {
		return "", err
	}

	if string(author.AuthChallenge) != decrmess {
		return "", fmt.Errorf("Error : The messages are not same")
	}

	t := time.Now().Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, CustomClaims{
		User:      username,
		ExpiresAt: t + 6*60*60,
		IssuedAt:  t,
		Issuer:    "beast-sds",
	})

	return token.SignedString([]byte(config.Cfg.JWTSecret))
}

func DecryptToken(authHeader string) string {
	values := strings.Split(authHeader, " ")
	userInfoEncr := strings.Split(values[1], ".")
	sDec, err1 := b64.StdEncoding.DecodeString(userInfoEncr[1])
	if err1 != nil {
		fmt.Printf("Error in decrypting JWT token")
	}
	in := []byte(sDec)
	var raw CustomClaims
	json.Unmarshal(in, &raw)
	return raw.User
}

func GenerateAuthChallenge(username string) (string, error) {
	author, err := database.QueryFirstAuthorEntry("name", username)

	if err != nil {
		return "", err
	}

	rMessage := make([]byte, 128)
	rand.Read(rMessage)

	database.UpdateAuthor(&author, map[string]interface{}{"AuthChallenge": rMessage})

	encMessage, err := EncryptMessage(rMessage, author.SshKey)
	if err != nil {
		return "", fmt.Errorf("Error while encrypting message : %s", err)
	}

	return string(encMessage), nil
}

func EncryptMessage(message []byte, sshKey string) ([]byte, error) {
	pubKey, err := ParseAuthorizedRSAKey(sshKey)
	if err != nil {
		return []byte{}, err
	}
	return rsa.EncryptPKCS1v15(rand.Reader, pubKey, message)
}

func ParseAuthorizedRSAKey(in string) (*rsa.PublicKey, error) {
	//string contains data in format ssh-rsa <some-string-in-format-given-ahead> <email>

	//Format of string:
	//base64encoded ->
	//First 4 bytes give its len then data for 3 data elements:
	//1) type of key
	//2) N
	//3) E
	//In rsa notation

	N := new(big.Int)
	n := new(big.Int)

	b64Encoded := (strings.Split(in, " "))[1]
	decodedInfo, err := base64.StdEncoding.DecodeString(b64Encoded)
	if err != nil {
		return nil, err
	}

	offset := 0

	datalen := 4
	n = n.SetBytes(decodedInfo[offset : offset+4])
	offset += datalen

	datalen = int(n.Int64())
	n = n.SetBytes(decodedInfo[offset : offset+datalen]) //ssh-rsa
	offset += datalen

	datalen = 4
	n = n.SetBytes(decodedInfo[offset : offset+datalen])
	offset += datalen

	datalen = int(n.Int64())
	n = n.SetBytes(decodedInfo[offset : offset+datalen]) //public exponent for rsa
	E := int(n.Int64())
	offset += datalen

	datalen = 4
	n = n.SetBytes(decodedInfo[offset : offset+datalen])
	offset += datalen

	datalen = int(n.Int64())
	N = N.SetBytes(decodedInfo[offset : offset+datalen]) //modulus for rsa

	return &rsa.PublicKey{N: N, E: E}, nil
}
