package api

import (
	"testing"

	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/auth"
)

func TestCanViewChallengeSecretsForIntrinsicRoles(t *testing.T) {
	challenge := database.Challenge{AuthorID: 7}
	challenge.ID = 11
	admin := database.User{AuthModel: auth.AuthModel{Role: core.USER_ROLES["admin"]}}
	if allowed, err := canViewChallengeSecrets(&admin, &challenge); err != nil || !allowed {
		t.Fatalf("admin access = %v, %v", allowed, err)
	}
	owner := database.User{AuthModel: auth.AuthModel{Role: core.USER_ROLES["author"]}}
	owner.ID = 7
	if allowed, err := canViewChallengeSecrets(&owner, &challenge); err != nil || !allowed {
		t.Fatalf("owner access = %v, %v", allowed, err)
	}
	contestant := database.User{AuthModel: auth.AuthModel{Role: core.USER_ROLES["contestant"]}}
	if allowed, err := canViewChallengeSecrets(&contestant, &challenge); err != nil || allowed {
		t.Fatalf("contestant access = %v, %v", allowed, err)
	}
}
