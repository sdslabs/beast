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
