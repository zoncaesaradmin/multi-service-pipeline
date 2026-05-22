package timeutil

import (
	"strings"
	"testing"
	"time"
)

func TestFormatUTCTimestampMilli(t *testing.T) {
	input := time.Date(2025, time.March, 4, 10, 11, 12, 345000000, time.FixedZone("IST", 5*60*60+30*60))
	got := FormatUTCTimestampMilli(input)
	want := "2025-03-04T04:41:12.345Z"
	if got != want {
		t.Fatalf("FormatUTCTimestampMilli() = %q, want %q", got, want)
	}
}

func TestNowUTCTimestampMilli(t *testing.T) {
	got := NowUTCTimestampMilli()
	if !strings.Contains(got, "T") {
		t.Fatalf("expected ISO-8601 style timestamp, got %q", got)
	}
	if _, err := time.Parse(ISO8601Milli, got); err != nil {
		t.Fatalf("timestamp %q did not parse with ISO8601Milli: %v", got, err)
	}
}
