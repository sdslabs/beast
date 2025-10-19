package emailverify

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/config"
)

type EmailVerifyPlugin struct {
	AllowedDomains []string
}

func (p *EmailVerifyPlugin) Name() string {
	return "EmailVerifyPlugin"
}

func (p *EmailVerifyPlugin) Description() string {
	return "A plugin to restrict registration to certain email domains"
}

func (p *EmailVerifyPlugin) isAllowedEmail(email string) bool {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	domain := parts[1]
	for _, allowedDomain := range p.AllowedDomains {
		if domain == allowedDomain {
			return true
		}
	}
	return false
}

func (p *EmailVerifyPlugin) Init(router *gin.Engine) error {
	cfg := config.Cfg
	checkFlag := false
	if len(cfg.EmailVerify.AllowedDomains) > 0 {
		p.AllowedDomains = cfg.EmailVerify.AllowedDomains
		checkFlag = true
	}
	authGroup := router.Group("/auth")
	authGroup.Use(func(c *gin.Context) {
		if checkFlag {
			if strings.Contains(c.Request.URL.Path, "/register") && c.Request.Method == "POST" {
				email := c.PostForm("email")
				if !p.isAllowedEmail(email) {
					c.JSON(http.StatusForbidden, gin.H{"error": "Registration restricted to organization emails only"})
					c.Abort()
					return
				}
			}
		}
		c.Next()
	})
	return nil
}
