package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"cloud-capacity-allocator/internal/core/entity"
	"cloud-capacity-allocator/internal/core/port"

	"github.com/google/uuid"
)

const (
	maxIdempotencyKeyLength = 128
	defaultTTLSeconds       = 600
	defaultPageSize         = 20
	maxPageSize             = 100
)

type ReservationOptions struct {
	// TTLSeconds is how long a hold stays valid.
	TTLSeconds int
}

type reservationService struct {
	repo port.ReservationRepository
	opts ReservationOptions
}

func NewReservationService(repo port.ReservationRepository, opts ReservationOptions) port.ReservationService {
	if opts.TTLSeconds <= 0 {
		opts.TTLSeconds = defaultTTLSeconds
	}
	return &reservationService{repo: repo, opts: opts}
}

func (s *reservationService) CreateReservation(ctx context.Context, req port.CreateReservationRequest) (port.CreateReservationResponse, error) {
	res, err := s.newReservation(req)
	if err != nil {
		return port.CreateReservationResponse{}, err
	}
	return s.repo.CreateReservation(ctx, res)
}

// newReservation validates the request and builds the hold to store.
func (s *reservationService) newReservation(req port.CreateReservationRequest) (*entity.Reservation, error) {
	regionID := strings.TrimSpace(req.RegionID)
	key := strings.TrimSpace(req.IdempotencyKey)

	switch {
	case req.CustomerID == uuid.Nil:
		return nil, fmt.Errorf("%w: customer_id is required", entity.ErrInvalidRequest)
	case regionID == "":
		return nil, fmt.Errorf("%w: region_id is required", entity.ErrInvalidRequest)
	case key == "" || len(key) > maxIdempotencyKeyLength:
		return nil, fmt.Errorf("%w: Idempotency-Key is required (max %d characters)", entity.ErrInvalidRequest, maxIdempotencyKeyLength)
	case len(req.Items) == 0:
		return nil, fmt.Errorf("%w: at least one item is required", entity.ErrInvalidRequest)
	}

	seen := make(map[string]bool, len(req.Items))
	items := make([]entity.ReservationItem, 0, len(req.Items))
	for _, item := range req.Items {
		item.ResourceTypeID = strings.TrimSpace(item.ResourceTypeID)
		switch {
		case item.ResourceTypeID == "" || item.Amount <= 0:
			return nil, fmt.Errorf("%w: each item needs a resource_type_id and a positive amount", entity.ErrInvalidRequest)
		case seen[item.ResourceTypeID]:
			return nil, fmt.Errorf("%w: resource_type_id %s is repeated", entity.ErrInvalidRequest, item.ResourceTypeID)
		}
		seen[item.ResourceTypeID] = true
		items = append(items, item)
	}
	// Every request locks capacity rows in this order, so two requests for
	// the same resources cannot deadlock.
	sort.Slice(items, func(i, j int) bool { return items[i].ResourceTypeID < items[j].ResourceTypeID })

	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	return &entity.Reservation{
		ID:             uuid.New(),
		CustomerID:     req.CustomerID,
		RegionID:       regionID,
		IdempotencyKey: key,
		Status:         entity.ReservationStatusReserved,
		TTLSeconds:     s.opts.TTLSeconds,
		Metadata:       metadata,
		Items:          items,
	}, nil
}

func (s *reservationService) GetReservation(ctx context.Context, req port.GetReservationRequest) (port.GetReservationResponse, error) {
	return s.repo.GetReservation(ctx, req)
}

func (s *reservationService) CommitReservation(ctx context.Context, req port.CommitReservationRequest) (port.CommitReservationResponse, error) {
	return s.repo.CommitReservation(ctx, req)
}

func (s *reservationService) CancelReservation(ctx context.Context, req port.CancelReservationRequest) (port.CancelReservationResponse, error) {
	return s.repo.CancelReservation(ctx, req)
}

func (s *reservationService) ListReservations(ctx context.Context, req port.ListReservationsRequest) (port.ListReservationsResponse, error) {
	if req.CustomerID == uuid.Nil {
		return port.ListReservationsResponse{}, fmt.Errorf("%w: customer_id is required", entity.ErrInvalidRequest)
	}
	if req.Status != nil && !req.Status.IsValid() {
		return port.ListReservationsResponse{}, fmt.Errorf("%w: unknown status %q", entity.ErrInvalidRequest, string(*req.Status))
	}

	page, pageSize := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return s.repo.ListReservations(ctx, port.ListReservationsQuery{
		CustomerID: req.CustomerID,
		Status:     req.Status,
		Limit:      pageSize,
		Offset:     (page - 1) * pageSize,
	})
}

func (s *reservationService) ReapExpired(ctx context.Context, req port.ReapExpiredRequest) (port.ReapExpiredResponse, error) {
	return s.repo.ReapExpired(ctx, req)
}
