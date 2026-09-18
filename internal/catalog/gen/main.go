// Command gen produces internal/catalog/objects_gen.go from a checkout of the
// unified-api repository.
//
// The Unified API registers every object route from TypeScript source: most go
// through `setRoute()` in src/server/routes/<category>/index.ts, and a handful
// are wired by hand with `addRoute()`. This generator reads both, cross-checks
// the result against the canonical `ObjectType` list in src/models/Unified.ts,
// and fails if the two disagree — so an object added to the API can never be
// silently missing from the CLI catalog.
//
// Usage:
//
//	go run ./internal/catalog/gen -api ../unified-api
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func main() {
	apiDir := flag.String("api", "../unified-api", "path to a unified-api checkout")
	out := flag.String("out", "internal/catalog/objects_gen.go", "output file")
	flag.Parse()

	src, err := generate(*apiDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, src, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "gen: wrote %s\n", *out)
}

// object is the generator's intermediate representation of one Unified object.
type object struct {
	Name     string
	Category string
	Resource string
	Methods  []string
	Filters  []filter
	Standard bool // list accepts the standard limit/offset/sort/... parameters
}

type filter struct {
	Name   string
	Values []string
	Multi  bool
}

func generate(apiDir string) ([]byte, error) {
	objectTypes, err := readObjectTypes(apiDir)
	if err != nil {
		return nil, err
	}
	enums, err := readEnums(apiDir)
	if err != nil {
		return nil, err
	}
	objects, err := readSetRoutes(apiDir, enums)
	if err != nil {
		return nil, err
	}
	specials, err := readSpecialRoutes(apiDir)
	if err != nil {
		return nil, err
	}
	for _, s := range specials {
		if _, dup := objects[s.Name]; dup {
			return nil, fmt.Errorf("%s is registered by both setRoute and a hand-written route", s.Name)
		}
		objects[s.Name] = s
	}

	if err := reconcile(objectTypes, objects); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(objects))
	for name := range objects {
		names = append(names, name)
	}
	sort.Strings(names)

	return render(names, objects)
}

// skipObjectTypes are ObjectType entries that are not addressable as
// /{category}/{connection_id}/{resource}, so they carry no catalog entry.
var skipObjectTypes = map[string]string{
	// Proxied straight through to the provider: /passthrough/{connection_id}/{path*}
	// (src/server/routes/passthrough/index.ts). The CLI exposes it as its own command.
	"passthrough": "handled by the passthrough command",
}

func reconcile(objectTypes []string, objects map[string]object) error {
	known := map[string]bool{}
	var problems []string
	for _, name := range objectTypes {
		known[name] = true
		if _, skipped := skipObjectTypes[name]; skipped {
			if _, found := objects[name]; found {
				problems = append(problems, fmt.Sprintf("%s is in skipObjectTypes but was also discovered as a route", name))
			}
			continue
		}
		if _, found := objects[name]; !found {
			problems = append(problems, fmt.Sprintf("%s is in ObjectType but no route was discovered for it", name))
		}
	}
	for name := range objects {
		if !known[name] {
			problems = append(problems, fmt.Sprintf("%s has a route but is not in ObjectType", name))
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("unified-api and the CLI catalog disagree:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

// --- unified-api source readers ---------------------------------------------

var (
	objectTypeBlockRe = regexp.MustCompile(`(?s)export const ObjectType = \[(.*?)\] as const;`)
	stringRe          = regexp.MustCompile(`'([^']*)'`)
	enumRe            = regexp.MustCompile(`(?s)export const ([A-Za-z0-9_]+) = \[(.*?)\] as const;`)
	setRouteRe        = regexp.MustCompile(`setRoute\s*(?:<[^>]*>)?\s*\(\s*server\s*,\s*\{`)
	objectFieldRe     = regexp.MustCompile(`\bobject\s*:\s*'([^']+)'`)
	methodsFieldRe    = regexp.MustCompile(`\bmethods\s*:\s*\[([^\]]*)\]`)
	listParamsFieldRe = regexp.MustCompile(`\blist_params\s*:\s*\[([^\]]*)\]`)
	listMultiFieldRe  = regexp.MustCompile(`\blist_params_multi\s*:\s*\[([^\]]*)\]`)
	enumEntryRe       = regexp.MustCompile(`([A-Za-z0-9_]+)\s*:\s*\[([^\]]*)\]`)
	spreadRe          = regexp.MustCompile(`\.\.\.([A-Za-z0-9_]+)`)
)

func readObjectTypes(apiDir string) ([]string, error) {
	path := filepath.Join(apiDir, "src", "models", "Unified.ts")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	m := objectTypeBlockRe.FindSubmatch(stripComments(data))
	if m == nil {
		return nil, fmt.Errorf("could not find `export const ObjectType` in %s", path)
	}
	names := literalStrings(string(m[1]))
	if len(names) == 0 {
		return nil, fmt.Errorf("ObjectType in %s is empty", path)
	}
	return names, nil
}

// readEnums collects `export const Foo = [...] as const;` from the Unified
// models so that `list_params_enums: { type: [...AtsDocumentType] }` can be
// resolved to its values.
func readEnums(apiDir string) (map[string][]string, error) {
	dir := filepath.Join(apiDir, "src", "models")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	enums := map[string][]string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "Unified") || !strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".test.ts") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		for _, m := range enumRe.FindAllSubmatch(stripComments(data), -1) {
			enums[string(m[1])] = literalStrings(string(m[2]))
		}
	}
	return enums, nil
}

func readSetRoutes(apiDir string, enums map[string][]string) (map[string]object, error) {
	dir := filepath.Join(apiDir, "src", "server", "routes")
	files, err := tsFiles(dir)
	if err != nil {
		return nil, err
	}

	objects := map[string]object{}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		src := string(stripComments(raw))
		for _, loc := range setRouteRe.FindAllStringIndex(src, -1) {
			block, ok := braceBlock(src, loc[1]-1)
			if !ok {
				return nil, fmt.Errorf("%s: unterminated setRoute() literal", path)
			}
			obj, err := parseSetRoute(block, enums)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			if prev, dup := objects[obj.Name]; dup {
				return nil, fmt.Errorf("%s: %s is registered twice (also as %v)", path, obj.Name, prev.Methods)
			}
			objects[obj.Name] = obj
		}
	}
	if len(objects) == 0 {
		return nil, fmt.Errorf("no setRoute() calls found under %s", dir)
	}
	return objects, nil
}

