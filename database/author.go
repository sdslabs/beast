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
	tools "github.com/sdslabs/beastv4/templates"
)

type Author struct {
	gorm.Model

	Challenges    []Challenge
	Name          string `gorm:"not null"`
	SshKey        string
	Email         string `gorm:"non null"`
	AuthChallenge []byte
}

// Queries all the authors entries where the column represented by key
// have the value in value.
func QueryAuthorEntries(key string, value string) ([]Author, error) {
	queryKey := fmt.Sprintf("%s = ?", key)

	var authors []Author
	tx := Db.Where(queryKey, value).Find(&authors)
	if tx.RecordNotFound() {
		return nil, nil
	}

	if tx.Error != nil {
		return authors, tx.Error
	}

	return authors, nil
}

// Using the column value in key and value in value get the first
// result of the query.
func QueryFirstAuthorEntry(key string, value string) (Author, error) {
	authors, err := QueryAuthorEntries(key, value)
	if err != nil {
		return Author{}, err
	}

	if len(authors) == 0 {
		return Author{}, nil
	}

	return authors[0], nil
}

// Create an entry for the author in the Author table
// It returns an error if anything wrong happen during the
// transaction.
func CreateAuthorEntry(author *Author) error {
	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("Error while starting transaction", tx.Error)
	}

	if err := tx.FirstOrCreate(author, *author).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

//Get Related Challenges
func GetRelatedChallenges(author *Author) []Challenge {
	var challenges []Challenge

	Db.Model(author).Related(&challenges)

	return challenges
}

//hook after create
func (author *Author) AfterCreate(scope *gorm.Scope) error {
	if err := addToAuthorizedKeys(author); err != nil {
		return fmt.Errorf("Error while adding authorized_keys : %s", err)
	}
	return nil
}

//hook after update
func (author *Author) AfterUpdate(scope *gorm.Scope) error {
	iFace, _ := scope.InstanceGet("gorm:update_attrs")
	updatedAttr := iFace.(map[string]interface{})
	if _, ok := updatedAttr["ssh_key"]; ok {
		err := deleteFromAuthorizedKeys(author)
		if err != nil {
			return fmt.Errorf("Error while deleting from authorized_keys : %s", err)
		}
		err = addToAuthorizedKeys(author)
		if err != nil {
			return fmt.Errorf("Error while adding authorized_keys : %s", err)
		}
		err = updateScript(author)
		if err != nil {
			return fmt.Errorf("Error while updating script : %s", err)
		}
	}
	return nil
}

// Updating data in same transaction
func (author *Author) AfterDelete(tx *gorm.DB) error {
	err := deleteFromAuthorizedKeys(author)
	return err
}

type AuthorizedKeyTemplate struct {
	AuthorID string
	Command  string
	PubKey   string
}

func generateContentAuthorizedKeyFile(author *Author) ([]byte, error) {
	SHA256 := sha256.New()
	SHA256.Write([]byte(author.Email))
	scriptPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SCRIPTS_DIR, fmt.Sprintf("%x", SHA256.Sum(nil)))

	data := AuthorizedKeyTemplate{
		AuthorID: strconv.Itoa(int(author.Model.ID)),
		Command:  scriptPath,
		PubKey:   author.SshKey,
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
func addToAuthorizedKeys(author *Author) error {
	f, err := os.OpenFile(config.Cfg.AuthorizedKeysFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Error while opening authorized keys file : %s", err)
	}
	defer f.Close()

	authBytes, err := generateContentAuthorizedKeyFile(author)
	if err != nil {
		return err
	}

	authBytes = bytes.Replace(authBytes, []byte("&#43;"), []byte("+"), -1)

	if _, err := f.Write(authBytes); err != nil {
		return fmt.Errorf("Error while appending key to authorized keys file : %s", err)
	}
	return nil
}

func deleteFromAuthorizedKeys(author *Author) error {

	keys, err := ioutil.ReadFile(config.Cfg.AuthorizedKeysFile)
	if err != nil {
		return fmt.Errorf("Error while reading auth file : %s", err)
	}

	regex := "(?m)[\r\n]+^.*\"SSH_USER=" + strconv.Itoa(int(author.ID)) + "\".*$"

	re := regexp.MustCompile(regex)
	newKeys := []byte(re.ReplaceAllString(string(keys), ""))

	err = ioutil.WriteFile(config.Cfg.AuthorizedKeysFile, newKeys, 0644)
	if err != nil {
		return fmt.Errorf("Error while writing to auth file : %s", err)
	}
	return nil
}
