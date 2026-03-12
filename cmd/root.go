package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var apiKey string

var rootCmd = &cobra.Command{
	Use:   "unified",
	Short: "CLI tool for the Unified.to API",
	Long:  "A command-line interface for accessing the Unified.to API.\nPerform CRUD operations on any Unified.to object.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&apiKey, "api-key", "k", "", "API key (or set UNIFIED_API_KEY env var)")
}

func getAPIKey() (string, error) {
	if apiKey != "" {
		return apiKey, nil
	}
	if envKey := os.Getenv("UNIFIED_API_KEY"); envKey != "" {
		return envKey, nil
	}
	return "", fmt.Errorf("API key required: use --api-key flag or set UNIFIED_API_KEY environment variable")
}
