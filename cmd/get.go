package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/unified-to/unified-cli/internal/api"
	"github.com/unified-to/unified-cli/internal/parser"
)

var getID string

var getCmd = &cobra.Command{
	Use:   "get <connection_id> <object>",
	Short: "Get a single object by ID",
	Long:  "Get a single object from the Unified.to API.\nExample: unified get abc123 ats_candidate --id ID456",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := getAPIKey()
		if err != nil {
			return err
		}

		if getID == "" {
			return fmt.Errorf("--id is required for get")
		}

		connectionID := args[0]
		category, object, err := parser.SplitObject(args[1])
		if err != nil {
			return err
		}

		client := api.NewClient(key, "")
		body, statusCode, err := client.Do("GET", category, connectionID, object, getID, nil, nil)
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
	getCmd.Flags().StringVarP(&getID, "id", "i", "", "Object ID (required)")
	getCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(getCmd)
}
