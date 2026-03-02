package app

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/migration"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		db, err := persistence.NewDB(cfg)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		if err := migration.AutoMigrate(db); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		fmt.Println("Migration completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
