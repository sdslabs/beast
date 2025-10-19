package api

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/auth"
	"gorm.io/gorm"
)

func generateOTP() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%06d", r.Intn(1000000)) // 6-digit OTP
}

// sendEmail sends an OTP email using an SMTP client with TLS. Falls back to plain text if template is missing.
func sendEmail(email, otp string) error {
	from := config.Cfg.MailConfig.From
	password := config.Cfg.MailConfig.Password
	smtpHost := config.Cfg.MailConfig.SMTPHost
	smtpPort := config.Cfg.MailConfig.SMTPPort

	// Email subject
	subject := "Your OTP Code"

	// Path to email template
	emailTemplatePath := filepath.Join(
		core.BEAST_GLOBAL_DIR,
		core.BEAST_ASSETS_DIR,
		core.BEAST_EMAIL_TEMPLATE_DIR,
		"email_template.html",
	)

	// Check if template file exists
	var body bytes.Buffer
	_, err := os.Stat(emailTemplatePath)
	if err == nil {
		// Template exists, parse and execute
		tmpl, err := template.ParseFiles(emailTemplatePath)
		if err != nil {
			log.Println("Failed to read email template:", err)
			return err
		}

		emailData := struct {
			OTP string
		}{OTP: otp}

		if err := tmpl.Execute(&body, emailData); err != nil {
			log.Println("Failed to execute email template:", err)
			return err
		}
	} else {
		// Template does not exist, send plain text email
		log.Println("Template not found, sending plain text email.")
		body.WriteString(fmt.Sprintf("Hello,\n\nYour OTP is: %s\nThis OTP will expire in 10 minutes.\n\nRegards,\nTeam", otp))
	}

	// Create email headers
	message := fmt.Sprintf("From: %s\r\n", from) +
		fmt.Sprintf("To: %s\r\n", email) +
		fmt.Sprintf("Subject: %s\r\n", subject) +
		"MIME-Version: 1.0\r\n"

	// Set Content-Type based on template availability
	if body.String()[0] == '<' {
		message += "Content-Type: text/html; charset=\"utf-8\"\r\n\r\n"
	} else {
		message += "Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n"
	}

	message += body.String()

	// Setup TLS connection
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Set true only if SMTP server uses self-signed certs
		ServerName:         smtpHost,
	}

	// Connect to SMTP server
	conn, err := tls.Dial("tcp", smtpHost+":"+smtpPort, tlsConfig)
	if err != nil {
		log.Println("Failed to connect to SMTP server:", err)
		return err
	}

	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		log.Println("Failed to create SMTP client:", err)
		return err
	}
	defer client.Close()

	// Authenticate
	auth := smtp.PlainAuth("", from, password, smtpHost)
	if err := client.Auth(auth); err != nil {
		log.Println("SMTP authentication failed:", err)
		return err
	}

	// Set sender and recipient
	if err := client.Mail(from); err != nil {
		log.Println("Failed to set sender:", err)
		return err
	}

	if err := client.Rcpt(email); err != nil {
		log.Println("Failed to set recipient:", err)
		return err
	}

	// Write email data
	w, err := client.Data()
	if err != nil {
		log.Println("Failed to get SMTP data writer:", err)
		return err
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		log.Println("Failed to write email content:", err)
		return err
	}

	err = w.Close()
	if err != nil {
		log.Println("Failed to close SMTP writer:", err)
		return err
	}

	// Quit SMTP session
	if err := client.Quit(); err != nil {
		log.Println("Failed to close SMTP connection:", err)
		return err
	}

	fmt.Println("OTP email sent successfully to", email)
	return nil
}

