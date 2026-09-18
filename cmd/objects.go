package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/unified-to/unified-cli/internal/catalog"
)

var objectsJSON bool

var objectsCmd = &cobra.Command{
	Use:   "objects [category|object]",
	Short: "List the objects the Unified.to API supports",
	Long: `List the objects the Unified.to API supports.

With no argument, prints every object grouped by category along with the
methods it supports. Pass a category to narrow the list, or an object name
to see its endpoints and list filters.

Examples:
  unified objects
  unified objects ats
  unified objects ats_candidate
  unified objects --json`,
	Args: cobra.MaximumNArgs(1),
	ValidArgsFunction: func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var out []string
		for _, name := range append(catalog.Categories(), catalog.Names()...) {
			if strings.HasPrefix(name, toComplete) {
				out = append(out, name)
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return printObjects(catalog.Objects)
		}

		arg := strings.ToLower(strings.TrimSpace(args[0]))
		if obj := catalog.Lookup(arg); obj != nil {
			return printObject(obj)
		}
		if objects := catalog.InCategory(arg); len(objects) > 0 {
			return printObjects(objects)
		}
		return unknownObjectError(args[0])
	},
}

func printObjects(objects []catalog.Object) error {
	if objectsJSON {
		return writeJSON(objects)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	category := ""
	for _, o := range objects {
		if o.Category != category {
			if category != "" {
				fmt.Fprintln(w)
			}
			category = o.Category
			fmt.Fprintf(w, "%s\n", category)
		}
		fmt.Fprintf(w, "  %s\t%s\n", o.Name, strings.Join(o.MethodNames(), ", "))
	}
	return w.Flush()
}

// methodRequests maps a method to the HTTP request the CLI sends for it.
var methodRequests = map[catalog.Method]struct {
	verb string
	byID bool
}{
	catalog.MethodList:   {"GET", false},
	catalog.MethodGet:    {"GET", true},
	catalog.MethodCreate: {"POST", false},
	catalog.MethodUpdate: {"PATCH", true},
	catalog.MethodRemove: {"DELETE", true},
}

func printObject(obj *catalog.Object) error {
	if objectsJSON {
		return writeJSON(obj)
	}

	fmt.Println(obj.Name)
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, m := range catalog.AllMethods {
		if !obj.Supports(m) {
			continue
		}
		req := methodRequests[m]
		path := fmt.Sprintf("/%s/{connection_id}/%s", obj.Category, obj.Resource)
		if req.byID {
			path += "/{id}"
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\n", m, req.verb, path)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	if !obj.Supports(catalog.MethodList) {
		return nil
	}

	if len(obj.Filters) > 0 {
		fmt.Println()
		fmt.Println("List filters:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, f := range obj.Filters {
			if len(f.Values) == 0 {
				fmt.Fprintf(w, "  --%s\n", f.Name)
				continue
			}
			detail := "one of: "
			if f.Multi {
				detail = "one or more (comma-delimited) of: "
			}
			fmt.Fprintf(w, "  --%s\t%s%s\n", f.Name, detail, strings.Join(f.Values, ", "))
		}
		if err := w.Flush(); err != nil {
			return err
		}
	}

	fmt.Println()
	if obj.StandardListParams {
		fmt.Printf("Standard list parameters: --%s\n", strings.Join(standardListFlags(), " --"))
	} else {
		fmt.Println("This endpoint does not accept the standard list parameters.")
	}
	return nil
}

// standardListFlags returns catalog.StandardListParams as the flags the CLI
// documents for them. The API parameter is accepted under either spelling.
func standardListFlags() []string {
	flags := make([]string, 0, len(catalog.StandardListParams))
	for _, p := range catalog.StandardListParams {
		if p == "updated_gte" {
			p = "updated-gte"
		}
		flags = append(flags, p)
	}
	return flags
}

func writeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func init() {
	objectsCmd.Flags().BoolVar(&objectsJSON, "json", false, "Print the catalog as JSON")
	rootCmd.AddCommand(objectsCmd)
}
