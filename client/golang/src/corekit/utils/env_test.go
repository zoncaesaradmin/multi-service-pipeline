package utils

import "testing"

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_UTILS_ENV", "value")
	if got := GetEnv("TEST_UTILS_ENV", "fallback"); got != "value" {
		t.Fatalf("GetEnv() = %q, want value", got)
	}
	if got := GetEnv("TEST_UTILS_ENV_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("GetEnv() = %q, want fallback", got)
	}
}

func TestGetEnvInt(t *testing.T) {
	t.Setenv("TEST_UTILS_INT", "42")
	if got := GetEnvInt("TEST_UTILS_INT", 7); got != 42 {
		t.Fatalf("GetEnvInt() = %d, want 42", got)
	}

	t.Setenv("TEST_UTILS_INT_INVALID", "not-an-int")
	if got := GetEnvInt("TEST_UTILS_INT_INVALID", 7); got != 7 {
		t.Fatalf("GetEnvInt() = %d, want fallback 7", got)
	}
}

func TestGetEnvBool(t *testing.T) {
	t.Setenv("TEST_UTILS_BOOL_TRUE", "true")
	if got := GetEnvBool("TEST_UTILS_BOOL_TRUE", false); !got {
		t.Fatal("GetEnvBool() = false, want true")
	}

	t.Setenv("TEST_UTILS_BOOL_INVALID", "nope")
	if got := GetEnvBool("TEST_UTILS_BOOL_INVALID", true); !got {
		t.Fatal("GetEnvBool() should return fallback true for invalid values")
	}
}
