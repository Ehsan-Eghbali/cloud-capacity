package port

import (
	"context"

	"cloud-capacity-allocator/internal/core/entity"

	"github.com/google/uuid"
)

type CreateReservationRequest struct {
	CustomerID     uuid.UUID
	RegionID       string
	IdempotencyKey string
	Metadata       map[string]any
	Items          []entity.ReservationItem
}

// Created is false when the idempotency key was already used and the stored
// reservation is returned.
type CreateReservationResponse struct {
	Reservation *entity.Reservation
	Created     bool
}

type GetReservationRequest struct {
	ReservationID uuid.UUID
}

type GetReservationResponse struct {
	Reservation *entity.Reservation
}

type CommitReservationRequest struct {
	ReservationID uuid.UUID
}

type CommitReservationResponse struct {
	Reservation *entity.Reservation
}

type CancelReservationRequest struct {
	ReservationID uuid.UUID
}

type CancelReservationResponse struct {
	Reservation *entity.Reservation
}

type ListReservationsRequest struct {
	CustomerID uuid.UUID
	Status     *entity.ReservationStatus
	Page       int
	PageSize   int
}

type ListReservationsResponse struct {
	Reservations []entity.Reservation
}

type ReapExpiredRequest struct {
	BatchSize int
}

type ReapExpiredResponse struct {
	ReapedCount int
}

type ReservationService interface {
	CreateReservation(ctx context.Context, req CreateReservationRequest) (CreateReservationResponse, error)
	GetReservation(ctx context.Context, req GetReservationRequest) (GetReservationResponse, error)
	CommitReservation(ctx context.Context, req CommitReservationRequest) (CommitReservationResponse, error)
	CancelReservation(ctx context.Context, req CancelReservationRequest) (CancelReservationResponse, error)
	ListReservations(ctx context.Context, req ListReservationsRequest) (ListReservationsResponse, error)
	ReapExpired(ctx context.Context, req ReapExpiredRequest) (ReapExpiredResponse, error)
}

type ListReservationsQuery struct {
	CustomerID uuid.UUID
	Status     *entity.ReservationStatus
	Limit      int
	Offset     int
}

// The repository reuses the service types wherever the data is the same.
type ReservationRepository interface {
	CreateReservation(ctx context.Context, res *entity.Reservation) (CreateReservationResponse, error)
	GetReservation(ctx context.Context, req GetReservationRequest) (GetReservationResponse, error)
	CommitReservation(ctx context.Context, req CommitReservationRequest) (CommitReservationResponse, error)
	CancelReservation(ctx context.Context, req CancelReservationRequest) (CancelReservationResponse, error)
	ListReservations(ctx context.Context, query ListReservationsQuery) (ListReservationsResponse, error)
	ReapExpired(ctx context.Context, req ReapExpiredRequest) (ReapExpiredResponse, error)
}
