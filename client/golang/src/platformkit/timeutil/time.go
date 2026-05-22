package timeutil

import "time"

const ISO8601Milli = "2006-01-02T15:04:05.000Z07:00"

// FormatUTCTimestampMilli formats a time in UTC using millisecond precision.
func FormatUTCTimestampMilli(t time.Time) string {
	return t.UTC().Format(ISO8601Milli)
}

// NowUTCTimestampMilli returns the current UTC timestamp with millisecond precision.
func NowUTCTimestampMilli() string {
	return FormatUTCTimestampMilli(time.Now())
}
