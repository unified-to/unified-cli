package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/unified-to/unified-cli/internal/api"
)

var passthroughData string

// passthroughMethods are the HTTP methods the passthrough API accepts.
var passthroughMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

var passthroughCmd = &cobra.Command{
	Use:   "passthrough <method> <connection_id> <path>",
	Short: "Call a third-party API directly through a connection",
	Long: `Send a request straight to the third-party API behind a connection, with
Unified.to handling authentication.

The path is everything after the provider's base URL. A query string on the
path is forwarded as-is.

Methods: ` + strings.Join(passthroughMethods, ", ") + `

Examples:
  unified passthrough GET abc123 v1/contacts
  unified passthrough GET abc123 "v1/contacts?updated_since=2026-01-01"
  unified passthrough POST abc123 v1/contacts -d '{"name":"John"}'
  unified passthrough DELETE abc123 v1/contacts/ID456`,
	Args: cobra.ExactArgs(3),
	ValidArgsFunction: func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return passthroughMethods, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := getAPIKey()
		if err != nil {
			return err
		}

		method := strings.ToUpper(args[0])
		if !contains(passthroughMethods, method) {
			return fmt.Errorf("unsupported passthrough method %q (supported: %s)", args[0], strings.Join(passthroughMethods, ", "))
		}

		var data []byte
		if passthroughData != "" {
			if data, err = readData(passthroughData); err != nil {
				return err
			}
		}

		connectionID := args[1]
		path, query, _ := strings.Cut(args[2], "?")
		path = strings.TrimPrefix(path, "/")
		if path == "" {
			return fmt.Errorf("a provider path is required, e.g. \"v1/contacts\"")
		}

		requestPath := fmt.Sprintf("/passthrough/%s/%s", connectionID, path)
		if query != "" {
			requestPath += "?" + query
		}

		client := api.NewClient(key, "", Version)
		body, statusCode, err := client.DoPath(method, requestPath, data, nil)
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

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func init() {
	passthroughCmd.Flags().StringVarP(&passthroughData, "data", "d", "", "JSON data (inline or @file.json)")
	rootCmd.AddCommand(passthroughCmd)
}
