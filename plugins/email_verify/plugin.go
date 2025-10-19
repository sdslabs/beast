package emailverify

import (
	"errors"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/auth"
	"github.com/sdslabs/beastv4/plugins"
)

type HTTPPlainResp struct {
	Message string `json:"message" example:"Messsage in response to your request"`
}

type HTTPPlainMapResp struct {
	Messages map[string]string `json:"messages" example:"{\"name1\": \"message1\", \"name2\": \"message2\"}"`
}

type HTTPErrorResp struct {
	Error string `json:"error" example:"Error occured while veifying the challenge."`
}

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
		log.Debugf("Checking domain: %s against allowed domain: %s", domain, allowedDomain)
		if domain == allowedDomain {
			return true
		}
	}
	return false
}

func (p *EmailVerifyPlugin) verifyEmailRegister(c *gin.Context, checkFlag bool) {
	name := c.PostForm("name")
	username := c.PostForm("username")
	password := c.PostForm("password")
	email := c.PostForm("email")
	sshKey := c.PostForm("ssh-key")

	name = strings.TrimSpace(name)
	username = strings.TrimSpace(strings.ToLower(username))
	password = strings.TrimSpace(password)
	email = strings.TrimSpace(strings.ToLower(email))
	sshKey = strings.TrimSpace(sshKey)

	if username == "" || password == "" || email == "" {

		c.JSON(http.StatusBadRequest, HTTPPlainResp{
			Message: "Username, password and email can not be empty",
		})
		return
	}

	if len(username) > 12 {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Username cannot be greater than 12 characters",
		})
		return
	}
	if checkFlag {
		if !p.isAllowedEmail(email) {
			log.Warnf("Email domain not allowed: %s", email)
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Registration restricted to organization emails only",
			})
			return
		}
	}

	userEntry := database.User{
		Name:      name,
		AuthModel: auth.CreateModel(username, password, core.USER_ROLES["contestant"]),
		Email:     email,
		SshKey:    sshKey,
	}

	if !config.SkipAuthorization {
		smtpHost := config.Cfg.MailConfig.SMTPHost
		smtpPort := config.Cfg.MailConfig.SMTPPort

		if smtpHost == "" || smtpPort == "" {
			log.Errorf("WARNING: SMTP not configured")
		} else {
			otpEntry, err := database.QueryOTPEntry(email)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusUnauthorized, HTTPErrorResp{
						Error: "OTP not found, email not verified",
					})
					return
				} else {
					log.Println("Failed to query OTP:", err)
					c.JSON(http.StatusInternalServerError, HTTPErrorResp{
						Error: "Failed to send OTP",
					})
					return
				}
			}
			if !otpEntry.Verified {
				c.JSON(http.StatusNotAcceptable, HTTPErrorResp{
					Error: "Email not verified, cannot register user",
				})
				return
			}
		}
	}

	err := database.CreateUserEntry(&userEntry)
	if err != nil {
		c.JSON(http.StatusNotAcceptable, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "User created successfully",
	})
}

func (p *EmailVerifyPlugin) Init(router *gin.Engine) error {
	cfg := config.Cfg
	checkFlag := false
	if len(cfg.EmailVerify.AllowedDomains) > 0 {
		p.AllowedDomains = cfg.EmailVerify.AllowedDomains
		checkFlag = true
	}

	router.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/auth/register" && c.Request.Method == "POST" {
			p.verifyEmailRegister(c, checkFlag)
			c.Abort()
		} else {
			c.Next()
		}
	})

	return nil
}

func init() {
	plugins.Register(&EmailVerifyPlugin{})
}
