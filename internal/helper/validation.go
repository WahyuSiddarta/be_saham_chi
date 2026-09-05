package helper

import "strings"

// CheckEmptyString checks if a string is empty or contains null-like values.
// Optimized with length-based short-circuiting and efficient comparisons.
func CheckEmptyString(param string) bool {
	// Fast path: check length first (most common case).
	switch len(param) {
	case 0:
		return true
	case 4:
		return param == "null"
	case 9:
		return param == "undefined"
	default:
		return false
	}
}

func CheckEmail(param string) bool {
	if CheckEmptyString(param) || strings.ContainsAny(param, " \t\r\n") {
		return false
	}

	at := strings.IndexByte(param, '@')
	if at <= 0 || at != strings.LastIndexByte(param, '@') {
		return false
	}

	domain := param[at+1:]
	if domain == "" || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	return strings.Contains(domain, ".")
}
