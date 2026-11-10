package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	httpApp "cloud-capacity-allocator/internal/app/http"

	"github.com/spf13/cobra"
)

func newServerCmd() *cobra.Command {
	var withWorker bool

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the HTTP API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			app, err := newApplication(ctx)
			if err != nil {
				return err
			}
			defer app.pool.Close()

			var wg sync.WaitGroup
			if withWorker {
				wg.Add(1)
				go func() {
					defer wg.Done()
					app.newSweeper().Start(ctx)
				}()
			}

			srv := &http.Server{
				Addr: fmt.Sprintf("%s:%d", app.cfg.Server.Host, app.cfg.Server.Port),
				Handler: httpApp.NewRouter(httpApp.RouterConfig{
					CapacityHandler:    httpApp.NewCapacityHandler(app.capacity),
					ReservationHandler: httpApp.NewReservationHandler(app.reservations),
					Ready:              app.pool.Ping,
				}),
				ReadTimeout:  app.cfg.Server.ReadTimeout,
				WriteTimeout: app.cfg.Server.WriteTimeout,
				IdleTimeout:  app.cfg.Server.IdleTimeout,
			}

			serveErr := make(chan error, 1)
			go func() {
				slog.Info("server listening", "address", srv.Addr)
				serveErr <- srv.ListenAndServe()
			}()

			select {
			case <-ctx.Done():
				slog.Info("shutdown signal received")
			case err := <-serveErr:
				return err
			}

			// Finish in-flight requests, then wait for the sweeper.
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("server shutdown failed", "error", err)
			}
			wg.Wait()
			slog.Info("shutdown complete")
			return nil
		},
	}

	cmd.Flags().BoolVar(&withWorker, "with-worker", true, "run the expiry sweeper inside the server process")
	return cmd
}
