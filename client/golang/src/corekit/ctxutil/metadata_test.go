package ctxutil

import (
	"context"
	"testing"
)

func TestFieldsFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-123")
	ctx = WithRequestID(ctx, "req-456")
	ctx = WithCorrelationID(ctx, "corr-789")
	ctx = WithUserID(ctx, "user-abc")
	ctx = WithTenantID(ctx, "tenant-def")
	ctx = WithDebugEnabled(ctx, true)
	ctx = WithField(ctx, "workflowId", "wf-001")

	fields := FieldsFromContext(ctx)

	want := map[string]interface{}{
		"traceId":       "trace-123",
		"requestId":     "req-456",
		"correlationId": "corr-789",
		"userId":        "user-abc",
		"tenantId":      "tenant-def",
		"debugEnabled":  true,
		"workflowId":    "wf-001",
	}

	for key, expected := range want {
		if got := fields[key]; got != expected {
			t.Fatalf("field %q = %v, want %v", key, got, expected)
		}
	}
}

func TestFieldsFromContextLegacyKeys(t *testing.T) {
	ctx := context.WithValue(context.Background(), "traceID", "legacy-trace")
	ctx = context.WithValue(ctx, "request_id", "legacy-request")
	ctx = context.WithValue(ctx, "enableDebug", "true")

	fields := FieldsFromContext(ctx)

	if fields["traceId"] != "legacy-trace" {
		t.Fatalf("traceId = %v, want legacy-trace", fields["traceId"])
	}
	if fields["requestId"] != "legacy-request" {
		t.Fatalf("requestId = %v, want legacy-request", fields["requestId"])
	}
	if fields["debugEnabled"] != true {
		t.Fatalf("debugEnabled = %v, want true", fields["debugEnabled"])
	}
}

func TestWithFieldsMergesValues(t *testing.T) {
	ctx := context.Background()
	ctx = WithField(ctx, "workflowId", "wf-001")
	ctx = WithFields(ctx, map[string]interface{}{
		"region":     "us-east-1",
		"workflowId": "wf-002",
	})

	fields := GetFields(ctx)
	if fields["workflowId"] != "wf-002" {
		t.Fatalf("workflowId = %v, want wf-002", fields["workflowId"])
	}
	if fields["region"] != "us-east-1" {
		t.Fatalf("region = %v, want us-east-1", fields["region"])
	}
}
