package http

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

const maxRequestBodyBytes = 1 << 20

type RouterConfig struct {
	CapacityHandler    *CapacityHandler
	ReservationHandler *ReservationHandler
	Ready              func(ctx context.Context) error
}

func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeData(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := cfg.Ready(ctx); err != nil {
			writeResponse(w, http.StatusServiceUnavailable, Response{Error: "database is not reachable"})
			return
		}
		writeData(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	mux.HandleFunc("GET /api/v1/regions", cfg.CapacityHandler.ListRegions)
	mux.HandleFunc("GET /api/v1/regions/{region_id}/capacity", cfg.CapacityHandler.GetRegionCapacity)
	mux.HandleFunc("GET /api/v1/resources", cfg.CapacityHandler.ListResourceTypes)

	mux.HandleFunc("POST /api/v1/reservations", cfg.ReservationHandler.CreateReservation)
	mux.HandleFunc("GET /api/v1/reservations", cfg.ReservationHandler.ListReservations)
	mux.HandleFunc("GET /api/v1/reservations/{id}", cfg.ReservationHandler.GetReservation)
	mux.HandleFunc("POST /api/v1/reservations/{id}/commit", cfg.ReservationHandler.CommitReservation)
	mux.HandleFunc("POST /api/v1/reservations/{id}/cancel", cfg.ReservationHandler.CancelReservation)

	return recoveryMiddleware(loggingMiddleware(corsMiddleware(bodyLimitMiddleware(mux))))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Reading past the limit fails, so an oversized body is answered with 400.
func bodyLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic in http handler", "error", rec, "stack", string(debug.Stack()))
				writeResponse(w, http.StatusInternalServerError, Response{Error: "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
