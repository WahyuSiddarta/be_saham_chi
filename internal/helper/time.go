package helper

import (
	"errors"
	"time"
)

func ParseDateString(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

// ParseDate parses a YYYY-MM-DD date, returning zero for an empty value and an error for an invalid date.
func ParseDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", value)
}

// ParseOptionalDate parses an optional YYYY-MM-DD date and names the field in validation errors.
func ParseOptionalDate(value string, field string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, errors.New("invalid " + field)
	}
	return parsed, nil
}

// ParseOptionalRFC3339 parses an optional RFC3339 timestamp and names the field in validation errors.
func ParseOptionalRFC3339(value string, field string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, errors.New("invalid " + field)
	}
	return parsed, nil
}

// FormatDate formats a YYYY-MM-DD date, returning an empty string for zero time.
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// FormatOptionalDate formats a YYYY-MM-DD date, returning nil for zero time.
func FormatOptionalDate(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	formatted := FormatDate(t)
	return &formatted
}

// FormatTime formats an RFC3339Nano timestamp, returning an empty string for zero time.
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}
