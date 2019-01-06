package database

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sdslabs/beastv4/core"
	tools "github.com/sdslabs/beastv4/templates"
	"github.com/sdslabs/beastv4/utils"
)

// The `challenges` table has the following columns
// name
// author
// format
// container_id
// image_id
// status
//
// Some hooks needs to be attached to these database transaction, and on the basis of
// the type of the transaction that is performed on the challenge table, we need to
// perform some action.
//
// Use gorm hooks for these purpose, currently the following hooks are
// implemented.
// * AfterUpdate
// * AfterCreate
// * AfterSave
// * AfterDelete
//
// All these hooks are used for generating the access shell script for the challenge
// to the challenge author
type Challenge struct {
	gorm.Model

	Name        string `gorm:"not null;type:varchar(64);unique"`
	Format      string `gorm:"not null"`
	ContainerId string `gorm:"size:64;unique"`
	ImageId     string `gorm:"size:64;unique"`
	Status      string `gorm:"not null;default:'Unknown'"`
	AuthorID    uint   `gorm:"not null"`
	Ports       []Port
}

// Create an entry for the challenge in the Challenge table
// It returns an error if anything wrong happen during the
// transaction.
func CreateChallengeEntry(challenge *Challenge) error {
	tx := Db.Begin()

	if tx.Error != nil {
		return fmt.Errorf("Error while starting transaction", tx.Error)
	}

	if err := tx.FirstOrCreate(challenge, *challenge).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// Query challenges table to get all the entries in the table
func QueryAllChallenges() ([]Challenge, error) {
	var challenges []Challenge

	tx := Db.Find(&challenges)
	if tx.RecordNotFound() {
		return nil, nil
	}

	return challenges, tx.Error
}

// Queries all the challenges entries where the column represented by key
// have the value in value.
func QueryChallengeEntries(key string, value string) ([]Challenge, error) {
	queryKey := fmt.Sprintf("%s = ?", key)

	var challenges []Challenge
	tx := Db.Where(queryKey, value).Find(&challenges)
	if tx.RecordNotFound() {
		return nil, nil
	}

	if tx.Error != nil {
		return challenges, tx.Error
	}

	return challenges, nil
}

// Using the column value in key and value in value get the first
// result of the query.
func QueryFirstChallengeEntry(key string, value string) (Challenge, error) {
	challenges, err := QueryChallengeEntries(key, value)
	if err != nil {
		return Challenge{}, err
	}

	if len(challenges) == 0 {
		return Challenge{}, nil
	}

	return challenges[0], nil
}

// This function updates a challenge entry in the database, whereMap is the map
// which contains key value pairs of column and values to filter out the record
// to update. chall is the Challenge variable with the values to update with.
// This function returns any error that might occur while updating the challenge
// entry which includes the error in case the challenge already does not exist in the
// database.
func UpdateChallengeEntry(whereMap map[string]interface{}, chall Challenge) error {
	var challenge Challenge

	tx := Db.Where(whereMap).First(&challenge)
	if tx.RecordNotFound() {
		return fmt.Errorf("No challenge entry to update : WhereClause : %s", whereMap)
	}

	if tx.Error != nil {
		return tx.Error
	}

	// Update the found entry
	tx = Db.Model(&challenge).Updates(chall)

	return tx.Error
}

//hook after update of challenge
func (challenge *Challenge) AfterUpdate(scope *gorm.Scope) error {
	iFace, _ := scope.InstanceGet("gorm:update_attrs")
	updatedAttr := iFace.(map[string]interface{})

	if _, ok := updatedAttr["container_id"]; ok {
		var author Author
		Db.Model(challenge).Related(&author)
		go updateScript(&author)
	}
	return nil
}

//hook after create of challenge
func (challenge *Challenge) AfterCreate(scope *gorm.Scope) error {
	var author Author
	Db.Model(challenge).Related(&author)
	go updateScript(&author)
	return nil
}

//hook after deleting the challenge
func (challenge *Challenge) AfterDelete() error {
	var author Author
	Db.Model(challenge).Related(&author)
	go updateScript(&author)
	return nil
}

type ScriptFile struct {
	Author     string
	Challenges map[string]string
}

//updates user script
func updateScript(author *Author) error {

	time.Sleep(time.Second)

	SHA256 := sha256.New()
	SHA256.Write([]byte(author.Email))
	scriptPath := filepath.Join(core.BEAST_GLOBAL_DIR, core.BEAST_SCRIPTS_DIR, fmt.Sprintf("%x", SHA256.Sum(nil)))

	challs := GetRelatedChallenges(author)

	mapOfChall := make(map[string]string)

	for _, chall := range challs {
		if chall.ContainerId != utils.GetTempImageId(chall.Name) {
			mapOfChall[chall.Name] = chall.ContainerId
		}
	}

	data := ScriptFile{
		Author:     author.Name,
		Challenges: mapOfChall,
	}

	var script bytes.Buffer
	scriptTemplate, err := template.New("script").Parse(tools.SSH_LOGIN_SCRIPT_TEMPLATE)
	if err != nil {
		return fmt.Errorf("Error while parsing script template :: %s", err)
	}

	err = scriptTemplate.Execute(&script, data)
	if err != nil {
		return fmt.Errorf("Error while executing script template :: %s", err)
	}

	return ioutil.WriteFile(scriptPath, script.Bytes(), 0755)
}
