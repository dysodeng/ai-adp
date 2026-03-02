package app

import (
	"github.com/spf13/cobra"
	"github.com/dysodeng/ai-adp/internal/di"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := di.InitApp(configPath)
		if err != nil {
			return err
		}
		return app.Run()
	},
}
