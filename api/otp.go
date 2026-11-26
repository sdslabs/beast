package api

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/big"
	"net"
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

func generateOTP() (string, error) {
	code, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", code.Int64()), nil
}

// sendEmail sends an OTP email using an SMTP client (supports implicit TLS on port 465, STARTTLS, and plain SMTP with optional auth). Falls back to plain text if template is missing.
func sendEmail(email, otp string) error {
	from := config.Cfg.MailConfig.From
	password := config.Cfg.MailConfig.Password
	smtpHost := config.Cfg.MailConfig.SMTPHost
	smtpPort := config.Cfg.MailConfig.SMTPPort

	if from == "" {
		from = "noreply@beast.local"
	}

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
		body.WriteString(fmt.Sprintf("Hello,\n\nYour OTP is: %s\nThis OTP will expire in 5 minutes.\n\nRegards,\nTeam", otp))
	}

	// Create email headers
	message := fmt.Sprintf("From: %s\r\n", from) +
		fmt.Sprintf("To: %s\r\n", email) +
		fmt.Sprintf("Subject: %s\r\n", subject) +
		"MIME-Version: 1.0\r\n"

	// Set Content-Type based on template availability
	if body.Len() > 0 && body.String()[0] == '<' {
		message += "Content-Type: text/html; charset=\"utf-8\"\r\n\r\n"
	} else {
		message += "Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n"
	}

	message += body.String()

	addr := net.JoinHostPort(smtpHost, smtpPort)
	tlsConfig := &tls.Config{ServerName: smtpHost, MinVersion: tls.VersionTLS12}
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect to SMTP: %w", err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}
	if smtpPort == "465" {
		conn = tls.Client(conn, tlsConfig)
	}
	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		return fmt.Errorf("initialize SMTP session: %w", err)
	}
	defer client.Close()
	if smtpPort != "465" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("start SMTP TLS: %w", err)
			}
		}
	}

	// Authenticate only when credentials are configured
	if password != "" {
		auth := smtp.PlainAuth("", from, password, smtpHost)
		if err := client.Auth(auth); err != nil {
			log.Println("SMTP authentication failed:", err)
			return err
		}
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

// @Summary Send a registration verification code
// @Tags auth
// @Accept x-www-form-urlencoded,mpfd
// @Produce json
// @Param email formData string true "Email address"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /auth/send-otp [post]
func sendOTPHandler(c *gin.Context) {
	if config.SkipAuthorization {
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "OTP verification skipped in no-auth mode",
		})
		return
	}
	email := c.PostForm("email")
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Email cannot be empty",
		})
		return
	}

	smtpHost := config.Cfg.MailConfig.SMTPHost
	smtpPort := config.Cfg.MailConfig.SMTPPort

	if smtpHost == "" || smtpPort == "" {
		log.Printf("WARNING: %s", "SMTP not configured")
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "SMTP not configured",
		})
		return
	}

	otp, err := generateOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{Error: "Failed to generate OTP"})
		return
	}
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

// @Summary Verify a registration code
// @Tags auth
// @Accept x-www-form-urlencoded,mpfd
// @Produce json
// @Param email formData string true "Email address"
// @Param otp formData string true "Six-digit email code"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 401 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /auth/verify-otp [post]
func verifyOTPHandler(c *gin.Context) {
	email := c.PostForm("email")
	otp := strings.TrimSpace(c.PostForm("otp"))
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" || otp == "" {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Email and OTP cannot be empty",
		})
		return
	}

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
				return
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

// @Summary Send a password recovery code
// @Tags auth
// @Accept x-www-form-urlencoded,mpfd
// @Produce json
// @Param email formData string true "Email address"
// @Success 200 {object} api.HTTPPlainResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /auth/send-otp-forget [post]
func sendOTPForForgetHandler(c *gin.Context) {
	if config.SkipAuthorization {
		c.JSON(http.StatusOK, HTTPPlainResp{
			Message: "OTP verification skipped in no-auth mode",
		})
		return
	}
	email := c.PostForm("email")
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Email cannot be empty",
		})
		return
	}

	smtpHost := config.Cfg.MailConfig.SMTPHost
	smtpPort := config.Cfg.MailConfig.SMTPPort

	if smtpHost == "" || smtpPort == "" {
		log.Printf("WARNING: %s", "SMTP not configured")
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "SMTP not configured",
		})
		return
	}

	otp, err := generateOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{Error: "Failed to generate OTP"})
		return
	}
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

// @Summary Verify a password recovery code
// @Tags auth
// @Accept x-www-form-urlencoded,mpfd
// @Produce json
// @Param email formData string true "Email address"
// @Param otp formData string true "Six-digit email code"
// @Success 200 {object} api.HTTPAuthorizeResp
// @Failure 400 {object} api.HTTPErrorResp
// @Failure 401 {object} api.HTTPErrorResp
// @Failure 500 {object} api.HTTPErrorResp
// @Router /auth/verify-otp-forget [post]
func verifyOTPForForgetHandler(c *gin.Context) {
	email := c.PostForm("email")
	otp := strings.TrimSpace(c.PostForm("otp"))
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" || otp == "" {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Email and OTP cannot be empty",
		})
		return
	}

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

		if otpEntry.Code == "" || otpEntry.Code != otp {
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

		otpEntry.Code = ""
		otpEntry.Expiry = time.Unix(0, 0)
		_ = database.CreateOTPEntry(&otpEntry)
	}
	userEntry, err := database.QueryFirstUserEntry("email", email)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: err.Error(),
		})
		return
	}
	if userEntry.ID == 0 {
		c.JSON(http.StatusNotFound, HTTPErrorResp{
			Error: "User not found",
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
