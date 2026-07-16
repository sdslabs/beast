package database

import "testing"

func TestIsChallengeMaintainer(t *testing.T) {
	cleanup := setupSubmissionTestDB(t)
	defer cleanup()

	user := createSubmissionTestUser(t, "author")
	challenge := createSubmissionTestChallenge(t, "challenge", 0, false)
	if err := Db.Create(&UserChallenges{UserID: user.ID, ChallengeID: challenge.ID}).Error; err != nil {
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
