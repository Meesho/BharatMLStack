package app

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/api"
	"github.com/Meesho/BharatMLStack/resource-manager/pkg/metric"
)

var requestSeq uint64

const authHeaderName = "Authorization"

func authMiddleware(expectedToken string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providedToken := r.Header.Get(authHeaderName)
		bearerToken := "Bearer " + expectedToken
		isRawMatch := subtle.ConstantTimeCompare([]byte(providedToken), []byte(expectedToken)) == 1
		isBearerMatch := subtle.ConstantTimeCompare([]byte(providedToken), []byte(bearerToken)) == 1
		if !isRawMatch && !isBearerMatch {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), atomic.AddUint64(&requestSeq, 1))
		}
		w.Header().Set("X-Request-ID", requestID)

		ctx := api.WithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		next.ServeHTTP(rec, r)

		metric.ObserveAPIRequest(
			r.URL.Path,
			r.Method,
			rec.statusCode,
			time.Since(start),
		)
	})
}
