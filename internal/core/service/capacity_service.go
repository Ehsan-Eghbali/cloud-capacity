package service

import (
	"context"

	"cloud-capacity-allocator/internal/core/port"
)

type capacityService struct {
	repo port.CapacityRepository
}

func NewCapacityService(repo port.CapacityRepository) port.CapacityService {
	return &capacityService{repo: repo}
}

func (s *capacityService) ListRegions(ctx context.Context) (port.ListRegionsResponse, error) {
	return s.repo.ListRegions(ctx)
}

func (s *capacityService) ListResourceTypes(ctx context.Context) (port.ListResourceTypesResponse, error) {
	return s.repo.ListResourceTypes(ctx)
}

func (s *capacityService) GetRegionCapacity(ctx context.Context, req port.GetRegionCapacityRequest) (port.GetRegionCapacityResponse, error) {
	return s.repo.GetRegionCapacity(ctx, req)
}