// @Summary Send OTP for email verification
// @Description Sends a one-time password to the specified email address for verification
// @Tags auth
// @Accept x-www-form-urlencoded
// @Produce json
// @Param email formData string true "User's email address"
// @Success 200 {object} api.HTTPPlainResp "OTP sent successfully"
// @Success 200 {object} api.HTTPPlainResp "Email already verified"
// @Failure 500 {object} api.HTTPErrorResp "SMTP not configured or Failed to send/store OTP"
// @Router /auth/send-otp [post]
func sendOTPHandler(c *gin.Context) {
	if config.SkipAuthorization {
		return
	}
	email := c.PostForm("email")
	email = strings.TrimSpace(strings.ToLower(email))

	smtpHost := config.Cfg.MailConfig.SMTPHost
	smtpPort := config.Cfg.MailConfig.SMTPPort

	if smtpHost == "" || smtpPort == "" {
		log.Printf("WARNING: %s", "SMTP not configured")
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "SMTP not configured",
		})
		return
	}

	otp := generateOTP()
	expiry := time.Now().Add(5 * time.Minute) // OTP expires in 5 minutes

	otpEntry, err := database.QueryOTPEntry(email)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			otpEntry = database.OTP{
				Email:  email,
				Code:   otp,
				Expiry: expiry,
			}
		} else {
			log.Println("Failed to query OTP:", err)
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "Failed to send OTP",
			})
			return
		}
	}

	if otpEntry.Verified {
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "Email already verified",
		})
		return
	}

	otpEntry.Code = otp
	otpEntry.Expiry = expiry

	err = database.CreateOTPEntry(&otpEntry)
	if err != nil {
		log.Println("Failed to store OTP:", err)
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "Failed to store OTP",
		})
		return
	}

	// Send OTP to email
	err = sendEmail(email, otp)
	if err != nil {
		log.Println("Failed to send OTP:", err)
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "Failed to send OTP",
		})
		return
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "OTP sent successfully",
	})
}

// @Summary Verify email OTP
// @Description Verifies the OTP sent to user's email
// @Tags auth
// @Accept x-www-form-urlencoded
// @Produce json
// @Param email formData string true "User's email address"
// @Param otp formData string true "OTP received in email"
// @Success 200 {object} api.HTTPPlainResp "OTP verified successfully"
// @Failure 400 {object} api.HTTPErrorResp "Invalid or expired OTP"
// @Failure 500 {object} api.HTTPErrorResp "SMTP not configured or verification failed"
// @Router /auth/verify-otp [post]
func verifyOTPHandler(c *gin.Context) {
	email := c.PostForm("email")
	otp := strings.TrimSpace(c.PostForm("otp"))
	email = strings.TrimSpace(strings.ToLower(email))

	if !config.SkipAuthorization {
		smtpHost := config.Cfg.MailConfig.SMTPHost
		smtpPort := config.Cfg.MailConfig.SMTPPort

		if smtpHost == "" || smtpPort == "" {
			log.Printf("WARNING: %s", "SMTP not configured")
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "SMTP not configured",
			})
			return
		}

		otpEntry, err := database.QueryOTPEntry(email)

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusUnauthorized, HTTPErrorResp{
					Error: "OTP not found",
				})
			} else {
				log.Println("Failed to query OTP:", err)
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "Failed to send OTP",
				})
				return
			}
		}

		if otpEntry.Verified {
			c.JSON(http.StatusOK, HTTPPlainResp{
				Message: "Email already verified",
			})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "Failed to verify OTP",
			})
			return
		}

		if otpEntry.Code != otp {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "Invalid OTP",
			})
			return
		}

		if time.Now().After(otpEntry.Expiry) {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "OTP expired",
			})
			return
		}
	}
	err := database.VerifyOTPEntry(email)

	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "Failed to verify OTP",
		})
		return
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "OTP verified successfully",
	})
}

