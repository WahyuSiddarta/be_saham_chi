package service

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
)

var errMalformedDividendHistory = errors.New("malformed dividend history")

func findMetricByKeys(value any, keys ...string) (float64, bool) {
	metrics, ok := value.(map[string]any)
	if !ok {
		return 0, false
	}
	for _, key := range keys {
		metric, ok := metrics[key].(map[string]any)
		if !ok {
			continue
		}
		raw, ok := metric["value"].(string)
		if !ok {
			continue
		}
		parsed, err := parseNumber(raw)
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func dividendPerShareTTM(history []any, asOf time.Time) (float64, error) {
	cutoff := asOf.AddDate(-1, 0, 0)
	total := 0.0
	for _, raw := range history {
		item, ok := raw.(map[string]any)
		if !ok {
			return 0, errMalformedDividendHistory
		}
		dateRaw, ok := item["exDate"].(string)
		if !ok {
			return 0, errMalformedDividendHistory
		}
		date, err := time.Parse("02 Jan 06", dateRaw)
		if err != nil {
			return 0, errMalformedDividendHistory
		}
		if date.Before(cutoff) || date.After(asOf) {
			continue
		}
		dividend, ok := item["dividend"].(string)
		if !ok {
			return 0, errMalformedDividendHistory
		}
		amount, err := parseNumber(dividend)
		if err != nil || amount < 0 {
			return 0, errMalformedDividendHistory
		}
		total += amount
	}
	return total, nil
}

func parseNumber(raw string) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(strings.NewReplacer(",", "", "%", "", "Rp", "", "IDR", "").Replace(raw)), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, errors.New("invalid number")
	}
	return value, nil
}
