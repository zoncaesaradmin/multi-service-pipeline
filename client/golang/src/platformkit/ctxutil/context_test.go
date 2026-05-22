package ctxutil

import (
	"context"
	"testing"
)

func TestTypedHelpers(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-123")
	ctx = WithRequestID(ctx, "req-456")
	ctx = WithCorrelationID(ctx, "corr-789")
	ctx = WithUserID(ctx, "user-abc")
	ctx = WithTenantID(ctx, "tenant-def")
	ctx = WithDebugEnabled(ctx, true)

	if got, ok := GetTraceID(ctx); !ok || got != "trace-123" {
		t.Fatalf("traceId = %q ok=%v, want trace-123 true", got, ok)
	}
	if got, ok := GetRequestID(ctx); !ok || got != "req-456" {
		t.Fatalf("requestId = %q ok=%v, want req-456 true", got, ok)
	}
	if got, ok := GetCorrelationID(ctx); !ok || got != "corr-789" {
		t.Fatalf("correlationId = %q ok=%v, want corr-789 true", got, ok)
	}
	if got, ok := GetUserID(ctx); !ok || got != "user-abc" {
		t.Fatalf("userId = %q ok=%v, want user-abc true", got, ok)
	}
	if got, ok := GetTenantID(ctx); !ok || got != "tenant-def" {
		t.Fatalf("tenantId = %q ok=%v, want tenant-def true", got, ok)
	}
	if got, ok := GetDebugEnabled(ctx); !ok || !got {
		t.Fatalf("debugEnabled = %v ok=%v, want true true", got, ok)
	}
}

func TestTypedHelpersLegacyKeys(t *testing.T) {
	ctx := context.WithValue(context.Background(), "traceID", "legacy-trace")
	ctx = context.WithValue(ctx, "request_id", "legacy-request")
	ctx = context.WithValue(ctx, "correlationID", "legacy-correlation")
	ctx = context.WithValue(ctx, "userID", "legacy-user")
	ctx = context.WithValue(ctx, "tenant_id", "legacy-tenant")
	ctx = context.WithValue(ctx, "enableDebug", "true")

	if got, ok := GetTraceID(ctx); !ok || got != "legacy-trace" {
		t.Fatalf("traceId = %q ok=%v, want legacy-trace true", got, ok)
	}
	if got, ok := GetRequestID(ctx); !ok || got != "legacy-request" {
		t.Fatalf("requestId = %q ok=%v, want legacy-request true", got, ok)
	}
	if got, ok := GetCorrelationID(ctx); !ok || got != "legacy-correlation" {
		t.Fatalf("correlationId = %q ok=%v, want legacy-correlation true", got, ok)
	}
	if got, ok := GetUserID(ctx); !ok || got != "legacy-user" {
		t.Fatalf("userId = %q ok=%v, want legacy-user true", got, ok)
	}
	if got, ok := GetTenantID(ctx); !ok || got != "legacy-tenant" {
		t.Fatalf("tenantId = %q ok=%v, want legacy-tenant true", got, ok)
	}
	if got, ok := GetDebugEnabled(ctx); !ok || !got {
		t.Fatalf("debugEnabled = %v ok=%v, want true true", got, ok)
	}
}
