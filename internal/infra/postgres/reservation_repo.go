package postgres

import (
	"context"
	"errors"
	"fmt"

	"cloud-capacity-allocator/internal/core/entity"
	"cloud-capacity-allocator/internal/core/port"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	reservationColumns = `id, customer_id, region_id, idempotency_key, status, ttl_seconds,
		expires_at, committed_at, cancelled_at, metadata, created_at, updated_at`

	selectByIDSQL  = `SELECT ` + reservationColumns + ` FROM reservations WHERE id = $1`
	selectByKeySQL = `SELECT ` + reservationColumns + ` FROM reservations WHERE customer_id = $1 AND idempotency_key = $2`

	commitSQL = `UPDATE reservations SET status = 'COMMITTED', committed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'RESERVED' AND expires_at > NOW() RETURNING ` + reservationColumns
	cancelSQL = `UPDATE reservations SET status = 'CANCELLED', cancelled_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'RESERVED' RETURNING ` + reservationColumns
	expireSQL = `UPDATE reservations SET status = 'EXPIRED', updated_at = NOW()
		WHERE id = $1 AND status = 'RESERVED' AND expires_at <= NOW() RETURNING ` + reservationColumns

	allocateSQL = `UPDATE region_capacities
		SET reserved_capacity = reserved_capacity - $1, allocated_capacity = allocated_capacity + $1, updated_at = NOW()
		WHERE region_id = $2 AND resource_type_id = $3`
	releaseSQL = `UPDATE region_capacities
		SET reserved_capacity = reserved_capacity - $1, updated_at = NOW()
		WHERE region_id = $2 AND resource_type_id = $3`

	auditSQL = `INSERT INTO capacity_audit_logs (reservation_id, action, region_id) VALUES ($1, $2, $3)`
)

// dbtx is satisfied by both the pool and a transaction.
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// queries runs statements on the pool or inside a transaction.
type queries struct {
	db dbtx
}

// statement is a query that returns reservation rows.
type statement struct {
	sql  string
	args []any
}

// transition moves a RESERVED reservation to a final status. statusSQL
// matches only when the change is allowed.
type transition struct {
	id          uuid.UUID
	statusSQL   string
	capacitySQL string
	action      entity.AuditAction
	final       entity.ReservationStatus
}

type reservationRepository struct {
	pool *pgxpool.Pool
}

func NewReservationRepository(pool *pgxpool.Pool) port.ReservationRepository {
	return &reservationRepository{pool: pool}
}

