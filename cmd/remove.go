package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/unified-to/unified-cli/internal/api"
	"github.com/unified-to/unified-cli/internal/parser"
)

var removeID string

var removeCmd = &cobra.Command{
	Use:   "remove <connection_id> <object>",
	Short: "Remove an object by ID",
	Long:  "Remove an object from the Unified.to API.\nExample: unified remove abc123 crm_contact --id ID456",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := getAPIKey()
		if err != nil {
			return err
		}

		if removeID == "" {
			return fmt.Errorf("--id is required for remove")
		}

		connectionID := args[0]
		category, object, err := parser.SplitObject(args[1])
		if err != nil {
			return err
		}

		client := api.NewClient(key, "", Version)
		body, statusCode, err := client.Do("DELETE", category, connectionID, object, removeID, nil, nil)
		if err != nil {
			return err
		}

		if statusCode >= 400 {
			fmt.Fprintln(os.Stderr, string(body))
			os.Exit(1)
		}

		if len(body) > 0 {
			fmt.Print(string(body))
		}
		return nil
	},
}

func init() {
	removeCmd.Flags().StringVarP(&removeID, "id", "i", "", "Object ID (required)")
	removeCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(removeCmd)
}
