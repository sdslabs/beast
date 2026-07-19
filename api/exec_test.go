package api

import (
	"strings"
	"testing"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/auth"
	"gorm.io/gorm"
)

func TestExecChallengeRequestValidation(t *testing.T) {
	tests := []ExecChallengeRequest{
		{},
		{Command: []string{""}},
		{Command: []string{"echo", string([]byte{'a', 0, 'b'})}},
		{Command: []string{"echo"}, TimeoutSeconds: maxExecTimeout + 1},
		{Command: []string{strings.Repeat("a", maxExecArgumentBytes+1)}},
	}
	for _, request := range tests {
		if err := request.validate(); err == nil {
			t.Fatalf("expected invalid request: %+v", request)
		}
	}

	valid := ExecChallengeRequest{Command: []string{"sh", "-lc", "id"}}
	if err := valid.validate(); err != nil {
		t.Fatal(err)
	}
	if valid.TimeoutSeconds != defaultExecTimeout {
		t.Fatalf("default timeout = %d", valid.TimeoutSeconds)
	}
}

func TestUserCanExecChallenge(t *testing.T) {
	challenge := database.Challenge{AuthorID: 1}
	if !userCanExecChallenge(database.User{Model: gorm.Model{ID: 1}, AuthModel: auth.AuthModel{Role: core.USER_ROLES["author"]}}, challenge, false) {
		t.Fatal("challenge author was denied")
	}
	if !userCanExecChallenge(database.User{Model: gorm.Model{ID: 2}, AuthModel: auth.AuthModel{Role: core.USER_ROLES["maintainer"]}}, challenge, true) {
		t.Fatal("challenge maintainer was denied")
	}
	if userCanExecChallenge(database.User{Model: gorm.Model{ID: 3}, AuthModel: auth.AuthModel{Role: core.USER_ROLES["contestant"]}}, challenge, true) {
		t.Fatal("contestant was allowed")
	}
}