var routeLetters = map[string]string{
	"L": "list",
	"R": "get",
	"C": "create",
	"U": "update",
	"D": "remove",
}

func parseSetRoute(block string, enums map[string][]string) (object, error) {
	m := objectFieldRe.FindStringSubmatch(block)
	if m == nil {
		return object{}, fmt.Errorf("setRoute() literal has no `object` field")
	}
	obj, err := newObject(m[1])
	if err != nil {
		return object{}, err
	}
	obj.Standard = true

	mm := methodsFieldRe.FindStringSubmatch(block)
	if mm == nil {
		return object{}, fmt.Errorf("%s: setRoute() literal has no `methods` field", obj.Name)
	}
	for _, letter := range literalStrings(mm[1]) {
		verb, ok := routeLetters[letter]
		if !ok {
			return object{}, fmt.Errorf("%s: unknown setRoute method letter %q", obj.Name, letter)
		}
		obj.Methods = append(obj.Methods, verb)
	}
	sortMethods(obj.Methods)

	var params []string
	if pm := listParamsFieldRe.FindStringSubmatch(block); pm != nil {
		params = literalStrings(pm[1])
	}
	multi := map[string]bool{}
	if mm := listMultiFieldRe.FindStringSubmatch(block); mm != nil {
		for _, name := range literalStrings(mm[1]) {
			multi[name] = true
		}
	}
	values, err := parseListParamEnums(obj.Name, block, enums)
	if err != nil {
		return object{}, err
	}
	for _, name := range params {
		obj.Filters = append(obj.Filters, filter{Name: name, Values: values[name], Multi: multi[name]})
	}
	if !containsString(obj.Methods, "list") && len(obj.Filters) > 0 {
		return object{}, fmt.Errorf("%s: declares list_params but no `L` method", obj.Name)
	}
	return obj, nil
}

func parseListParamEnums(name, block string, enums map[string][]string) (map[string][]string, error) {
	idx := strings.Index(block, "list_params_enums")
	if idx == -1 {
		return nil, nil
	}
	open := strings.Index(block[idx:], "{")
	if open == -1 {
		return nil, fmt.Errorf("%s: malformed list_params_enums", name)
	}
	enumBlock, ok := braceBlock(block, idx+open)
	if !ok {
		return nil, fmt.Errorf("%s: unterminated list_params_enums", name)
	}

	values := map[string][]string{}
	for _, m := range enumEntryRe.FindAllStringSubmatch(enumBlock, -1) {
		param, body := m[1], m[2]
		if spread := spreadRe.FindStringSubmatch(body); spread != nil {
			resolved, ok := enums[spread[1]]
			if !ok {
				return nil, fmt.Errorf("%s: list_params_enums references unknown enum %s", name, spread[1])
			}
			values[param] = resolved
			continue
		}
		literals := literalStrings(body)
		if len(literals) == 0 {
			return nil, fmt.Errorf("%s: could not resolve list_params_enums for %q", name, param)
		}
		values[param] = literals
	}
	return values, nil
}

