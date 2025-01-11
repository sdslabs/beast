package database

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/pkg/auth"
	tools "github.com/sdslabs/beastv4/templates"
	log "github.com/sirupsen/logrus"
	_ "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	auth.AuthModel

	Challenges []*Challenge `gorm:"many2many:user_challenges;"`
	Name       string       `gorm:"not null"`
	Email      string       `gorm:"non null;unique"`
	SshKey     string
	Status     uint `gorm:"not null;default:0"` // 0 for unbanned, 1 for banned
	Score      uint `gorm:"default:0"`
}

// Queries all the users entries where the column represented by key
// have the value in value.
func QueryUserEntries(key string, value string) ([]User, error) {
	queryKey := fmt.Sprintf("%s = ?", key)

	var users []User

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where(queryKey, value).Find(&users)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return users, tx.Error
	}

	return users, nil
}

// Query all the entries in the User table
func QueryAllUsers() ([]User, error) {
	var users []User

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Find(&users)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return users, tx.Error
}

func QueryUserById(authorID uint) (User, error) {
	var user User

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.First(&user, authorID)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return User{}, nil
	}

	return user, tx.Error
}

func GetUserRank(userID uint, userScore uint, updatedAt time.Time) (rank int64, error error) {
	var users []User

	rank = 1

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where("id != ? AND score >= ? AND role == ? AND status == ?", userID, userScore, core.USER_ROLES["contestant"], 0).Find(&users)

	for _, user := range users {
		if user.Score > userScore {
			rank++
		} else if user.UpdatedAt.Before(updatedAt) {
			rank++
		}
	}

	return rank, tx.Error
}

// Using the column value in key and value in value get the first
// result of the query.
func QueryFirstUserEntry(key string, value string) (User, error) {
	users, err := QueryUserEntries(key, value)
	if err != nil {
		return User{}, err
	}

	if len(users) == 0 {
		return User{}, nil
	}

	return users[0], nil
}

// Create an entry for the user in the User table
// It returns an error if anything wrong happen during the
// transaction.
func CreateUserEntry(user *User) error {
	DBMux.Lock()
	defer DBMux.Unlock()
	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("Error while starting transaction", tx.Error)
	}

	if err := tx.FirstOrCreate(user, *user).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// Update an entry for the user in the User table
func UpdateUser(user *User, m map[string]interface{}) error {

	DBMux.Lock()
	defer DBMux.Unlock()

	return Db.Model(user).Updates(m).Error
}

// Get Related Challenges
func GetRelatedChallenges(user *User) ([]Challenge, error) {
	var challenges []Challenge

	DBMux.Lock()
	defer DBMux.Unlock()

	if err := Db.Preload("Tags").Model(user).Association("Challenges").Find(&challenges); err != nil {
		return challenges, err
	}

	return challenges, nil
}

// Check whether challenge is submitted by the user
func CheckPreviousSubmissions(userId uint, challId uint) (bool, error) {
	var userChallenges []UserChallenges
	var count int64
	count = 0

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where("user_id = ? AND challenge_id = ? AND solved = ?", userId, challId, true).Find(&userChallenges).Count(&count)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return false, nil
	}

	return (count >= 1), tx.Error
}

// hook after create
func (user *User) AfterCreate(tx *gorm.DB) error {
	if user.SshKey == "" {
		return nil
	}
	if err := addToAuthorizedKeys(user); err != nil {
		return fmt.Errorf("Error while adding userized_keys : %s", err)
	}
	return nil
}

// hook after update
func (user *User) AfterUpdate(tx *gorm.DB) error {
	iFace, _ := tx.InstanceGet("gorm:update_attrs")
	if iFace == nil {
		return nil
	}
	updatedAttr := iFace.(map[string]interface{})
	if _, ok := updatedAttr["ssh_key"]; ok {
		err := deleteFromAuthorizedKeys(user)
		if err != nil {
			return fmt.Errorf("Error while deleting from userized_keys : %s", err)
		}
		if user.SshKey == "" {
			return nil
		}
		err = addToAuthorizedKeys(user)
		if err != nil {
			return fmt.Errorf("Error while adding userized_keys : %s", err)
		}
		err = updateScript(user)
		if err != nil {
			return fmt.Errorf("Error while updating script : %s", err)
		}
	}
	return nil
}

// Updating data in same transaction
func (user *User) AfterDelete(tx *gorm.DB) error {
	err := deleteFromAuthorizedKeys(user)
	return err
}

type AuthorizedKeyTemplate struct {
	UserID  string
	Command string
	PubKey  string
}

func generateContentAuthorizedKeyFile(user *User) ([]byte, error) {
	SHA256 := sha256.New()
	SHA256.Write([]byte(user.Email))
	scriptPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SCRIPTS_DIR, fmt.Sprintf("%x", SHA256.Sum(nil)))

	data := AuthorizedKeyTemplate{
		UserID:  strconv.Itoa(int(user.Model.ID)),
		Command: scriptPath,
		PubKey:  user.SshKey,
	}

	var authKey bytes.Buffer
	authKeyTemplate, err := template.New("authKey").Parse(tools.AUTHORIZED_KEY_TEMPLATE)
	if err != nil {
		return []byte(""), fmt.Errorf("Error while parsing script template :: %s", err)
	}

	err = authKeyTemplate.Execute(&authKey, data)
	if err != nil {
		return []byte(""), fmt.Errorf("Error while executing script template :: %s", err)
	}

	return authKey.Bytes(), nil
}

// adds to authorized keys
func addToAuthorizedKeys(user *User) error {
	if config.Cfg == nil {
		log.Warn("No config initialized, skipping add to authorized keys hook")
		return nil
	}

	f, err := os.OpenFile(config.Cfg.AuthorizedKeysFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Error while opening userized keys file : %s", err)
	}
	defer f.Close()

	authBytes, err := generateContentAuthorizedKeyFile(user)
	if err != nil {
		return err
	}

	authBytes = bytes.Replace(authBytes, []byte("&#43;"), []byte("+"), -1)

	if _, err := f.Write(authBytes); err != nil {
		return fmt.Errorf("Error while appending key to userized keys file : %s", err)
	}
	return nil
}

func deleteFromAuthorizedKeys(user *User) error {

	if config.Cfg == nil {
		log.Warn("Config is not initialized, skipping delete from auth keys hook")
		return nil
	}

	keys, err := ioutil.ReadFile(config.Cfg.AuthorizedKeysFile)
	if err != nil {
		return fmt.Errorf("Error while reading auth file : %s", err)
	}

	regex := "(?m)[\r\n]+^.*\"SSH_USER=" + strconv.Itoa(int(user.ID)) + "\".*$"

	re := regexp.MustCompile(regex)
	newKeys := []byte(re.ReplaceAllString(string(keys), ""))

	err = ioutil.WriteFile(config.Cfg.AuthorizedKeysFile, newKeys, 0644)
	if err != nil {
		return fmt.Errorf("Error while writing to auth file : %s", err)
	}
	return nil
}
