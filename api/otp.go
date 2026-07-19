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
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
	"github.com/sdslabs/beastv4/pkg/auth"
	"gorm.io/gorm"
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

func sendOTPHandler(c *gin.Context) {
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

func verifyOTPHandler(c *gin.Context) {
	email := c.PostForm("email")
	otp := strings.TrimSpace(c.PostForm("otp"))
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
	err = database.VerifyOTPEntry(email)

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

func sendOTPForForgetHandler(c *gin.Context) {
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

func verifyOTPForForgetHandler(c *gin.Context) {
	email := c.PostForm("email")
	otp := strings.TrimSpace(c.PostForm("otp"))
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

	now := time.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.CustomClaims{
		User: userEntry.Username,
		Role: userEntry.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    auth.ISSUER,
		},
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
