package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/pkg/auth"
)

func TestManagerAuthorizationRequiresVerifiedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth.Init(1, 32, 60, "issuer", "secret", []string{"author"}, []string{"admin"}, []string{"contestant"})

	router := gin.New()
	router.GET("/protected", managerAuthorize, func(c *gin.Context) {
		claims, exists := c.Get("authClaims")
		if !exists || claims.(*auth.CustomClaims).User != "alice" {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})

	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorized.Code)
	}

	token, err := auth.GenerateJWT(auth.AuthModel{Username: "alice", Role: core.USER_ROLES["author"]})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	authorized := httptest.NewRecorder()
	router.ServeHTTP(authorized, request)
	if authorized.Code != http.StatusServiceUnavailable {
		t.Fatalf("unbacked token status = %d, body = %s", authorized.Code, authorized.Body.String())
	}
}
