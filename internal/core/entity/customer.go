package entity

import "fmt"

type CustomerStatus string

const (
	CustomerStatusActive    CustomerStatus = "ACTIVE"
	CustomerStatusSuspended CustomerStatus = "SUSPENDED"
)

// CustomerLimit is the customer's contract for one resource type. The total
// limit counts usage across all regions.
type CustomerLimit struct {
	ResourceTypeID string
	MaxTotalLimit  int64
	MinPerRequest  int64
	MaxPerRequest  int64
	IsEnabled      bool
}

// usage is what the customer already holds (RESERVED + COMMITTED).
func (l CustomerLimit) Validate(amount, usage int64) error {
	if !l.IsEnabled {
		return fmt.Errorf("%w: %s", ErrResourceNotPermitted, l.ResourceTypeID)
	}
	if amount < l.MinPerRequest {
		return fmt.Errorf("%w: %s minimum per request is %d", ErrBelowMinimumAmount, l.ResourceTypeID, l.MinPerRequest)
	}
	if l.MaxPerRequest > 0 && amount > l.MaxPerRequest {
		return fmt.Errorf("%w: %s maximum per request is %d", ErrAboveMaximumPerRequest, l.ResourceTypeID, l.MaxPerRequest)
	}
	if usage+amount > l.MaxTotalLimit {
		return fmt.Errorf("%w: %s limit is %d, in use %d, requested %d",
			ErrQuotaExceeded, l.ResourceTypeID, l.MaxTotalLimit, usage, amount)
	}
	return nil
}
