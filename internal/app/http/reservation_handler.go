package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud-capacity-allocator/internal/core/entity"
	"cloud-capacity-allocator/internal/core/port"

	"github.com/google/uuid"
)

type itemDTO struct {
	ResourceTypeID string `json:"resource_type_id"`
	Amount         int64  `json:"amount"`
}

type createReservationDTO struct {
	CustomerID     string         `json:"customer_id"`
	RegionID       string         `json:"region_id"`
	IdempotencyKey string         `json:"idempotency_key"`
	Metadata       map[string]any `json:"metadata"`
	Items          []itemDTO      `json:"items"`
}

type reservationDTO struct {
	ID             string         `json:"id"`
	CustomerID     string         `json:"customer_id"`
	RegionID       string         `json:"region_id"`
	IdempotencyKey string         `json:"idempotency_key"`
	Status         string         `json:"status"`
	TTLSeconds     int            `json:"ttl_seconds"`
	ExpiresAt      time.Time      `json:"expires_at"`
	CommittedAt    *time.Time     `json:"committed_at,omitempty"`
	CancelledAt    *time.Time     `json:"cancelled_at,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Items          []itemDTO      `json:"items"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func toReservationDTO(res *entity.Reservation) reservationDTO {
	items := make([]itemDTO, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, itemDTO{ResourceTypeID: item.ResourceTypeID, Amount: item.Amount})
	}
	return reservationDTO{
		ID:             res.ID.String(),
		CustomerID:     res.CustomerID.String(),
		RegionID:       res.RegionID,
		IdempotencyKey: res.IdempotencyKey,
		Status:         string(res.Status),
		TTLSeconds:     res.TTLSeconds,
		ExpiresAt:      res.ExpiresAt.UTC(),
		CommittedAt:    utc(res.CommittedAt),
		CancelledAt:    utc(res.CancelledAt),
		Metadata:       res.Metadata,
		Items:          items,
		CreatedAt:      res.CreatedAt.UTC(),
		UpdatedAt:      res.UpdatedAt.UTC(),
	}
}

func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

type ReservationHandler struct {
	service port.ReservationService
}

func NewReservationHandler(service port.ReservationService) *ReservationHandler {
	return &ReservationHandler{service: service}
}

// 201 for a new hold, 200 when the Idempotency-Key was already processed.
func (h *ReservationHandler) CreateReservation(w http.ResponseWriter, r *http.Request) {
	var body createReservationDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeBadRequest(w, "invalid request body")
		return
	}
	customerID, err := uuid.Parse(body.CustomerID)
	if err != nil {
		writeBadRequest(w, "valid customer_id is required")
		return
	}

	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		key = body.IdempotencyKey
	}
	items := make([]entity.ReservationItem, 0, len(body.Items))
	for _, item := range body.Items {
		items = append(items, entity.ReservationItem{ResourceTypeID: item.ResourceTypeID, Amount: item.Amount})
	}

	resp, err := h.service.CreateReservation(r.Context(), port.CreateReservationRequest{
		CustomerID:     customerID,
		RegionID:       body.RegionID,
		IdempotencyKey: key,
		Metadata:       body.Metadata,
		Items:          items,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	status := http.StatusOK
	if resp.Created {
		status = http.StatusCreated
	}
	writeData(w, status, toReservationDTO(resp.Reservation))
}

func (h *ReservationHandler) GetReservation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeBadRequest(w, "invalid reservation id")
		return
	}
	resp, err := h.service.GetReservation(r.Context(), port.GetReservationRequest{ReservationID: id})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, toReservationDTO(resp.Reservation))
}

func (h *ReservationHandler) CommitReservation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeBadRequest(w, "invalid reservation id")
		return
	}
	resp, err := h.service.CommitReservation(r.Context(), port.CommitReservationRequest{ReservationID: id})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, toReservationDTO(resp.Reservation))
}

func (h *ReservationHandler) CancelReservation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeBadRequest(w, "invalid reservation id")
		return
	}
	resp, err := h.service.CancelReservation(r.Context(), port.CancelReservationRequest{ReservationID: id})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeData(w, http.StatusOK, toReservationDTO(resp.Reservation))
}

func (h *ReservationHandler) ListReservations(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	customerID, err := uuid.Parse(query.Get("customer_id"))
	if err != nil {
		writeBadRequest(w, "valid customer_id is required")
		return
	}

	var status *entity.ReservationStatus
	if raw := query.Get("status"); raw != "" {
		s := entity.ReservationStatus(strings.ToUpper(raw))
		status = &s
	}
	page, _ := strconv.Atoi(query.Get("page"))
	pageSize, _ := strconv.Atoi(query.Get("page_size"))

	resp, err := h.service.ListReservations(r.Context(), port.ListReservationsRequest{
		CustomerID: customerID,
		Status:     status,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	items := make([]reservationDTO, 0, len(resp.Reservations))
	for i := range resp.Reservations {
		items = append(items, toReservationDTO(&resp.Reservations[i]))
	}
	writeData(w, http.StatusOK, items)
}
