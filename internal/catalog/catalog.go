// Package catalog describes the Unified objects the API exposes and the
// operations each one supports.
//
// The data in objects_gen.go is generated from the unified-api route
// definitions; see internal/catalog/gen.
package catalog

import (
	"sort"
	"strings"
)

//go:generate go run ./gen -api ../../../unified-api -out objects_gen.go

// Method is a CRUD operation the CLI can perform on an object.
type Method string

const (
	MethodList   Method = "list"
	MethodGet    Method = "get"
	MethodCreate Method = "create"
	MethodUpdate Method = "update"
	MethodRemove Method = "remove"
)

// Filter is a query parameter accepted by an object's list operation, in
// addition to StandardListParams.
type Filter struct {
	// Name is the query parameter, passed to the CLI as --<name>.
	Name string `json:"name"`
	// Values are the accepted values, when the API constrains them.
	Values []string `json:"values,omitempty"`
	// Multi reports whether several Values may be passed comma-delimited.
	Multi bool `json:"multi,omitempty"`
}

// Object is a Unified object type, addressable at
// /{Category}/{connection_id}/{Resource}.
type Object struct {
	// Name is the object type, e.g. "ats_candidate".
	Name string `json:"name"`
	// Category is the leading path segment, e.g. "ats".
	Category string `json:"category"`
	// Resource is the trailing path segment, e.g. "candidate".
	Resource string `json:"resource"`
	// Methods are the operations the API exposes for this object.
	Methods []Method `json:"methods"`
	// Filters are object-specific list query parameters.
	Filters []Filter `json:"filters,omitempty"`
	// StandardListParams reports whether list accepts the parameters in
	// StandardListParams — a few hand-wired endpoints do not.
	StandardListParams bool `json:"standard_list_params"`
}

// StandardListParams are the list query parameters accepted by every object
// whose routes come from the API's shared route builder.
var StandardListParams = []string{"query", "limit", "offset", "updated_gte", "sort", "order", "fields", "raw"}

// AllMethods are the operations the CLI supports, in the order they are
// reported.
var AllMethods = []Method{MethodList, MethodGet, MethodCreate, MethodUpdate, MethodRemove}

var byName = func() map[string]*Object {
	m := make(map[string]*Object, len(Objects))
	for i := range Objects {
		m[Objects[i].Name] = &Objects[i]
	}
	return m
}()

// Lookup returns the object with the given name, or nil if it is not in the
// catalog.
func Lookup(name string) *Object {
	return byName[strings.ToLower(strings.TrimSpace(name))]
}

// Names returns every object name, sorted.
func Names() []string {
	names := make([]string, 0, len(Objects))
	for _, o := range Objects {
		names = append(names, o.Name)
	}
	return names
}

// Categories returns every category, sorted and deduplicated.
func Categories() []string {
	seen := map[string]bool{}
	var out []string
	for _, o := range Objects {
		if !seen[o.Category] {
			seen[o.Category] = true
			out = append(out, o.Category)
		}
	}
	sort.Strings(out)
	return out
}

// InCategory returns the objects belonging to a category, sorted by name.
func InCategory(category string) []Object {
	category = strings.ToLower(strings.TrimSpace(category))
	var out []Object
	for _, o := range Objects {
		if o.Category == category {
			out = append(out, o)
		}
	}
	return out
}

// Supports reports whether the object exposes the given method.
func (o *Object) Supports(m Method) bool {
	for _, have := range o.Methods {
		if have == m {
			return true
		}
	}
	return false
}

// MethodNames returns the object's methods as strings, for display.
func (o *Object) MethodNames() []string {
	out := make([]string, 0, len(o.Methods))
	for _, m := range o.Methods {
		out = append(out, string(m))
	}
	return out
}

// Filter returns the named list filter, or nil.
func (o *Object) Filter(name string) *Filter {
	for i := range o.Filters {
		if o.Filters[i].Name == name {
			return &o.Filters[i]
		}
	}
	return nil
}

// Suggest returns catalog names close to the given one, best match first, for
// "did you mean" hints. It returns at most max entries.
func Suggest(name string, max int) []string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return nil
	}

	type scored struct {
		name  string
		score int
	}
	var candidates []scored
	for _, o := range Objects {
		d := editDistance(name, o.Name)
		// Anything sharing a category or a resource is worth offering even
		// when the rest of the name is far off.
		category, resource, _ := strings.Cut(name, "_")
		related := o.Category == category || o.Resource == resource || strings.Contains(o.Name, name) || strings.Contains(name, o.Resource)
		if d <= 3 || related {
			candidates = append(candidates, scored{o.Name, d})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score < candidates[j].score
		}
		return candidates[i].name < candidates[j].name
	})

	out := make([]string, 0, max)
	for _, c := range candidates {
		if len(out) == max {
			break
		}
		out = append(out, c.name)
	}
	return out
}

// editDistance is the Levenshtein distance between a and b.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
