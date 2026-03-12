package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/unified-to/unified-cli/internal/api"
	"github.com/unified-to/unified-cli/internal/parser"
)

var createData string

var createCmd = &cobra.Command{
	Use:   "create <connection_id> <object>",
	Short: "Create a new object",
	Long:  "Create a new object via the Unified.to API.\nExample: unified create abc123 crm_contact -d '{\"name\":\"John\"}'",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := getAPIKey()
		if err != nil {
			return err
		}

		if createData == "" {
			return fmt.Errorf("--data is required for create")
		}

		data, err := readData(createData)
		if err != nil {
			return err
		}

		connectionID := args[0]
		category, object, err := parser.SplitObject(args[1])
		if err != nil {
			return err
		}

		client := api.NewClient(key, "")
		body, statusCode, err := client.Do("POST", category, connectionID, object, "", data, nil)
		if err != nil {
			return err
		}

		if statusCode >= 400 {
			fmt.Fprintln(os.Stderr, string(body))
			os.Exit(1)
		}

		fmt.Print(string(body))
		return nil
	},
}

// readData returns the JSON data from the flag value.
// If the value starts with @, it reads the file at that path.
func readData(value string) ([]byte, error) {
	if strings.HasPrefix(value, "@") {
		filePath := strings.TrimPrefix(value, "@")
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read data file %q: %w", filePath, err)
		}
		return data, nil
	}
	return []byte(value), nil
}

func init() {
	createCmd.Flags().StringVarP(&createData, "data", "d", "", "JSON data (inline or @file.json) (required)")
	createCmd.MarkFlagRequired("data")
	rootCmd.AddCommand(createCmd)
}
