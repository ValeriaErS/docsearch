package request

import (
	"context"
	"fmt"
	"time"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

func GetRequestID(ctx context.Context) string {  //  извлекает request_id из контекста
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return "unknown"
}

func GenerateRequestID() string {  //  создает новый request_id
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().UnixNano()%1000)
}