package api

import (
	"testing"

	"github.com/sdslabs/beastv4/core"
	challengeConfig "github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/auth"
)

func TestUserOwnsChallengeConfig(t *testing.T) {
	configuration := challengeConfig.BeastChallengeConfig{
		Author:      challengeConfig.Author{Email: "author@example.com"},
		Maintainers: []challengeConfig.Author{{Email: "maintainer@example.com"}},
	}
	tests := []struct {
		name string
		user database.User
		want bool
	}{
		{name: "author", user: database.User{Email: "AUTHOR@example.com"}, want: true},
		{name: "maintainer", user: database.User{Email: "maintainer@example.com"}, want: true},
		{name: "unrelated", user: database.User{Email: "other@example.com"}, want: false},
		{name: "admin", user: database.User{Email: "other@example.com", AuthModel: auth.AuthModel{Role: core.USER_ROLES["admin"]}}, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := userOwnsChallengeConfig(test.user, configuration); got != test.want {
				t.Fatalf("userOwnsChallengeConfig() = %t, want %t", got, test.want)
			}
		})
	}
}
