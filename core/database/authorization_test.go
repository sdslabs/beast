package database

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestSetChallengeRelationsRequiresPersistedModels(t *testing.T) {
	if err := SetChallengeRelations(nil, nil, nil); err == nil {
		t.Fatal("expected missing challenge error")
	}
	if err := SetChallengeRelations(&Challenge{}, nil, nil); err == nil {
		t.Fatal("expected unpersisted challenge error")
	}
	if err := SetChallengeRelations(&Challenge{Model: gorm.Model{ID: 1}}, nil, []*User{{}}); err == nil {
		t.Fatal("expected unpersisted manager error")
	}
}

func TestIsChallengeMaintainer(t *testing.T) {
	cleanup := setupSubmissionTestDB(t)
	defer cleanup()

	user := createSubmissionTestUser(t, "author")
	challenge := createSubmissionTestChallenge(t, "challenge", 0, false)
	if err := Db.Create(&ChallengeMaintainer{UserID: user.ID, ChallengeID: challenge.ID}).Error; err != nil {
		t.Fatal(err)
	}

	allowed, err := IsChallengeMaintainer(user.ID, challenge.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Fatal("expected related user to be a maintainer")
	}
	allowed, err = IsChallengeMaintainer(user.ID+1, challenge.ID)
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Fatal("unrelated user was treated as maintainer")
	}
}

func TestUserQueriesAndCreationFailClosed(t *testing.T) {
	cleanup := setupSubmissionTestDB(t)
	defer cleanup()

	if _, err := QueryUserById(999999); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected missing user error, got %v", err)
	}
	if err := CreateUserEntry(nil); err == nil {
		t.Fatal("expected nil user error")
	}

	user := createSubmissionTestUser(t, "unique-user")
	duplicate := User{Name: "duplicate", Email: user.Email, AuthModel: user.AuthModel}
	if err := CreateUserEntry(&duplicate); err == nil {
		t.Fatal("expected duplicate user error")
	}
}

func TestMigrateChallengeMaintainersPreservesManagers(t *testing.T) {
	cleanup := setupSubmissionTestDB(t)
	defer cleanup()

	author := createSubmissionTestUser(t, "challenge-author")
	if err := Db.Model(&author).Update("role", "author").Error; err != nil {
		t.Fatal(err)
	}
	maintainer := createSubmissionTestUser(t, "challenge-maintainer")
	if err := Db.Model(&maintainer).Update("role", "maintainer").Error; err != nil {
		t.Fatal(err)
	}
	challenge := createSubmissionTestChallenge(t, "managed-challenge", 0, false)
	if err := Db.Model(&challenge).Update("author_id", author.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := Db.Create(&UserChallenges{UserID: maintainer.ID, ChallengeID: challenge.ID}).Error; err != nil {
		t.Fatal(err)
	}

	if err := MigrateChallengeMaintainers(); err != nil {
		t.Fatal(err)
	}
	for _, userID := range []uint{author.ID, maintainer.ID} {
		allowed, err := IsChallengeMaintainer(userID, challenge.ID)
		if err != nil || !allowed {
			t.Fatalf("manager %d was not migrated: allowed=%t err=%v", userID, allowed, err)
		}
	}
}
