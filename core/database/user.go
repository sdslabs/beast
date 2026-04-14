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
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	auth.AuthModel

	Challenges  []*Challenge `gorm:"many2many:user_challenges;"`
	Name        string       `gorm:"not null"`
	Email       string       `gorm:"non null;unique"`
	SshKey      string
	Status      uint    `gorm:"not null;default:0"` // 0 for unbanned, 1 for banned
	Score       uint    `gorm:"default:0"`
	FrozenScore uint    `gorm:"default:0"`
	Hints       []*Hint `gorm:"many2many:user_hints;references:HintID;joinReferences:HintID"`
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

	tx := Db.Where("id != ? AND score >= ? AND role = ? AND status = ?", userID, userScore, core.USER_ROLES["contestant"], 0).Find(&users)

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
		return fmt.Errorf("Error while starting transaction: %v", tx.Error)
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

// UserSolvedChallenge represents a challenge solved by a user with the actual solve timestamp
type UserSolvedChallenge struct {
	ChallengeID uint
	Name        string
	Type        string
	Points      uint
	Tags        []*Tag
	SolvedAt    time.Time
}

// GetUserSolvedChallenges returns distinct challenges solved by a user, with the actual solve timestamps.
// Unlike GetRelatedChallenges, this avoids duplicates from multiple user_challenges rows per challenge.
func GetUserSolvedChallenges(userID uint) ([]UserSolvedChallenge, error) {
	type solveRow struct {
		ChallengeID uint
		Name        string
		Type        string
		Points      uint
		SolvedAt    time.Time
	}
	var rows []solveRow

	DBMux.Lock()
	defer DBMux.Unlock()

	err := Db.Table("user_challenges").
		Select("DISTINCT ON (user_challenges.challenge_id) user_challenges.challenge_id, challenges.name, challenges.type, challenges.points, user_challenges.created_at as solved_at").
		Joins("JOIN challenges ON challenges.id = user_challenges.challenge_id").
		Where("user_challenges.user_id = ? AND user_challenges.solved = ?", userID, true).
		Order("user_challenges.challenge_id, user_challenges.created_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return []UserSolvedChallenge{}, nil
	}

	// Load tags for all solved challenges in a single query instead of one per challenge.
	challengeIDs := make([]uint, len(rows))
	for i, r := range rows {
		challengeIDs[i] = r.ChallengeID
	}

	type tagRow struct {
		ChallengeID uint
		TagID       uint
		TagName     string
	}
	var tagRows []tagRow
	err = Db.Table("tags").
		Select("tag_challenges.challenge_id, tags.id as tag_id, tags.tag_name").
		Joins("JOIN tag_challenges ON tag_challenges.tag_id = tags.id").
		Where("tag_challenges.challenge_id IN ?", challengeIDs).
		Scan(&tagRows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load tags for solved challenges: %w", err)
	}

	tagMap := make(map[uint][]*Tag)
	for _, tr := range tagRows {
		tagMap[tr.ChallengeID] = append(tagMap[tr.ChallengeID], &Tag{
			Model:   gorm.Model{ID: tr.TagID},
			TagName: tr.TagName,
		})
	}

	results := make([]UserSolvedChallenge, 0, len(rows))
	for _, r := range rows {
		results = append(results, UserSolvedChallenge{
			ChallengeID: r.ChallengeID,
			Name:        r.Name,
			Type:        r.Type,
			Points:      r.Points,
			Tags:        tagMap[r.ChallengeID],
			SolvedAt:    r.SolvedAt,
		})
	}

	return results, nil
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

func QueryTopUsersByScore(limit int) ([]User, error) {
	var users []User

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where("role = ? AND status = ?", core.USER_ROLES["contestant"], 0).
		Order("score desc, updated_at asc").
		Limit(limit).
		Find(&users)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		log.Warn("No users found")
		return nil, fmt.Errorf("no users found")
	}

	return users, tx.Error
}

func QueryUsersByScoreOffsetLimit(limit, offset int) ([]User, error) {
	var users []User

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where("role = ? AND status = ?", core.USER_ROLES["contestant"], 0).
		Order("score desc, updated_at asc").
		Limit(limit).
		Offset(offset).
		Find(&users)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		log.Warn("No users found")
		return nil, fmt.Errorf("no users found")
	}

	return users, tx.Error
}

func QueryTopUsersByFrozenScore(limit int) ([]User, error) {
	var users []User

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where("role = ? AND status = ?", core.USER_ROLES["contestant"], 0).
		Order("frozen_score desc, updated_at asc").
		Limit(limit).
		Find(&users)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		log.Warn("No users found")
		return nil, fmt.Errorf("no users found")
	}

	return users, tx.Error
}

func QueryUsersByFrozenScoreOffsetLimit(limit, offset int) ([]User, error) {
	var users []User

	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Where("role = ? AND status = ?", core.USER_ROLES["contestant"], 0).
		Order("frozen_score desc, updated_at asc").
		Limit(limit).
		Offset(offset).
		Find(&users)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		log.Warn("No users found")
		return nil, fmt.Errorf("no users found")
	}

	return users, tx.Error
}

func UpdateFrozenScores() error {
	DBMux.Lock()
	defer DBMux.Unlock()
	return Db.Exec("UPDATE users SET frozen_score = score").Error
}

func ResetFrozenScores() error {
	DBMux.Lock()
	defer DBMux.Unlock()
	return Db.Exec("UPDATE users SET frozen_score = 0").Error
}

func IsFrozenScoreSet() (bool, error) {
	var count int64
	DBMux.Lock()
	defer DBMux.Unlock()
	err := Db.Model(&User{}).Where("frozen_score != 0").Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func GetUserCount() (int64, error) {
	var count int64
	DBMux.Lock()
	defer DBMux.Unlock()
	tx := Db.Model(&User{}).Where("role = ?", "contestant").Count(&count)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return count, tx.Error
}

func QueryAllUniqueTags() ([]string, error) {
	var tags []string
	DBMux.Lock()
	defer DBMux.Unlock()

	tx := Db.Model(&Challenge{}).Distinct().Pluck("tag", &tags)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tags, nil
}
