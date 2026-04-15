package core

// BhawanInfo pairs a persisted code with the official hostel name (IIT Roorkee).
// Keep order: boys, girls, married, co-ed — matches public hostel groupings.
type BhawanInfo struct {
	Code string
	Name string
}

// AllHostels is the authoritative list; ValidBhawans is derived in init.
var AllHostels = []BhawanInfo{
	// Boys' hostels
	{"AB", "Azad Bhawan"},
	{"CB", "Cautley Bhawan"},
	{"GB", "Ganga Bhawan"},
	{"GOB", "Govind Bhawan"},
	{"JWB", "Jawahar Bhawan"},
	{"RB", "Rajendra Bhawan"},
	{"RKB", "Radhakrishnan Bhawan"},
	{"RJB", "Rajiv Bhawan"},
	{"RVB", "Ravindra Bhawan"},
	{"MB", "Malviya Bhawan"},
	// Girls' hostels
	{"HMB", "Himalaya Bhawan"},
	{"INB", "Indira Bhawan"},
	{"KB", "Kasturba Bhawan"},
	{"SB", "Sarojini Bhawan"},
	// Married hostels
	{"GPH", "G.P. Hostel"},
	{"MRC", "M.R. Chopra"},
	{"AZW", "Azad Wing"},
	{"DSB", "D.S. Barrack"},
	{"ANK", "A.N. Khosla House"},
	{"KIH", "K.I.H."},
	// Co-ed
	{"VVK", "Vivekananda Bhawan"},
	{"VK", "Vigyan Bhawan"},
}

// ValidBhawans is the allowlist for user registration and persistence.
var ValidBhawans []string

func init() {
	ValidBhawans = make([]string, len(AllHostels))
	for i, h := range AllHostels {
		ValidBhawans[i] = h.Code
	}
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
