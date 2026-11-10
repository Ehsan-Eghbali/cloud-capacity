package worker

import (
	"context"
	"log/slog"
	"time"

	"cloud-capacity-allocator/internal/core/port"
)

type SweeperConfig struct {
	Interval  time.Duration
	BatchSize int
}

// Sweeper releases holds that were not committed or cancelled before their deadline.
type Sweeper struct {
	service port.ReservationService
	cfg     SweeperConfig
}

func NewSweeper(service port.ReservationService, cfg SweeperConfig) *Sweeper {
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	return &Sweeper{service: service, cfg: cfg}
}

func (s *Sweeper) Start(ctx context.Context) {
	slog.Info("sweeper started", "interval", s.cfg.Interval.String(), "batch_size", s.cfg.BatchSize)
	ticker := time.NewTicker(s.cfg.Interval)
	defer ticker.Stop()

	for {
		s.sweep(ctx)
		select {
		case <-ctx.Done():
			slog.Info("sweeper stopped")
			return
		case <-ticker.C:
		}
	}
}

func (s *Sweeper) sweep(ctx context.Context) {
	for ctx.Err() == nil {
		resp, err := s.service.ReapExpired(ctx, port.ReapExpiredRequest{BatchSize: s.cfg.BatchSize})
		if err != nil {
			slog.Error("sweep failed", "error", err)
			return
		}
		if resp.ReapedCount > 0 {
			slog.Info("expired reservations released", "count", resp.ReapedCount)
		}
		if resp.ReapedCount < s.cfg.BatchSize {
			return
		}
	}
}
