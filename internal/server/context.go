package server

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

func RequestIDMiddleware(next http.HandlerFunc) http.HandlerFunc {  //  добавляет уникальный ID каждому запросу
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().UnixNano()%1000)
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		next(w, r.WithContext(ctx))
	}
}

func GetRequestID(ctx context.Context) string {  //  извлекает request_id из контекста
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return "unknown"
}