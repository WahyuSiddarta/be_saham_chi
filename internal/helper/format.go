package helper

import "time"

func DateOrNil(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func NumberOrNil(value float64) any {
	if value == 0 {
		return nil
	}
	return value
}

func EmptyStringOrNil(value string) any {
	if value == "" {
		return nil
	}
	return value
}