func scanReservation(row pgx.CollectableRow) (entity.Reservation, error) {
	var r entity.Reservation
	err := row.Scan(&r.ID, &r.CustomerID, &r.RegionID, &r.IdempotencyKey, &r.Status, &r.TTLSeconds,
		&r.ExpiresAt, &r.CommittedAt, &r.CancelledAt, &r.Metadata, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

// list returns the reservations a statement selects, with their items. Items
// come back in insertion order, which is sorted by resource type.
func (q queries) list(ctx context.Context, st statement) (port.ListReservationsResponse, error) {
	rows, err := q.db.Query(ctx, st.sql, st.args...)
	if err != nil {
		return port.ListReservationsResponse{}, err
	}
	reservations, err := pgx.CollectRows(rows, scanReservation)
	if err != nil {
		return port.ListReservationsResponse{}, err
	}
	if len(reservations) == 0 {
		return port.ListReservationsResponse{}, nil
	}

	ids := make([]uuid.UUID, len(reservations))
	index := make(map[uuid.UUID]int, len(reservations))
	for i, res := range reservations {
		ids[i] = res.ID
		index[res.ID] = i
	}

	rows, err = q.db.Query(ctx, `
		SELECT reservation_id, resource_type_id, amount
		FROM reservation_items
		WHERE reservation_id = ANY($1)
		ORDER BY id`, ids)
	if err != nil {
		return port.ListReservationsResponse{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var item entity.ReservationItem
		if err := rows.Scan(&id, &item.ResourceTypeID, &item.Amount); err != nil {
			return port.ListReservationsResponse{}, err
		}
		res := &reservations[index[id]]
		res.Items = append(res.Items, item)
	}
	if err := rows.Err(); err != nil {
		return port.ListReservationsResponse{}, err
	}
	return port.ListReservationsResponse{Reservations: reservations}, nil
}

func (q queries) one(ctx context.Context, st statement) (*entity.Reservation, error) {
	found, err := q.list(ctx, st)
	if err != nil {
		return nil, err
	}
	if len(found.Reservations) == 0 {
		return nil, entity.ErrReservationNotFound
	}
	return &found.Reservations[0], nil
}

func (r *reservationRepository) GetReservation(ctx context.Context, req port.GetReservationRequest) (port.GetReservationResponse, error) {
	res, err := queries{db: r.pool}.one(ctx, statement{sql: selectByIDSQL, args: []any{req.ReservationID}})
	if err != nil {
		return port.GetReservationResponse{}, err
	}
	return port.GetReservationResponse{Reservation: res}, nil
}

func (r *reservationRepository) ListReservations(ctx context.Context, query port.ListReservationsQuery) (port.ListReservationsResponse, error) {
	return queries{db: r.pool}.list(ctx, statement{
		sql: `SELECT ` + reservationColumns + `
			FROM reservations
			WHERE customer_id = $1 AND ($2::text IS NULL OR status = $2::text)
			ORDER BY created_at DESC, id DESC
			LIMIT $3 OFFSET $4`,
		args: []any{query.CustomerID, query.Status, query.Limit, query.Offset},
	})
}

func (r *reservationRepository) CreateReservation(ctx context.Context, res *entity.Reservation) (port.CreateReservationResponse, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return port.CreateReservationResponse{}, err
	}
	defer tx.Rollback(ctx)
	q := queries{db: tx}

	// The row lock makes requests of one customer run one at a time, so the
	// idempotency and quota checks below cannot race.
	var customerStatus entity.CustomerStatus
	err = tx.QueryRow(ctx, `SELECT status FROM customers WHERE id = $1 FOR NO KEY UPDATE`, res.CustomerID).Scan(&customerStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return port.CreateReservationResponse{}, entity.ErrCustomerNotFound
	}
	if err != nil {
		return port.CreateReservationResponse{}, err
	}
	if customerStatus != entity.CustomerStatusActive {
		return port.CreateReservationResponse{}, entity.ErrCustomerNotActive
	}

	existing, err := q.one(ctx, statement{sql: selectByKeySQL, args: []any{res.CustomerID, res.IdempotencyKey}})
	if err == nil {
		if !existing.SameRequest(res) {
			return port.CreateReservationResponse{}, entity.ErrIdempotencyKeyReused
		}
		return port.CreateReservationResponse{Reservation: existing}, nil
	}
	if !errors.Is(err, entity.ErrReservationNotFound) {
		return port.CreateReservationResponse{}, err
	}

	var regionStatus entity.RegionStatus
	err = tx.QueryRow(ctx, `SELECT status FROM regions WHERE id = $1`, res.RegionID).Scan(&regionStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return port.CreateReservationResponse{}, entity.ErrRegionNotFound
	}
	if err != nil {
		return port.CreateReservationResponse{}, err
	}
	if regionStatus != entity.RegionStatusOnline {
		return port.CreateReservationResponse{}, fmt.Errorf("%w: region is %s", entity.ErrRegionUnavailable, regionStatus)
	}

	if err := q.checkLimits(ctx, res); err != nil {
		return port.CreateReservationResponse{}, err
	}
	if err := q.reserveCapacity(ctx, res); err != nil {
		return port.CreateReservationResponse{}, err
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO reservations (id, customer_id, region_id, idempotency_key, status, ttl_seconds, expires_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6::int, NOW() + make_interval(secs => $6::int), $7)
		RETURNING expires_at, created_at, updated_at`,
		res.ID, res.CustomerID, res.RegionID, res.IdempotencyKey, res.Status, res.TTLSeconds, res.Metadata,
	).Scan(&res.ExpiresAt, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return port.CreateReservationResponse{}, err
	}
	for _, item := range res.Items {
		_, err := tx.Exec(ctx,
			`INSERT INTO reservation_items (reservation_id, resource_type_id, amount) VALUES ($1, $2, $3)`,
			res.ID, item.ResourceTypeID, item.Amount)
		if err != nil {
			return port.CreateReservationResponse{}, err
		}
	}
	if _, err := tx.Exec(ctx, auditSQL, res.ID, entity.AuditActionReserve, res.RegionID); err != nil {
		return port.CreateReservationResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return port.CreateReservationResponse{}, err
	}
	return port.CreateReservationResponse{Reservation: res, Created: true}, nil
}

// A resource without a contract line has no quota.
func (q queries) checkLimits(ctx context.Context, res *entity.Reservation) error {
	for _, item := range res.Items {
		var minAmount int64
		err := q.db.QueryRow(ctx, `SELECT min_amount FROM resource_types WHERE id = $1`, item.ResourceTypeID).Scan(&minAmount)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %s", entity.ErrResourceNotAvailable, item.ResourceTypeID)
		}
		if err != nil {
			return err
		}
		if item.Amount < minAmount {
			return fmt.Errorf("%w: %s minimum is %d", entity.ErrBelowMinimumAmount, item.ResourceTypeID, minAmount)
		}

		limit := entity.CustomerLimit{ResourceTypeID: item.ResourceTypeID}
		err = q.db.QueryRow(ctx, `
			SELECT max_total_limit, min_per_request, max_per_request, is_enabled
			FROM customer_limits
			WHERE customer_id = $1 AND resource_type_id = $2`,
			res.CustomerID, item.ResourceTypeID,
		).Scan(&limit.MaxTotalLimit, &limit.MinPerRequest, &limit.MaxPerRequest, &limit.IsEnabled)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}

		var usage int64
		err = q.db.QueryRow(ctx, `
			SELECT COALESCE(SUM(ri.amount), 0)::bigint
			FROM reservation_items ri
			JOIN reservations r ON r.id = ri.reservation_id
			WHERE r.customer_id = $1 AND ri.resource_type_id = $2 AND r.status IN ('RESERVED', 'COMMITTED')`,
			res.CustomerID, item.ResourceTypeID,
		).Scan(&usage)
		if err != nil {
			return err
		}
		if err := limit.Validate(item.Amount, usage); err != nil {
			return err
		}
	}
	return nil
}

// The availability check and the update are one statement, so two requests
// cannot both take the same capacity.
func (q queries) reserveCapacity(ctx context.Context, res *entity.Reservation) error {
	for _, item := range res.Items {
		tag, err := q.db.Exec(ctx, `
			UPDATE region_capacities
			SET reserved_capacity = reserved_capacity + $1, updated_at = NOW()
			WHERE region_id = $2 AND resource_type_id = $3
			  AND total_capacity - allocated_capacity - reserved_capacity >= $1`,
			item.Amount, res.RegionID, item.ResourceTypeID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 1 {
			continue
		}

		var available int64
		err = q.db.QueryRow(ctx, `
			SELECT total_capacity - allocated_capacity - reserved_capacity
			FROM region_capacities
			WHERE region_id = $1 AND resource_type_id = $2`,
			res.RegionID, item.ResourceTypeID,
		).Scan(&available)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %s", entity.ErrResourceNotAvailable, item.ResourceTypeID)
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("%w: %s available %d, requested %d",
			entity.ErrInsufficientCapacity, item.ResourceTypeID, available, item.Amount)
	}
	return nil
}

// finish applies a transition and updates the region counters. It returns nil
// when the reservation is not in a status that allows the change.
func (r *reservationRepository) finish(ctx context.Context, t transition) (*entity.Reservation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	res, err := queries{db: tx}.one(ctx, statement{sql: t.statusSQL, args: []any{t.id}})
	if errors.Is(err, entity.ErrReservationNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for _, item := range res.Items {
		if _, err := tx.Exec(ctx, t.capacitySQL, item.Amount, res.RegionID, item.ResourceTypeID); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, auditSQL, res.ID, t.action, res.RegionID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return res, nil
}

// refused explains why a commit or cancel changed nothing.
func (r *reservationRepository) refused(ctx context.Context, t transition) (*entity.Reservation, error) {
	res, err := queries{db: r.pool}.one(ctx, statement{sql: selectByIDSQL, args: []any{t.id}})
	if err != nil {
		return nil, err
	}
	switch res.Status {
	case t.final:
		return res, nil
	case entity.ReservationStatusReserved, entity.ReservationStatusExpired:
		return nil, entity.ErrReservationExpired
	default:
		return nil, fmt.Errorf("%w: status is %s", entity.ErrInvalidReservationState, res.Status)
	}
}

func (r *reservationRepository) CommitReservation(ctx context.Context, req port.CommitReservationRequest) (port.CommitReservationResponse, error) {
	t := transition{
		id:          req.ReservationID,
		statusSQL:   commitSQL,
		capacitySQL: allocateSQL,
		action:      entity.AuditActionCommit,
		final:       entity.ReservationStatusCommitted,
	}
	res, err := r.finish(ctx, t)
	if err == nil && res == nil {
		res, err = r.refused(ctx, t)
	}
	if err != nil {
		return port.CommitReservationResponse{}, err
	}
	return port.CommitReservationResponse{Reservation: res}, nil
}

func (r *reservationRepository) CancelReservation(ctx context.Context, req port.CancelReservationRequest) (port.CancelReservationResponse, error) {
	t := transition{
		id:          req.ReservationID,
		statusSQL:   cancelSQL,
		capacitySQL: releaseSQL,
		action:      entity.AuditActionCancel,
		final:       entity.ReservationStatusCancelled,
	}
	res, err := r.finish(ctx, t)
	if err == nil && res == nil {
		res, err = r.refused(ctx, t)
	}
	if err != nil {
		return port.CancelReservationResponse{}, err
	}
	return port.CancelReservationResponse{Reservation: res}, nil
}

// Each hold is expired in its own transaction.
func (r *reservationRepository) ReapExpired(ctx context.Context, req port.ReapExpiredRequest) (port.ReapExpiredResponse, error) {
	var resp port.ReapExpiredResponse

	rows, err := r.pool.Query(ctx, `
		SELECT id FROM reservations
		WHERE status = 'RESERVED' AND expires_at <= NOW()
		ORDER BY expires_at
		LIMIT $1`, req.BatchSize)
	if err != nil {
		return resp, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return resp, err
	}

	for _, id := range ids {
		res, err := r.finish(ctx, transition{
			id:          id,
			statusSQL:   expireSQL,
			capacitySQL: releaseSQL,
			action:      entity.AuditActionExpire,
			final:       entity.ReservationStatusExpired,
		})
		if err != nil {
			return resp, err
		}
		if res != nil {
			resp.ReapedCount++
		}
	}
	return resp, nil
}
