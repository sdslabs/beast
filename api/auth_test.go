package api

import "testing"

func TestCredentialValidation(t *testing.T) {
	for _, password := range []string{"short", "            ", string(make([]byte, 129))} {
		if err := validatePassword(password); err == nil {
			t.Fatalf("expected password %q to be rejected", password)
		}
	}
	if err := validatePassword("correct horse battery staple"); err != nil {
		t.Fatal(err)
	}
	for _, username := range []string{"ab", "UPPER", "../escape", "space name"} {
		if contestantUsernamePattern.MatchString(username) {
			t.Fatalf("expected username %q to be rejected", username)
		}
	}
}
