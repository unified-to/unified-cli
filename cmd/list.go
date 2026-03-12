package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/unified-to/unified-cli/internal/api"
	"github.com/unified-to/unified-cli/internal/parser"
)

var listQuery string
var listLimit string
var listOffset string
var listUpdatedGte string
var listSort string
var listOrder string
var listFields string

var listCmd = &cobra.Command{
	Use:   "list <connection_id> <object>",
	Short: "List objects",
	Long:  "List objects from the Unified.to API.\nExample: unified list abc123 ats_candidate --limit 10 --job_id J123",
	Args:  cobra.ExactArgs(2),
	FParseErrWhitelist: cobra.FParseErrWhitelist{
		UnknownFlags: true,
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := getAPIKey()
		if err != nil {
			return err
		}

		connectionID := args[0]
		category, object, err := parser.SplitObject(args[1])
		if err != nil {
			return err
		}

		// Build query params from known flags
		params := map[string]string{}
		if listQuery != "" {
			params["query"] = listQuery
		}
		if listLimit != "" {
			params["limit"] = listLimit
		}
		if listOffset != "" {
			params["offset"] = listOffset
		}
		if listUpdatedGte != "" {
			params["updated_gte"] = listUpdatedGte
		}
		if listSort != "" {
			params["sort"] = listSort
		}
		if listOrder != "" {
			params["order"] = listOrder
		}
		if listFields != "" {
			params["fields"] = listFields
		}

		// Parse unknown flags as extra query params
		extraParams := parseUnknownFlags(os.Args)
		for k, v := range extraParams {
			params[k] = v
		}

		client := api.NewClient(key, "")
		body, statusCode, err := client.Do("GET", category, connectionID, object, "", nil, params)
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

// parseUnknownFlags extracts --key value pairs that aren't known Cobra flags.
func parseUnknownFlags(args []string) map[string]string {
	knownFlags := map[string]bool{
		"--api-key": true, "-k": true,
		"--query": true, "-q": true,
		"--limit": true, "-l": true,
		"--offset": true, "-o": true,
		"--updated-gte": true,
		"--sort": true, "-s": true,
		"--order": true,
		"--fields": true, "-f": true,
		"--help": true, "-h": true,
	}

	extra := map[string]string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		// Handle --key=value
		if idx := strings.Index(arg, "="); idx != -1 {
			key := arg[:idx]
			if !knownFlags[key] {
				extra[strings.TrimPrefix(key, "--")] = arg[idx+1:]
			}
			continue
		}
		// Handle --key value
		if !knownFlags[arg] && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			extra[strings.TrimPrefix(arg, "--")] = args[i+1]
			i++
		}
	}
	return extra
}

func init() {
	listCmd.Flags().StringVarP(&listQuery, "query", "q", "", "Search/filter query")
	listCmd.Flags().StringVarP(&listLimit, "limit", "l", "", "Max results to return")
	listCmd.Flags().StringVarP(&listOffset, "offset", "o", "", "Pagination offset")
	listCmd.Flags().StringVar(&listUpdatedGte, "updated-gte", "", "ISO-8601 datetime filter")
	listCmd.Flags().StringVarP(&listSort, "sort", "s", "", "Sort field")
	listCmd.Flags().StringVar(&listOrder, "order", "", "Sort direction (asc/desc)")
	listCmd.Flags().StringVarP(&listFields, "fields", "f", "", "Comma-delimited fields to return")
	rootCmd.AddCommand(listCmd)
}
