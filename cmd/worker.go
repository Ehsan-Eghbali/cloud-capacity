package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func newWorkerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "worker",
		Short: "Run only the expiry sweeper",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			app, err := newApplication(ctx)
			if err != nil {
				return err
			}
			defer app.pool.Close()

			app.newSweeper().Start(ctx)
			return nil
		},
	}
}
