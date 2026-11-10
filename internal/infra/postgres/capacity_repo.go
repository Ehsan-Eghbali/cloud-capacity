package postgres

import (
	"context"
	"errors"

	"cloud-capacity-allocator/internal/core/entity"
	"cloud-capacity-allocator/internal/core/port"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type capacityRepository struct {
	pool *pgxpool.Pool
}

func NewCapacityRepository(pool *pgxpool.Pool) port.CapacityRepository {
	return &capacityRepository{pool: pool}
}

func scanRegion(row pgx.CollectableRow) (entity.Region, error) {
	var r entity.Region
	err := row.Scan(&r.ID, &r.Name, &r.Location, &r.Status, &r.Metadata)
	return r, err
}

func scanResourceType(row pgx.CollectableRow) (entity.ResourceType, error) {
	var t entity.ResourceType
	err := row.Scan(&t.ID, &t.Name, &t.Unit, &t.IsDivisible, &t.MinAmount)
	return t, err
}

func scanCapacity(row pgx.CollectableRow) (entity.RegionCapacity, error) {
	var c entity.RegionCapacity
	err := row.Scan(&c.RegionID, &c.ResourceTypeID, &c.TotalCapacity, &c.AllocatedCapacity, &c.ReservedCapacity, &c.UpdatedAt)
	return c, err
}

func (r *capacityRepository) ListRegions(ctx context.Context) (port.ListRegionsResponse, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, location, status, metadata FROM regions ORDER BY id`)
	if err != nil {
		return port.ListRegionsResponse{}, err
	}
	regions, err := pgx.CollectRows(rows, scanRegion)
	if err != nil {
		return port.ListRegionsResponse{}, err
	}
	return port.ListRegionsResponse{Regions: regions}, nil
}

func (r *capacityRepository) ListResourceTypes(ctx context.Context) (port.ListResourceTypesResponse, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, unit, is_divisible, min_amount FROM resource_types ORDER BY id`)
	if err != nil {
		return port.ListResourceTypesResponse{}, err
	}
	types, err := pgx.CollectRows(rows, scanResourceType)
	if err != nil {
		return port.ListResourceTypesResponse{}, err
	}
	return port.ListResourceTypesResponse{ResourceTypes: types}, nil
}

func (r *capacityRepository) GetRegionCapacity(ctx context.Context, req port.GetRegionCapacityRequest) (port.GetRegionCapacityResponse, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, location, status, metadata FROM regions WHERE id = $1`, req.RegionID)
	if err != nil {
		return port.GetRegionCapacityResponse{}, err
	}
	region, err := pgx.CollectOneRow(rows, scanRegion)
	if errors.Is(err, pgx.ErrNoRows) {
		return port.GetRegionCapacityResponse{}, entity.ErrRegionNotFound
	}
	if err != nil {
		return port.GetRegionCapacityResponse{}, err
	}

	rows, err = r.pool.Query(ctx, `
		SELECT region_id, resource_type_id, total_capacity, allocated_capacity, reserved_capacity, updated_at
		FROM region_capacities
		WHERE region_id = $1
		ORDER BY resource_type_id`, req.RegionID)
	if err != nil {
		return port.GetRegionCapacityResponse{}, err
	}
	capacities, err := pgx.CollectRows(rows, scanCapacity)
	if err != nil {
		return port.GetRegionCapacityResponse{}, err
	}
	return port.GetRegionCapacityResponse{Region: &region, Capacities: capacities}, nil
}
