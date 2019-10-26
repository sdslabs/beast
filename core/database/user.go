package database

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/pkg/auth"
	tools "github.com/sdslabs/beastv4/templates"
	log "github.com/sirupsen/logrus"
)

type User struct {
	gorm.Model
	auth.AuthModel

	Challenges []*Challenge `gorm:"many2many:user_challenges;"`
	Name       string       `gorm:"not null"`
	Email      string       `gorm:"non null";unique`
	SshKey     string
}

// Queries all the users entries where the column represented by key
// have the value in value.
func QueryUserEntries(key string, value string) ([]User, error) {
	queryKey := fmt.Sprintf("%s = ?", key)

	var users []User

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where(queryKey, value).Find(&users)
	if tx.RecordNotFound() {
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
	if tx.RecordNotFound() {
		return nil, nil
	}

	return users, tx.Error
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

	return Db.Model(user).Update(m).Error
}

//Get Related Challenges
func GetRelatedChallenges(user *User) ([]Challenge, error) {
	var challenges []Challenge

	DBMux.Lock()
	defer DBMux.Unlock()

	if err := Db.Model(user).Related(&challenges, "Challenges").Error; err != nil {
		return challenges, err
	}

	return challenges, nil
}

//hook after create
func (user *User) AfterCreate(scope *gorm.Scope) error {
	if user.SshKey == "" {
		return nil
	}
	if err := addToAuthorizedKeys(user); err != nil {
		return fmt.Errorf("Error while adding userized_keys : %s", err)
	}
	return nil
}

//hook after update
func (user *User) AfterUpdate(scope *gorm.Scope) error {
	iFace, _ := scope.InstanceGet("gorm:update_attrs")
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

//adds to authorized keys
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
