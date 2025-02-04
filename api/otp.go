package api

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
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

func sendEmail(email, otp string) error {
	from := config.Cfg.MailConfig.From
	password := config.Cfg.MailConfig.Password
	smtpHost := config.Cfg.MailConfig.SMTPHost
	smtpPort := config.Cfg.MailConfig.SMTPPort

	to := []string{email}

	emailTemplatePath := filepath.Join(
		core.BEAST_GLOBAL_DIR,
		core.BEAST_ASSETS_DIR,
		core.BEAST_EMAIL_TEMPLATE_DIR,
		"email_template.html",
	)
	// Load email template
	tmpl, err := template.ParseFiles(emailTemplatePath)
	if err != nil {
		log.Println("Error loading email template:", err)
		return err
	}

	// Replace placeholders in template
	var body bytes.Buffer
	err = tmpl.Execute(&body, struct{ OTP string }{OTP: otp})
	if err != nil {
		log.Println("Warning: Email template not found. Sending plain text email.")

		// Fallback to plain text email
		body.WriteString(fmt.Sprintf("Subject: OTP Verification\r\n\r\nYour OTP is: %s", otp))
	} else {
		// Replace placeholders in the template
		err = tmpl.Execute(&body, struct{ OTP string }{OTP: otp})
		if err != nil {
			log.Println("Error executing template:", err)
			return err
		}

		// Add email headers for HTML
		bodyHeader := "Subject: OTP Verification\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n"
		bodyString := bodyHeader + body.String()
		body.Reset()
		body.WriteString(bodyString)
	}

	// SMTP Authentication
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Send email
	err = smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, body.Bytes())
	if err != nil {
		log.Println("Failed to send email:", err)
		return err
	}

	fmt.Println("OTP email sent successfully to", email)
	return nil
}

func sendOTPHandler(email string) error {
	otp := generateOTP()
	expiry := time.Now().Add(5 * time.Minute) // OTP expires in 5 minutes

	otpEntry := database.OTP{
		Email:  email,
		Code:   otp,
		Expiry: expiry,
	}

	err := database.CreateOTPEntry(&otpEntry)
	if err != nil {
		return fmt.Errorf("failed to store OTP: %w", err)
	}

	// Send OTP to email
	err = sendEmail(email, otp)
	if err != nil {
		return fmt.Errorf("failed to send OTP: %w", err)
	}

	return nil
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

	err = database.DeleteOTPEntry(req.Email)

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
