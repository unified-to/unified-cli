package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/unified-to/unified-cli/internal/api"
	"github.com/unified-to/unified-cli/internal/parser"
)

var updateID string
var updateData string

var updateCmd = &cobra.Command{
	Use:   "update <connection_id> <object>",
	Short: "Update an existing object (partial update via PATCH)",
	Long:  "Update an existing object via the Unified.to API using PATCH.\nExample: unified update abc123 crm_contact --id ID456 -d '{\"name\":\"Jane\"}'",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := getAPIKey()
		if err != nil {
			return err
		}

		if updateID == "" {
			return fmt.Errorf("--id is required for update")
		}
		if updateData == "" {
			return fmt.Errorf("--data is required for update")
		}

		data, err := readData(updateData)
		if err != nil {
			return err
		}

		connectionID := args[0]
		category, object, err := parser.SplitObject(args[1])
		if err != nil {
			return err
		}

		client := api.NewClient(key, "")
		body, statusCode, err := client.Do("PATCH", category, connectionID, object, updateID, data, nil)
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

func init() {
	updateCmd.Flags().StringVarP(&updateID, "id", "i", "", "Object ID (required)")
	updateCmd.Flags().StringVarP(&updateData, "data", "d", "", "JSON data (inline or @file.json) (required)")
	updateCmd.MarkFlagRequired("id")
	updateCmd.MarkFlagRequired("data")
	rootCmd.AddCommand(updateCmd)
}
