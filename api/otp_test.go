package api

import (
	"bytes"
	"regexp"
	"testing"

	"github.com/sdslabs/beastv4/core/config"
)

func TestGenerateOTPProducesSixDigits(t *testing.T) {
	otp, err := generateOTP()
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9]{6}$`).MatchString(otp) {
		t.Fatalf("unexpected OTP format %q", otp)
	}
}

func TestOTPHashBindsPurposeAndEmail(t *testing.T) {
	previous := config.Cfg
	config.Cfg = &config.BeastConfig{JWTSecret: "01234567890123456789012345678901"}
	t.Cleanup(func() { config.Cfg = previous })

	registration := otpCodeHash("user@example.com", otpPurposeRegistration, "123456")
	reset := otpCodeHash("user@example.com", otpPurposePasswordReset, "123456")
	otherUser := otpCodeHash("other@example.com", otpPurposeRegistration, "123456")
	if bytes.Equal(registration, reset) || bytes.Equal(registration, otherUser) {
		t.Fatal("OTP hash was not bound to purpose and email")
	}
}

func TestCanonicalMailboxRejectsHeaderInjection(t *testing.T) {
	for _, value := range []string{"Name <user@example.com>", "user@example.com\r\nBcc: victim@example.com", "not-an-email"} {
		if _, err := canonicalMailbox(value); err == nil {
			t.Fatalf("expected mailbox %q to be rejected", value)
		}
	}
	if got, err := canonicalMailbox("user@example.com"); err != nil || got != "user@example.com" {
		t.Fatalf("expected canonical mailbox, got %q, %v", got, err)
	}
}
