package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/auth"
	"gorm.io/gorm"
)

const (
	otpPurposeRegistration  = "registration"
	otpPurposePasswordReset = "password_reset"
	otpLifetime             = 5 * time.Minute
)

func generateOTP() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

// sendEmail sends an OTP email using an SMTP client with TLS. Falls back to plain text if template is missing.
func sendEmail(email, otp string) error {
	from := config.Cfg.MailConfig.From
	password := config.Cfg.MailConfig.Password
	smtpHost := config.Cfg.MailConfig.SMTPHost
	smtpPort := config.Cfg.MailConfig.SMTPPort
	fromAddress, err := canonicalMailbox(from)
	if err != nil {
		return fmt.Errorf("invalid SMTP sender: %w", err)
	}
	recipientAddress, err := canonicalMailbox(email)
	if err != nil {
		return fmt.Errorf("invalid OTP recipient: %w", err)
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
	htmlBody := false
	_, err = os.Stat(emailTemplatePath)
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
		htmlBody = true
	} else {
		// Template does not exist, send plain text email
		log.Println("Template not found, sending plain text email.")
		body.WriteString(fmt.Sprintf("Hello,\n\nYour OTP is: %s\nThis OTP will expire in 10 minutes.\n\nRegards,\nTeam", otp))
	}

	// Create email headers
	message := fmt.Sprintf("From: %s\r\n", fromAddress) +
		fmt.Sprintf("To: %s\r\n", recipientAddress) +
		fmt.Sprintf("Subject: %s\r\n", subject) +
		"MIME-Version: 1.0\r\n"

	// Set Content-Type based on template availability
	if htmlBody {
		message += "Content-Type: text/html; charset=\"utf-8\"\r\n\r\n"
	} else {
		message += "Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n"
	}

	message += body.String()

	// Setup TLS connection
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: smtpHost,
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	rawConn, err := dialer.Dial("tcp", net.JoinHostPort(smtpHost, smtpPort))
	if err != nil {
		log.Println("Failed to connect to SMTP server:", err)
		return err
	}
	conn := tls.Client(rawConn, tlsConfig)
	if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		_ = conn.Close()
		return err
	}
	if err := conn.Handshake(); err != nil {
		_ = conn.Close()
		return fmt.Errorf("verify SMTP TLS connection: %w", err)
	}

	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		_ = conn.Close()
		log.Println("Failed to create SMTP client:", err)
		return err
	}
	defer client.Close()

	// Authenticate
	auth := smtp.PlainAuth("", fromAddress, password, smtpHost)
	if err := client.Auth(auth); err != nil {
		log.Println("SMTP authentication failed:", err)
		return err
	}

	// Set sender and recipient
	if err := client.Mail(fromAddress); err != nil {
		log.Println("Failed to set sender:", err)
		return err
	}

	if err := client.Rcpt(recipientAddress); err != nil {
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

	return nil
}

func canonicalMailbox(value string) (string, error) {
	if strings.ContainsAny(value, "\r\n") {
		return "", errors.New("mailbox contains a line break")
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value {
		return "", fmt.Errorf("mailbox must be a bare email address")
	}
	return parsed.Address, nil
}

func otpCodeHash(email, purpose, code string) []byte {
	mac := hmac.New(sha256.New, []byte(config.Cfg.JWTSecret))
	_, _ = mac.Write([]byte(purpose + "\x00" + email + "\x00" + code))
	return mac.Sum(nil)
}

func requestedOTPEmail(c *gin.Context) (string, error) {
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	canonical, err := canonicalMailbox(email)
	if err != nil {
		return "", err
	}
	return canonical, nil
}

func issueOTP(c *gin.Context, purpose string, existingUserRequired bool) {
	email, err := requestedOTPEmail(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{Error: "A valid email address is required"})
		return
	}
	mailConfig := config.Cfg.MailConfig
	if mailConfig.From == "" || mailConfig.Password == "" || mailConfig.SMTPHost == "" || mailConfig.SMTPPort == "" {
		c.JSON(http.StatusServiceUnavailable, HTTPErrorResp{Error: "SMTP not configured"})
		return
	}
	eligible := true
	_, userErr := database.QueryFirstUserEntry("email", email)
	if userErr != nil && !errors.Is(userErr, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{Error: "Failed to send OTP"})
		return
	}
	if existingUserRequired {
		eligible = userErr == nil
	} else if userErr == nil {
		c.JSON(http.StatusOK, HTTPPlainResp{Message: "If the request is eligible, an OTP has been sent"})
		return
	}
	code, err := generateOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{Error: "Failed to generate OTP"})
		return
	}
	now := time.Now()
	if err := database.IssueOTP(email, purpose, otpCodeHash(email, purpose, code), now, now.Add(otpLifetime)); err != nil {
		if errors.Is(err, database.ErrOTPRateLimited) {
			c.Header("Retry-After", "60")
			c.JSON(http.StatusTooManyRequests, HTTPErrorResp{Error: "OTP requested too recently"})
			return
		}
		log.Println("Failed to store OTP:", err)
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{Error: "Failed to store OTP"})
		return
	}
	if !eligible {
		c.JSON(http.StatusOK, HTTPPlainResp{Message: "If the request is eligible, an OTP has been sent"})
		return
	}
	if err := sendEmail(email, code); err != nil {
		_ = database.DeleteOTPEntry(email)
		log.Println("Failed to send OTP:", err)
		c.JSON(http.StatusBadGateway, HTTPErrorResp{Error: "Failed to send OTP"})
		return
	}
	c.JSON(http.StatusOK, HTTPPlainResp{Message: "If the request is eligible, an OTP has been sent"})
}

func verifyRequestedOTP(c *gin.Context, purpose string) (string, bool) {
	email, err := requestedOTPEmail(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{Error: "A valid email address is required"})
		return "", false
	}
	code := strings.TrimSpace(c.PostForm("otp"))
	if len(code) != 6 || strings.IndexFunc(code, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		c.JSON(http.StatusUnauthorized, HTTPErrorResp{Error: "Invalid OTP"})
		return "", false
	}
	err = database.VerifyOTPCode(email, purpose, otpCodeHash(email, purpose, code), time.Now())
	if err != nil {
		switch {
		case errors.Is(err, database.ErrOTPAttempts):
			c.JSON(http.StatusTooManyRequests, HTTPErrorResp{Error: "Too many OTP attempts"})
		case errors.Is(err, database.ErrOTPInvalid), errors.Is(err, database.ErrOTPExpired), errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusUnauthorized, HTTPErrorResp{Error: "Invalid or expired OTP"})
		default:
			log.Println("Failed to verify OTP:", err)
			c.JSON(http.StatusInternalServerError, HTTPErrorResp{Error: "Failed to verify OTP"})
		}
		return "", false
	}
	return email, true
}

func sendOTPHandler(c *gin.Context) {
	issueOTP(c, otpPurposeRegistration, false)
}

func verifyOTPHandler(c *gin.Context) {
	if _, ok := verifyRequestedOTP(c, otpPurposeRegistration); !ok {
		return
	}
	c.JSON(http.StatusOK, HTTPPlainResp{Message: "OTP verified successfully"})
}

func sendOTPForForgetHandler(c *gin.Context) {
	issueOTP(c, otpPurposePasswordReset, true)
}

func verifyOTPForForgetHandler(c *gin.Context) {
	email, ok := verifyRequestedOTP(c, otpPurposePasswordReset)
	if !ok {
		return
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

	tempToken, err := auth.GeneratePasswordResetJWT(userEntry.Username, userEntry.Role, 5*time.Minute)

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