// specialRoute describes an object whose routes are registered with bespoke
// addRoute() calls rather than setRoute(). Each entry names the source file and
// the route paths it expects to find there; the generator verifies both, so the
// hand-maintained detail below cannot drift from unified-api unnoticed.
type specialRoute struct {
	object
	file   string
	expect []string
}

// joiScimQueryParams in src/server/routes/scim/index.ts.
var scimQueryParams = []filter{
	{Name: "filter"}, {Name: "sortBy"}, {Name: "sortOrder"}, {Name: "startIndex"}, {Name: "count"},
}

var specialRoutes = []specialRoute{
	{
		// src/server/routes/enrich/index.ts — a search, not a collection: no id
		// route and no standard list parameters.
		object: object{
			Name: "enrich_person", Methods: []string{"list"},
			Filters: []filter{{Name: "email"}, {Name: "name"}, {Name: "company_name"}, {Name: "twitter"}, {Name: "linkedin_url"}},
		},
		file:   "src/server/routes/enrich/index.ts",
		expect: []string{"enrich/{connection_id}/person"},
	},
	{
		object: object{
			Name: "enrich_company", Methods: []string{"list"},
			Filters: []filter{{Name: "domain"}, {Name: "name"}},
		},
		file:   "src/server/routes/enrich/index.ts",
		expect: []string{"enrich/{connection_id}/company"},
	},
	{
		// src/server/routes/assessment/index.ts — list takes limit/offset only.
		object: object{
			Name:    "assessment_package",
			Methods: []string{"list", "get", "create", "update", "remove"},
			Filters: []filter{{Name: "limit"}, {Name: "offset"}},
		},
		file: "src/server/routes/assessment/index.ts",
		expect: []string{
			"assessment/{connection_id}/package",
			"assessment/{connection_id}/package/{id}",
		},
	},
	{
		// src/server/routes/scim/index.ts — SCIM 2.0 shapes, registered under
		// both `users` and `Users`; the CLI uses the lowercase form.
		object: object{
			Name: "scim_users", Methods: []string{"list", "get", "create", "update", "remove"},
			Filters: scimQueryParams,
		},
		file: "src/server/routes/scim/index.ts",
		expect: []string{
			"scim/{connection_id}/users",
			"scim/{connection_id}/users/{id}",
		},
	},
	{
		object: object{
			Name: "scim_groups", Methods: []string{"list", "get", "create", "update", "remove"},
			Filters: scimQueryParams,
		},
		file: "src/server/routes/scim/index.ts",
		expect: []string{
			"scim/{connection_id}/groups",
			"scim/{connection_id}/groups/{id}",
		},
	},
}

func readSpecialRoutes(apiDir string) ([]object, error) {
	cache := map[string]string{}
	out := make([]object, 0, len(specialRoutes))
	for _, s := range specialRoutes {
		src, ok := cache[s.file]
		if !ok {
			data, err := os.ReadFile(filepath.Join(apiDir, s.file))
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", s.file, err)
			}
			src = string(stripComments(data))
			cache[s.file] = src
		}
		for _, path := range s.expect {
			if !strings.Contains(src, "'"+path+"'") {
				return nil, fmt.Errorf("%s: expected route %q for %s is gone — update specialRoutes", s.file, path, s.Name)
			}
		}
		obj, err := newObject(s.Name)
		if err != nil {
			return nil, err
		}
		obj.Methods = append(obj.Methods, s.Methods...)
		sortMethods(obj.Methods)
		obj.Filters = append(obj.Filters, s.Filters...)
		out = append(out, obj)
	}
	return out, nil
}

// --- helpers ----------------------------------------------------------------

func newObject(name string) (object, error) {
	category, resource, ok := strings.Cut(name, "_")
	if !ok || category == "" || resource == "" {
		return object{}, fmt.Errorf("object name %q is not <category>_<resource>", name)
	}
	return object{Name: name, Category: category, Resource: resource}, nil
}

