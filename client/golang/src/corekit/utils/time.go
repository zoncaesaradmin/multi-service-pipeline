package utils

import "time"

const ISO8601Milli = "2006-01-02T15:04:05.000Z07:00"

// FormatUTCTimestampMilli formats a time in UTC using millisecond precision.
func FormatUTCTimestampMilli(t time.Time) string {
	return t.UTC().Format(ISO8601Milli)
}

// GetCurrentUTCTimestampMilli returns the current UTC timestamp with millisecond precision.
func GetCurrentUTCTimestampMilli() string {
	return FormatUTCTimestampMilli(time.Now())
}
