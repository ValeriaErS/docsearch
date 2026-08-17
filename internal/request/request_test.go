package request

import (
	"context"
	"testing"
)

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	if id1 == "" {
		t.Error("GenerateRequestID() вернул пустую строку")
	}
	if id2 == "" {
		t.Error("GenerateRequestID() вернул пустую строку")
	}
	if id1 == id2 {
		t.Error("GenerateRequestID() вернул одинаковые ID")
	}
}

func TestGetRequestID(t *testing.T) {
	ctx := context.Background()
	id := GetRequestID(ctx)

	if id != "unknown" {
		t.Errorf("GetRequestID() без ID = %s, ожидалось 'unknown'", id)
	}

	expectedID := "test-123"
	ctxWithID := context.WithValue(ctx, RequestIDKey, expectedID)
	id = GetRequestID(ctxWithID)

	if id != expectedID {
		t.Errorf("GetRequestID() = %s, ожидалось %s", id, expectedID)
	}
}

func TestRequestIDKey(t *testing.T) {
	if RequestIDKey != "request_id" {
		t.Errorf("RequestIDKey = %s, ожидалось 'request_id'", RequestIDKey)
	}
}