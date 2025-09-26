package validate

import "strings"

// Email Validation
func Email(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	i := strings.IndexByte(s, '@')
	if i <= 0 || i == len(s)-1 {
		return false
	}
	domain := s[i+1:]
	return strings.Count(s, "@") == 1 && strings.Contains(domain, ".")
}
