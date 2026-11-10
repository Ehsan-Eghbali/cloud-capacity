package port

import (
	"context"

	"cloud-capacity-allocator/internal/core/entity"
)

type ListRegionsResponse struct {
	Regions []entity.Region
}

type ListResourceTypesResponse struct {
	ResourceTypes []entity.ResourceType
}

type GetRegionCapacityRequest struct {
	RegionID string
}

type GetRegionCapacityResponse struct {
	Region     *entity.Region
	Capacities []entity.RegionCapacity
}

type CapacityService interface {
	ListRegions(ctx context.Context) (ListRegionsResponse, error)
	ListResourceTypes(ctx context.Context) (ListResourceTypesResponse, error)
	GetRegionCapacity(ctx context.Context, req GetRegionCapacityRequest) (GetRegionCapacityResponse, error)
}

type CapacityRepository interface {
	ListRegions(ctx context.Context) (ListRegionsResponse, error)
	ListResourceTypes(ctx context.Context) (ListResourceTypesResponse, error)
	GetRegionCapacity(ctx context.Context, req GetRegionCapacityRequest) (GetRegionCapacityResponse, error)
}
