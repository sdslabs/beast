package api

import (
	"regexp"
	"testing"
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
