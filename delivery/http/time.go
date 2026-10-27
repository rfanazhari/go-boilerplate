package httpapi

import (
	"fmt"
	"strings"
	"time"
)

const millisLayout = "2006-01-02T15:04:05.000Z07:00"

func parseRFC3339Millis(value string) (time.Time, error) {
	if err := rejectExtraPrecision(value); err != nil {
		return time.Time{}, err
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		parsed, err = time.Parse(millisLayout, value)
		if err != nil {
			return time.Time{}, err
		}
	}
	return parsed.UTC().Truncate(time.Millisecond), nil
}

func rejectExtraPrecision(value string) error {
	dot := strings.LastIndex(value, ".")
	if dot < 0 {
		return nil
	}
	rest := value[dot+1:]
	digits := 0
	for _, r := range rest {
		if r < '0' || r > '9' {
			break
		}
		digits++
	}
	if digits > 3 {
		return fmt.Errorf("timestamp fractional precision exceeds milliseconds")
	}
	return nil
}

func formatTime(value time.Time) string {
	return value.UTC().Truncate(time.Millisecond).Format(millisLayout)
}

func formatTimePtr(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatTime(*value)
}
