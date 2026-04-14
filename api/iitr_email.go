package api

import "strings"

const iitrEmailHost = "iitr.ac.in"

// isIitrEmailAddress returns true for @iitr.ac.in and subdomains such as @ece.iitr.ac.in.
func isIitrEmailAddress(email string) bool {
	email = strings.TrimSpace(strings.ToLower(email))
	at := strings.LastIndexByte(email, '@')
	if at <= 0 || at == len(email)-1 {
		return false
	}
	host := email[at+1:]
	if host == iitrEmailHost {
		return true
	}
	return strings.HasSuffix(host, "."+iitrEmailHost)
}
