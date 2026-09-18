package catalog

import (
	"strings"
	"testing"
)

func TestObjectsAreWellFormed(t *testing.T) {
	if len(Objects) < 100 {
		t.Fatalf("catalog looks truncated: %d objects", len(Objects))
	}

	seen := map[string]bool{}
	for _, o := range Objects {
		if seen[o.Name] {
			t.Errorf("%s: duplicate entry", o.Name)
		}
		seen[o.Name] = true

		if o.Category+"_"+o.Resource != o.Name {
			t.Errorf("%s: category %q and resource %q do not reconstruct the name", o.Name, o.Category, o.Resource)
		}
		if len(o.Methods) == 0 {
			t.Errorf("%s: no methods", o.Name)
		}
		if !o.Supports(MethodList) && len(o.Filters) > 0 {
			t.Errorf("%s: has list filters but no list method", o.Name)
		}
		for _, f := range o.Filters {
			if f.Name == "" {
				t.Errorf("%s: unnamed filter", o.Name)
			}
			if f.Multi && len(f.Values) == 0 {
				t.Errorf("%s: filter %q is multi-valued but has no values", o.Name, f.Name)
			}
		}
	}
}

func TestObjectsAreSortedByName(t *testing.T) {
	for i := 1; i < len(Objects); i++ {
		if Objects[i-1].Name >= Objects[i].Name {
			t.Fatalf("objects are not sorted: %q before %q", Objects[i-1].Name, Objects[i].Name)
		}
	}
}

func TestLookup(t *testing.T) {
	obj := Lookup("ats_candidate")
	if obj == nil {
		t.Fatal("expected ats_candidate in the catalog")
	}
	if obj.Category != "ats" || obj.Resource != "candidate" {
		t.Errorf("unexpected path segments: %s/%s", obj.Category, obj.Resource)
	}
	if !obj.Supports(MethodCreate) {
		t.Error("expected ats_candidate to support create")
	}

	if Lookup("  ATS_Candidate ") == nil {
		t.Error("expected lookup to normalise case and whitespace")
	}
	if Lookup("nope_nope") != nil {
		t.Error("expected nil for an unknown object")
	}
	if Lookup("passthrough") != nil {
		t.Error("passthrough is not an object endpoint and should not be in the catalog")
	}
}

func TestReadOnlyObject(t *testing.T) {
	obj := Lookup("ats_applicationstatus")
	if obj == nil {
		t.Fatal("expected ats_applicationstatus in the catalog")
	}
	if !obj.Supports(MethodList) {
		t.Error("expected list support")
	}
	for _, m := range []Method{MethodGet, MethodCreate, MethodUpdate, MethodRemove} {
		if obj.Supports(m) {
			t.Errorf("expected %s to be unsupported", m)
		}
	}
}

func TestEndpointsWithoutStandardListParams(t *testing.T) {
	// The enrich endpoints are a lookup by identifier, not a collection.
	obj := Lookup("enrich_person")
	if obj == nil {
		t.Fatal("expected enrich_person in the catalog")
	}
	if obj.StandardListParams {
		t.Error("enrich_person does not accept the standard list parameters")
	}
	if obj.Filter("email") == nil {
		t.Error("expected an email filter on enrich_person")
	}
	if obj.Filter("limit") != nil {
		t.Error("did not expect a limit filter on enrich_person")
	}
}

func TestFilterValues(t *testing.T) {
	obj := Lookup("ats_document")
	if obj == nil {
		t.Fatal("expected ats_document in the catalog")
	}
	f := obj.Filter("type")
	if f == nil {
		t.Fatal("expected a type filter on ats_document")
	}
	if len(f.Values) == 0 {
		t.Fatal("expected the type filter to be constrained to a set of values")
	}
	var found bool
	for _, v := range f.Values {
		if v == "RESUME" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected RESUME among %v", f.Values)
	}
}

func TestCategories(t *testing.T) {
	categories := Categories()
	if len(categories) < 20 {
		t.Fatalf("expected many categories, got %d", len(categories))
	}
	for i := 1; i < len(categories); i++ {
		if categories[i-1] >= categories[i] {
			t.Fatalf("categories are not sorted: %q before %q", categories[i-1], categories[i])
		}
	}

	ats := InCategory("ats")
	if len(ats) == 0 {
		t.Fatal("expected objects in the ats category")
	}
	for _, o := range ats {
		if !strings.HasPrefix(o.Name, "ats_") {
			t.Errorf("%s is not an ats object", o.Name)
		}
	}
	if len(InCategory("nope")) != 0 {
		t.Error("expected no objects for an unknown category")
	}
}

func TestSuggest(t *testing.T) {
	got := Suggest("ats_candidates", 5)
	if len(got) == 0 || got[0] != "ats_candidate" {
		t.Errorf("expected ats_candidate first, got %v", got)
	}
	if len(got) > 5 {
		t.Errorf("expected at most 5 suggestions, got %d", len(got))
	}

	// A plausible name from the wrong category should still find the object.
	got = Suggest("hr_employee", 3)
	var found bool
	for _, s := range got {
		if s == "hris_employee" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected hris_employee among %v", got)
	}

	if Suggest("", 5) != nil {
		t.Error("expected no suggestions for an empty name")
	}
}

func TestMethodNames(t *testing.T) {
	obj := Lookup("crm_contact")
	if obj == nil {
		t.Fatal("expected crm_contact in the catalog")
	}
	names := obj.MethodNames()
	if len(names) != len(obj.Methods) {
		t.Fatalf("expected %d names, got %d", len(obj.Methods), len(names))
	}
	if names[0] != "list" {
		t.Errorf("expected list first, got %q", names[0])
	}
}