// @Summary Send OTP for password reset
// @Description Sends a one-time password to email for password reset flow
// @Tags auth
// @Accept x-www-form-urlencoded
// @Produce json
// @Param email formData string true "User's email address"
// @Success 200 {object} HTTPPlainResp "OTP sent successfully"
// @Failure 404 {object} HTTPErrorResp "User not found"
// @Failure 500 {object} HTTPErrorResp "Failed to send OTP or SMTP not configured"
// @Router /auth/send-otp-forget [post]
func sendOTPForForgetHandler(c *gin.Context) {
	if config.SkipAuthorization {
		return
	}
	email := c.PostForm("email")
	email = strings.TrimSpace(strings.ToLower(email))

	smtpHost := config.Cfg.MailConfig.SMTPHost
	smtpPort := config.Cfg.MailConfig.SMTPPort

	if smtpHost == "" || smtpPort == "" {
		log.Printf("WARNING: %s", "SMTP not configured")
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "SMTP not configured",
		})
		return
	}

	otp := generateOTP()
	expiry := time.Now().Add(5 * time.Minute) // OTP expires in 5 minutes

	otpEntry, err := database.QueryOTPEntry(email)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			otpEntry = database.OTP{
				Email:  email,
				Code:   otp,
				Expiry: expiry,
			}
		} else {
			log.Println("Failed to query OTP:", err)
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "Failed to send OTP",
			})
			return
		}
	}

	otpEntry.Code = otp
	otpEntry.Expiry = expiry

	err = database.CreateOTPEntry(&otpEntry)
	if err != nil {
		log.Println("Failed to store OTP:", err)
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "Failed to store OTP",
		})
		return
	}

	// Send OTP to email
	err = sendEmail(email, otp)
	if err != nil {
		log.Println("Failed to send OTP:", err)
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "Failed to send OTP",
		})
		return
	}

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "OTP sent successfully",
	})
}

// @Summary Verify OTP for password reset
// @Description Verifies the OTP sent for password reset and updates the password
// @Tags auth
// @Accept x-www-form-urlencoded
// @Produce json
// @Param email formData string true "User's email address"
// @Param otp formData string true "OTP received in email"
// @Param new_password formData string true "New password to set"
// @Success 200 {object} HTTPPlainResp "Password reset successful"
// @Failure 400 {object} HTTPErrorResp "Invalid or expired OTP"
// @Failure 404 {object} HTTPErrorResp "User not found"
// @Failure 500 {object} HTTPErrorResp "Verification failed"
// @Router /auth/verify-otp-forget [post]
func verifyOTPForForgetHandler(c *gin.Context) {
	email := c.PostForm("email")
	otp := strings.TrimSpace(c.PostForm("otp"))
	email = strings.TrimSpace(strings.ToLower(email))
	if !config.SkipAuthorization {
		smtpHost := config.Cfg.MailConfig.SMTPHost
		smtpPort := config.Cfg.MailConfig.SMTPPort

		if smtpHost == "" || smtpPort == "" {
			log.Printf("WARNING: %s", "SMTP not configured")
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{
				Error: "SMTP not configured",
			})
			return
		}

		otpEntry, err := database.QueryOTPEntry(email)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusUnauthorized, HTTPErrorResp{
					Error: "OTP not found",
				})
			} else {
				log.Println("Failed to query OTP:", err)
				c.JSON(http.StatusInternalServerError, HTTPErrorResp{
					Error: "Failed to verify OTP",
				})
			}
			return
		}

		if otpEntry.Code != otp {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "Invalid OTP",
			})
			return
		}

		if time.Now().After(otpEntry.Expiry) {
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{
				Error: "OTP expired",
			})
			return
		}
	}
	userEntry, err := database.QueryFirstUserEntry("email", email)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}

	if userEntry.Status == 1 {
		c.JSON(http.StatusForbidden, HTTPErrorResp{
			Error: "The user has been banned from this competition. Please contact competition admin for more information",
		})
		return
	}

	t := time.Now().Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.CustomClaims{
		User:      userEntry.Username,
		Role:      userEntry.Role,
		ExpiresAt: t + 300,
		IssuedAt:  t,
		Issuer:    auth.ISSUER,
	})

	tempToken, err := token.SignedString([]byte(auth.JWTSECRET))

	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "Failed to create authentication session",
		})
		return
	}

	c.JSON(http.StatusOK, HTTPAuthorizeResp{
		Token:   tempToken,
		Role:    userEntry.Role,
		Message: "OTP verified. Use this token to reset your password within 5 minutes.",
	})
}
