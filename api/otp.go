package api

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core"
	"github.com/sdslabs/beastv4/core/config"
	"github.com/sdslabs/beastv4/core/database"
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
		InsecureSkipVerify: false, // Set true only if SMTP server uses self-signed certs
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

func sendOTPHandler(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Invalid request",
		})
		return
	}

	email := req.Email
	otp := generateOTP()
	expiry := time.Now().Add(5 * time.Minute) // OTP expires in 5 minutes

	otpEntry := database.OTP{
		Email:  email,
		Code:   otp,
		Expiry: expiry,
	}

	err := database.CreateOTPEntry(&otpEntry)
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
	var req struct {
		Email string `json:"email"`
		OTP   string `json:"otp"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPErrorResp{
			Error: "Invalid request",
		})
		return
	}

	otpEntry, err := database.QueryOTPEntry(req.Email)

	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPErrorResp{
			Error: "Failed to verify OTP",
		})
		return
	}

	if otpEntry.Code != req.OTP {
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

	c.JSON(http.StatusOK, HTTPPlainResp{
		Message: "OTP verified successfully",
	})
}
