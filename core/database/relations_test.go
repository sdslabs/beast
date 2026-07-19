package database

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestTagQueriesReturnChallengeRelations(t *testing.T) {
	cleanup := setupSubmissionTestDB(t)
	defer cleanup()

	challenge := createSubmissionTestChallenge(t, "tagged-challenge", -1, false)
	tags := []*Tag{{TagName: "web"}, {TagName: "linux"}}
	if err := UpdateTags(tags, &challenge); err != nil {
		t.Fatal(err)
	}

	related, err := GetRelatedTags(&challenge)
	if err != nil {
		t.Fatal(err)
	}
	if len(related) != 2 {
		t.Fatalf("expected two related tags, got %d", len(related))
	}

	unique, err := QueryAllUniqueTags()
	if err != nil {
		t.Fatal(err)
	}
	if len(unique) != 2 || unique[0] != "linux" || unique[1] != "web" {
		t.Fatalf("unexpected unique tags: %v", unique)
	}

	challenges, err := QueryRelatedChallenges(&Tag{TagName: "web"})
	if err != nil {
		t.Fatal(err)
	}
	if len(challenges) != 1 || challenges[0].ID != challenge.ID {
		t.Fatalf("unexpected related challenges: %#v", challenges)
	}
	if _, err := QueryRelatedChallenges(&Tag{TagName: "missing"}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected missing tag error, got %v", err)
	}
}

func TestPortsAreUniquePerServerAndDeleteByID(t *testing.T) {
	cleanup := setupSubmissionTestDB(t)
	defer cleanup()

	first := createSubmissionTestChallenge(t, "first-port", -1, false)
	first.ServerDeployed = "worker-a"
	if err := Db.Model(&first).Update("server_deployed", first.ServerDeployed).Error; err != nil {
		t.Fatal(err)
	}
	second := createSubmissionTestChallenge(t, "second-port", -1, false)
	second.ServerDeployed = "worker-b"
	if err := Db.Model(&second).Update("server_deployed", second.ServerDeployed).Error; err != nil {
		t.Fatal(err)
	}

	firstPort, err := PortEntryGetOrCreate(&Port{ChallengeID: first.ID, PortNo: 8080})
	if err != nil {
		t.Fatal(err)
	}
	secondPort, err := PortEntryGetOrCreate(&Port{ChallengeID: second.ID, PortNo: 8080})
	if err != nil {
		t.Fatal(err)
	}
	if firstPort.Server == secondPort.Server {
		t.Fatalf("expected distinct port servers, got %q", firstPort.Server)
	}

	third := createSubmissionTestChallenge(t, "third-port", -1, false)
	third.ServerDeployed = first.ServerDeployed
	if err := Db.Model(&third).Update("server_deployed", third.ServerDeployed).Error; err != nil {
		t.Fatal(err)
	}
	existing, err := PortEntryGetOrCreate(&Port{ChallengeID: third.ID, PortNo: 8080})
	if err != nil {
		t.Fatal(err)
	}
	if existing.ChallengeID != first.ID {
		t.Fatalf("same-server port should remain owned by challenge %d, got %d", first.ID, existing.ChallengeID)
	}

	if err := DeleteRelatedPorts(nil); err != nil {
		t.Fatal(err)
	}
	if err := DeleteRelatedPorts([]Port{firstPort}); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := Db.Model(&Port{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected only one port after targeted deletion, got %d", count)
	}
}

func TestChallengeIdentifiersAllowEmptyAndProtectActiveValues(t *testing.T) {
	cleanup := setupSubmissionTestDB(t)
	defer cleanup()

	first := Challenge{Name: "empty-identifiers-a", Format: "static", AuthorID: 1}
	second := Challenge{Name: "empty-identifiers-b", Format: "static", AuthorID: 1}
	if err := Db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := Db.Create(&second).Error; err != nil {
		t.Fatalf("empty identifiers should be reusable: %v", err)
	}

	first.ContainerId = "container-id"
	first.ImageId = "image-id"
	if err := Db.Save(&first).Error; err != nil {
		t.Fatal(err)
	}
	duplicate := Challenge{
		Name:        "duplicate-identifiers",
		Format:      "docker",
		AuthorID:    1,
		ContainerId: first.ContainerId,
		ImageId:     first.ImageId,
	}
	if err := Db.Create(&duplicate).Error; err == nil {
		t.Fatal("active challenge identifiers must remain unique")
	}

	if err := Db.Delete(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := Db.Create(&duplicate).Error; err != nil {
		t.Fatalf("deleted challenge identifiers should be reusable: %v", err)
	}
}

func TestSaveTransactionPreservesRepeatedActions(t *testing.T) {
	cleanup := setupSubmissionTestDB(t)
	defer cleanup()

	user := createSubmissionTestUser(t, "audit-user")
	challenge := createSubmissionTestChallenge(t, "audit-challenge", -1, false)
	for range 2 {
		if err := SaveTransaction(&Transaction{Action: "deploy", UserID: user.ID, ChallengeID: challenge.ID}); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := Db.Model(&Transaction{}).Where("user_id = ? AND challenge_id = ? AND action = ?", user.ID, challenge.ID, "deploy").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("transaction count = %d, want 2", count)
	}
}