var methodOrder = map[string]int{"list": 0, "get": 1, "create": 2, "update": 3, "remove": 4}

func sortMethods(m []string) {
	sort.Slice(m, func(i, j int) bool { return methodOrder[m[i]] < methodOrder[m[j]] })
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func literalStrings(s string) []string {
	var out []string
	for _, m := range stringRe.FindAllStringSubmatch(s, -1) {
		out = append(out, m[1])
	}
	return out
}

func tsFiles(dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".test.ts") {
			return nil
		}
		out = append(out, path)
		return nil
	})
	sort.Strings(out)
	return out, err
}

// stripComments blanks out // and /* */ comments so that commented-out route
// registrations are not mistaken for live ones.
func stripComments(src []byte) []byte {
	var out bytes.Buffer
	out.Grow(len(src))
	const (
		code = iota
		lineComment
		blockComment
		single
		double
		backtick
	)
	state := code
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch state {
		case code:
			switch {
			case c == '/' && i+1 < len(src) && src[i+1] == '/':
				state = lineComment
				i++
			case c == '/' && i+1 < len(src) && src[i+1] == '*':
				state = blockComment
				i++
			default:
				if c == '\'' {
					state = single
				} else if c == '"' {
					state = double
				} else if c == '`' {
					state = backtick
				}
				out.WriteByte(c)
			}
		case lineComment:
			if c == '\n' {
				state = code
				out.WriteByte(c)
			}
		case blockComment:
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				state = code
				i++
			} else if c == '\n' {
				out.WriteByte(c)
			}
		case single, double, backtick:
			out.WriteByte(c)
			if c == '\\' && i+1 < len(src) {
				i++
				out.WriteByte(src[i])
				continue
			}
			if (state == single && c == '\'') || (state == double && c == '"') || (state == backtick && c == '`') {
				state = code
			}
		}
	}
	return out.Bytes()
}

// braceBlock returns the {...} block starting at src[start], braces included.
func braceBlock(src string, start int) (string, bool) {
	if start < 0 || start >= len(src) || src[start] != '{' {
		return "", false
	}
	depth := 0
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start : i+1], true
			}
		}
	}
	return "", false
}

// --- rendering --------------------------------------------------------------

func render(names []string, objects map[string]object) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by internal/catalog/gen from the unified-api routes. DO NOT EDIT.\n\n")
	fmt.Fprintf(&b, "package catalog\n\n")
	fmt.Fprintf(&b, "// Objects lists every Unified object the API exposes at\n")
	fmt.Fprintf(&b, "// /{category}/{connection_id}/{resource}, sorted by name.\n")
	fmt.Fprintf(&b, "var Objects = []Object{\n")
	for _, name := range names {
		o := objects[name]
		fmt.Fprintf(&b, "\t{\n")
		fmt.Fprintf(&b, "\t\tName:     %q,\n", o.Name)
		fmt.Fprintf(&b, "\t\tCategory: %q,\n", o.Category)
		fmt.Fprintf(&b, "\t\tResource: %q,\n", o.Resource)
		fmt.Fprintf(&b, "\t\tMethods:  []Method{%s},\n", renderMethods(o.Methods))
		if o.Standard {
			fmt.Fprintf(&b, "\t\tStandardListParams: true,\n")
		}
		if len(o.Filters) > 0 {
			fmt.Fprintf(&b, "\t\tFilters: []Filter{\n")
			for _, f := range o.Filters {
				fmt.Fprintf(&b, "\t\t\t{Name: %q", f.Name)
				if f.Multi {
					fmt.Fprintf(&b, ", Multi: true")
				}
				if len(f.Values) > 0 {
					fmt.Fprintf(&b, ", Values: []string{%s}", renderStrings(f.Values))
				}
				fmt.Fprintf(&b, "},\n")
			}
			fmt.Fprintf(&b, "\t\t},\n")
		}
		fmt.Fprintf(&b, "\t},\n")
	}
	fmt.Fprintf(&b, "}\n")

	src, err := format.Source(b.Bytes())
	if err != nil {
		return nil, fmt.Errorf("formatting generated source: %w", err)
	}
	return src, nil
}

func renderMethods(methods []string) string {
	parts := make([]string, 0, len(methods))
	for _, m := range methods {
		parts = append(parts, "Method"+strings.ToUpper(m[:1])+m[1:])
	}
	return strings.Join(parts, ", ")
}

func renderStrings(values []string) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, fmt.Sprintf("%q", v))
	}
	return strings.Join(parts, ", ")
}
