package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/handlers/auth"
	"github.com/sdslabs/beastv4/api/handlers/me"
)

// RegisterAccountRoutes registers session logout, email verification, and authenticated /me routes.
func RegisterAccountRoutes(accountGroup *gin.RouterGroup) {
	authAccount := accountGroup.Group("/auth")
	{
		authAccount.POST("/logout", auth.LogoutHandler)
		authAccount.POST("/email-verification/request", auth.RequestEmailVerificationHandler)
		authAccount.POST("/email-verification/confirm", auth.ConfirmEmailVerificationHandler)
	}

	meAccount := accountGroup.Group("/me")
	{
		meAccount.GET("", me.GetMeHandler)
		meAccount.PATCH("", me.UpdateMeHandler)
		meAccount.POST("/password", me.ChangePasswordHandler)
	}
}
