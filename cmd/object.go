package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/unified-to/unified-cli/internal/catalog"
	"github.com/unified-to/unified-cli/internal/parser"
)

// resolveObject looks the object argument up in the catalog of objects the API
// exposes and checks that it supports the requested method.
//
// With --no-validate an object that is not in the catalog still resolves, to a
// synthetic entry built by splitting the name, so that a newly released object
// can be used before the catalog is regenerated.
func resolveObject(name string, method catalog.Method) (*catalog.Object, error) {
	obj := catalog.Lookup(name)
	if obj == nil {
		if !noValidate {
			return nil, unknownObjectError(name)
		}
		category, resource, err := parser.SplitObject(name)
		if err != nil {
			return nil, err
		}
		return &catalog.Object{Name: name, Category: category, Resource: resource}, nil
	}
	if !obj.Supports(method) && !noValidate {
		return nil, fmt.Errorf(
			"%s does not support %q (supported: %s)\n\nRun \"unified objects %s\" for details, or pass --no-validate to send the request anyway",
			obj.Name, method, strings.Join(obj.MethodNames(), ", "), obj.Name)
	}
	return obj, nil
}

func unknownObjectError(name string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "unknown object %q", name)

	if strings.EqualFold(strings.TrimSpace(name), "passthrough") {
		b.WriteString("\n\nThe passthrough API has its own command: unified passthrough <method> <connection_id> <path>")
		return fmt.Errorf("%s", b.String())
	}

	if suggestions := catalog.Suggest(name, 5); len(suggestions) > 0 {
		b.WriteString("\n\nDid you mean one of these?")
		for _, s := range suggestions {
			fmt.Fprintf(&b, "\n    %s", s)
		}
	}
	fmt.Fprintf(&b, "\n\nRun \"unified objects\" to list all %d objects, or pass --no-validate to send the request anyway.", len(catalog.Objects))
	return fmt.Errorf("%s", b.String())
}

// validateListParams checks the query parameters against the ones the object's
// list endpoint accepts.
//
// The API ignores query parameters it does not recognise rather than rejecting
// them, so an unnoticed typo silently returns unfiltered results. Catching it
// here is the only place the user finds out.
func validateListParams(obj *catalog.Object, params map[string]string) error {
	allowed := map[string]bool{}
	if obj.StandardListParams {
		for _, p := range catalog.StandardListParams {
			allowed[p] = true
		}
	}
	for _, f := range obj.Filters {
		allowed[f.Name] = true
	}

	var unknown []string
	for name := range params {
		if !allowed[name] {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)

	accepted := make([]string, 0, len(allowed))
	for name := range allowed {
		accepted = append(accepted, name)
	}
	sort.Strings(accepted)

	var b strings.Builder
	fmt.Fprintf(&b, "%s does not accept the list parameter", obj.Name)
	if len(unknown) > 1 {
		b.WriteString("s")
	}
	fmt.Fprintf(&b, " %s\n\nAccepted: %s", quoteAll(unknown), strings.Join(accepted, ", "))
	fmt.Fprintf(&b, "\n\nRun \"unified objects %s\" for details, or pass --no-validate to send the request anyway.", obj.Name)
	return fmt.Errorf("%s", b.String())
}

func quoteAll(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, fmt.Sprintf("%q", v))
	}
	return strings.Join(quoted, ", ")
}

// completeObjects completes the object argument of a CRUD command with the
// catalog objects that support the given method.
func completeObjects(method catalog.Method) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// args[0] is the connection ID; the object is the second argument.
		if len(args) != 1 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var out []string
		for _, o := range catalog.Objects {
			if o.Supports(method) && strings.HasPrefix(o.Name, toComplete) {
				out = append(out, o.Name)
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}
