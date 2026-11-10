package cmd

import (
	"log/slog"
	"os"

	"cloud-capacity-allocator/pkg/logger"

	"github.com/spf13/cobra"
)

var ConfigPath string

var RootCmd = &cobra.Command{
	Use:   "cloud-capacity-allocator",
	Short: "Cloud Resource Capacity Allocation & Quota Engine",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.SetupLogger()
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		slog.Error("command execution failed", "error", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&ConfigPath, "config", "c", "config.yaml", "path to yaml configuration file")

	RootCmd.AddCommand(newServerCmd())
	RootCmd.AddCommand(newMigrateCmd())
	RootCmd.AddCommand(newSeedCmd())
	RootCmd.AddCommand(newWorkerCmd())
}
