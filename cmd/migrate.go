package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Apply or roll back the database schema",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Create the schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			return execSQLFile("migrations/000001_init_schema.up.sql")
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "down",
		Short: "Drop the schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			return execSQLFile("migrations/000001_init_schema.down.sql")
		},
	})
	return cmd
}

// execSQLFile runs all statements of a SQL file in one implicit transaction.
func execSQLFile(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	app, err := newApplication(ctx)
	if err != nil {
		return err
	}
	defer app.pool.Close()

	if _, err := app.pool.Exec(ctx, string(content)); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
