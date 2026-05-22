package envutil

import "testing"

func TestGet(t *testing.T) {
	t.Setenv("TEST_UTILS_ENV", "value")
	if got := Get("TEST_UTILS_ENV", "fallback"); got != "value" {
		t.Fatalf("Get() = %q, want value", got)
	}
	if got := Get("TEST_UTILS_ENV_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("Get() = %q, want fallback", got)
	}
}

func TestGetInt(t *testing.T) {
	t.Setenv("TEST_UTILS_INT", "42")
	if got := GetInt("TEST_UTILS_INT", 7); got != 42 {
		t.Fatalf("GetInt() = %d, want 42", got)
	}

	t.Setenv("TEST_UTILS_INT_INVALID", "not-an-int")
	if got := GetInt("TEST_UTILS_INT_INVALID", 7); got != 7 {
		t.Fatalf("GetInt() = %d, want fallback 7", got)
	}
}

func TestGetBool(t *testing.T) {
	t.Setenv("TEST_UTILS_BOOL_TRUE", "true")
	if got := GetBool("TEST_UTILS_BOOL_TRUE", false); !got {
		t.Fatal("GetBool() = false, want true")
	}

	t.Setenv("TEST_UTILS_BOOL_INVALID", "nope")
	if got := GetBool("TEST_UTILS_BOOL_INVALID", true); !got {
		t.Fatal("GetBool() should return fallback true for invalid values")
	}
}
