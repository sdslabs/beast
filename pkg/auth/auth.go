package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"golang.org/x/crypto/pbkdf2"
)

var (
	ITERATIONS  int
	HASH_LENGTH int
	TIME_PERIOD int64
	ISSUER      string
	JWTSECRET   string
)

type AuthModel struct {
	Username string `gorm:"not null";unique`
	Password []byte `gorm:"non null"`
	Role     string `gorm:"non null"`
	Salt     []byte
}

func CreateModel(username, password, role string) AuthModel {

	salt := make([]byte, 16)
	rand.Read(salt)

	auth1 := AuthModel{
		Username: username,
		Password: pbkdf2.Key([]byte(password), salt, ITERATIONS, HASH_LENGTH, sha256.New),
		Salt:     salt,
		Role:     role,
	}
	return auth1
}

func Authenticate(username, password string, authEntry AuthModel) (string, error) {
	hashedPassword := pbkdf2.Key([]byte(password), authEntry.Salt, ITERATIONS, HASH_LENGTH, sha256.New)
	if !bytes.Equal(hashedPassword, authEntry.Password) {
		return "", errors.New("The username or password is invalid")
	}

	return GenerateJWT(authEntry)
}

func Init(iter, hashLength int, timePeriod int64, issuer, jwtSecret string) {
	ITERATIONS = iter
	HASH_LENGTH = hashLength
	TIME_PERIOD = timePeriod
	ISSUER = issuer
	JWTSECRET = jwtSecret
}
