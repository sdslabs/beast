package api

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/core/database"
)

func generateOTP() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%06d", r.Intn(1000000)) // 6-digit OTP
}

func sendEmail(email, otp string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")

	to := []string{email}

	// Load email template
	tmpl, err := template.ParseFiles("email_template.html")
	if err != nil {
		log.Println("Error loading email template:", err)
		return err
	}

	// Replace placeholders in template
	var body bytes.Buffer
	err = tmpl.Execute(&body, struct{ OTP string }{OTP: otp})
	if err != nil {
		log.Println("Error executing template:", err)
		return err
	}

	// Email headers
	subject := "Subject: OTP Verification\r\n"
	mime := "MIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n"
	message := []byte(subject + mime + "\r\n" + body.String())

	auth := smtp.PlainAuth("", from, password, smtpHost)

	err = smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	if err != nil {
		fmt.Println("Failed to send email:", err)
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
