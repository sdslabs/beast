package core

// ValidBhawans is the allowlist for user registration and persistence.
// Extend this slice when new hostels are added.
var ValidBhawans = []string{
	"RJB",
	"RKB",
	"RB",
	"SB",
	"KB",
	"VVK",
	"RVB",
}

// FresherBhawans lists hostels treated as first-year for the public "overall" leaderboard.
var FresherBhawans = []string{"RJB", "SB"}

// IsValidBhawan reports whether s is an allowed bhawan code.
func IsValidBhawan(s string) bool {
	for _, b := range ValidBhawans {
		if b == s {
			return true
		}
	}
	return false
}
