package app

import "github.com/spf13/cobra"

var configPath string

var rootCmd = &cobra.Command{
	Use:   "ai-adp",
	Short: "AI Development Platform",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "configs/app.yaml", "config file path")
	rootCmd.AddCommand(serveCmd)
}
