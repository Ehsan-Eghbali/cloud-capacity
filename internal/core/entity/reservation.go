package entity

import (
	"time"

	"github.com/google/uuid"
)

type ReservationStatus string

const (
	ReservationStatusReserved  ReservationStatus = "RESERVED"
	ReservationStatusCommitted ReservationStatus = "COMMITTED"
	ReservationStatusCancelled ReservationStatus = "CANCELLED"
	ReservationStatusExpired   ReservationStatus = "EXPIRED"
)

func (s ReservationStatus) IsValid() bool {
	switch s {
	case ReservationStatusReserved, ReservationStatusCommitted, ReservationStatusCancelled, ReservationStatusExpired:
		return true
	}
	return false
}

type ReservationItem struct {
	ResourceTypeID string
	Amount         int64
}

type Reservation struct {
	ID             uuid.UUID
	CustomerID     uuid.UUID
	RegionID       string
	IdempotencyKey string
	Status         ReservationStatus
	TTLSeconds     int
	ExpiresAt      time.Time
	CommittedAt    *time.Time
	CancelledAt    *time.Time
	Metadata       map[string]any
	Items          []ReservationItem
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SameRequest compares region and items; both item lists are sorted.
func (r *Reservation) SameRequest(other *Reservation) bool {
	if r.RegionID != other.RegionID || len(r.Items) != len(other.Items) {
		return false
	}
	for i := range r.Items {
		if r.Items[i] != other.Items[i] {
			return false
		}
	}
	return true
}

type AuditAction string

const (
	AuditActionReserve AuditAction = "RESERVE"
	AuditActionCommit  AuditAction = "COMMIT"
	AuditActionCancel  AuditAction = "CANCEL"
	AuditActionExpire  AuditAction = "EXPIRE"
)
