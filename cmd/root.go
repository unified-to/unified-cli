package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var apiKey string
var noValidate bool

var rootCmd = &cobra.Command{
	Use:   "unified",
	Short: "CLI tool for the Unified.to API",
	Long: `A command-line interface for accessing the Unified.to API.
Perform CRUD operations on any Unified.to object.

Run "unified objects" to browse the objects the API supports and the
operations and list filters available for each one.`,
	// Errors are printed once by Execute; the messages below name the command
	// to run for more detail, which is more use than a wall of usage text.
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	cmd, err := rootCmd.ExecuteC()
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "Error:", err)
	if cmd != nil {
		fmt.Fprintf(os.Stderr, "Run \"%s --help\" for usage.\n", cmd.CommandPath())
	}
	os.Exit(1)
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&apiKey, "api-key", "k", "", "API key (or set UNIFIED_API_KEY env var)")
	rootCmd.PersistentFlags().BoolVar(&noValidate, "no-validate", false, "Skip the object catalog check and send the request as-is")
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
