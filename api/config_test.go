package api

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/database"
)

func challengeUpdateContext(values url.Values) *gin.Context {
	request := httptest.NewRequest("POST", "/", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request
	return context
}

func TestParseChallengeUpdateRejectsInvalidValues(t *testing.T) {
	challenge := database.Challenge{MaxPoints: 500, MinPoints: 100}
	tests := []url.Values{
		{"points": {"99"}},
		{"ports": {"80,80"}},
		{"tags": {"web" + core.DELIMITER + "../admin"}},
		{"assets": {"../flag"}},
		{"additionalLinks": {"file:///etc/passwd"}},
	}
	for _, values := range tests {
		if _, _, _, err := parseChallengeUpdate(challengeUpdateContext(values), challenge); err == nil {
			t.Fatalf("expected invalid update to fail: %v", values)
		}
	}
}

func TestParseChallengeUpdateProducesTypedValues(t *testing.T) {
	values := url.Values{
		"points": {"250"},
		"ports":  {"8080, 8443"},
		"tags":   {"web" + core.DELIMITER + "beginner"},
	}
	updates, ports, tags, err := parseChallengeUpdate(challengeUpdateContext(values), database.Challenge{MaxPoints: 500, MinPoints: 100})
	if err != nil {
		t.Fatal(err)
	}
	if updates["points"] != uint(250) || len(*ports) != 2 || len(*tags) != 2 {
		t.Fatalf("unexpected parsed update: updates=%v ports=%v tags=%v", updates, ports, tags)
	}
}
