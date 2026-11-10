package cmd

import (
	"context"

	"cloud-capacity-allocator/internal/app/worker"
	"cloud-capacity-allocator/internal/config"
	"cloud-capacity-allocator/internal/core/port"
	"cloud-capacity-allocator/internal/core/service"
	"cloud-capacity-allocator/internal/infra/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

type application struct {
	cfg          *config.Config
	pool         *pgxpool.Pool
	capacity     port.CapacityService
	reservations port.ReservationService
}

func newApplication(ctx context.Context) (*application, error) {
	cfg, err := config.Load(ConfigPath)
	if err != nil {
		return nil, err
	}
	pool, err := postgres.NewConnectionPool(ctx, cfg.Database.DSN())
	if err != nil {
		return nil, err
	}
	return &application{
		cfg:      cfg,
		pool:     pool,
		capacity: service.NewCapacityService(postgres.NewCapacityRepository(pool)),
		reservations: service.NewReservationService(postgres.NewReservationRepository(pool), service.ReservationOptions{
			TTLSeconds: cfg.Reservation.TTLSeconds,
		}),
	}, nil
}

func (a *application) newSweeper() *worker.Sweeper {
	return worker.NewSweeper(a.reservations, worker.SweeperConfig{
		Interval:  a.cfg.Worker.SweeperInterval,
		BatchSize: a.cfg.Worker.SweeperBatchSize,
	})
}
