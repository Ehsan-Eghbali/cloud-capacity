package entity

import "errors"

var (
	ErrInvalidRequest = errors.New("invalid request")

	ErrRegionNotFound      = errors.New("region not found")
	ErrCustomerNotFound    = errors.New("customer not found")
	ErrReservationNotFound = errors.New("reservation not found")

	ErrCustomerNotActive = errors.New("customer account is not active")
	ErrRegionUnavailable = errors.New("region is not accepting new reservations")

	ErrResourceNotAvailable   = errors.New("resource type is not available in this region")
	ErrResourceNotPermitted   = errors.New("resource type is not permitted for customer")
	ErrBelowMinimumAmount     = errors.New("requested amount is below the allowed minimum")
	ErrAboveMaximumPerRequest = errors.New("requested amount exceeds the maximum allowed per request")
	ErrQuotaExceeded          = errors.New("customer quota exceeded")
	ErrInsufficientCapacity   = errors.New("insufficient capacity")

	ErrIdempotencyKeyReused    = errors.New("idempotency key was already used for a different request")
	ErrInvalidReservationState = errors.New("reservation is not in a valid state for this action")
	ErrReservationExpired      = errors.New("reservation has expired")
)
