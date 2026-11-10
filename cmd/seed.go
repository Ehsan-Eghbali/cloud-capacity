package cmd

import "github.com/spf13/cobra"

func newSeedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Load demo data (regions, resources, capacities, customers, limits)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return execSQLFile("seeds/seed.sql")
		},
	}
}
