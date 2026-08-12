package server

import (
	"context"
	"net/http"
	"docsearch/internal/request"
)

func RequestIDMiddleware(next http.HandlerFunc) http.HandlerFunc {  //уникальный добавляет id каждому запросу
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = request.GenerateRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), request.RequestIDKey, requestID)
		next(w, r.WithContext(ctx))
	}
}