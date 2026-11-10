package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"cloud-capacity-allocator/internal/core/entity"
)

type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func writeResponse(w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func writeData(w http.ResponseWriter, status int, data any) {
	writeResponse(w, status, Response{Success: true, Data: data})
}

func writeBadRequest(w http.ResponseWriter, message string) {
	writeResponse(w, http.StatusBadRequest, Response{Error: message})
}

func statusForError(err error) int {
	switch {
	case errors.Is(err, entity.ErrInvalidRequest):
		return http.StatusBadRequest
	case errors.Is(err, entity.ErrCustomerNotActive):
		return http.StatusForbidden
	case errors.Is(err, entity.ErrCustomerNotFound),
		errors.Is(err, entity.ErrRegionNotFound),
		errors.Is(err, entity.ErrReservationNotFound):
		return http.StatusNotFound
	case errors.Is(err, entity.ErrInsufficientCapacity),
		errors.Is(err, entity.ErrRegionUnavailable),
		errors.Is(err, entity.ErrInvalidReservationState),
		errors.Is(err, entity.ErrReservationExpired):
		return http.StatusConflict
	case errors.Is(err, entity.ErrQuotaExceeded),
		errors.Is(err, entity.ErrResourceNotPermitted),
		errors.Is(err, entity.ErrResourceNotAvailable),
		errors.Is(err, entity.ErrBelowMinimumAmount),
		errors.Is(err, entity.ErrAboveMaximumPerRequest),
		errors.Is(err, entity.ErrIdempotencyKeyReused):
		return http.StatusUnprocessableEntity
	}
	return http.StatusInternalServerError
}

// Unexpected errors are logged and answered with a generic message.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status := statusForError(err)
	message := err.Error()
	if status == http.StatusInternalServerError {
		slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		message = "internal server error"
	}
	writeResponse(w, status, Response{Error: message})
}
