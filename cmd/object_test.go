package cmd

import (
	"strings"
	"testing"

	"github.com/unified-to/unified-cli/internal/catalog"
)

func withNoValidate(t *testing.T, v bool) {
	t.Helper()
	prev := noValidate
	noValidate = v
	t.Cleanup(func() { noValidate = prev })
}

func TestResolveObject(t *testing.T) {
	obj, err := resolveObject("ats_candidate", catalog.MethodList)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obj.Category != "ats" || obj.Resource != "candidate" {
		t.Errorf("unexpected path segments: %s/%s", obj.Category, obj.Resource)
	}
}

func TestResolveObjectUnknown(t *testing.T) {
	_, err := resolveObject("ats_candidates", catalog.MethodList)
	if err == nil {
		t.Fatal("expected an error for an unknown object")
	}
	if !strings.Contains(err.Error(), "ats_candidate") {
		t.Errorf("expected a suggestion in: %v", err)
	}
	if !strings.Contains(err.Error(), "--no-validate") {
		t.Errorf("expected the escape hatch to be mentioned in: %v", err)
	}
}

func TestResolveObjectPassthroughPointsAtItsCommand(t *testing.T) {
	_, err := resolveObject("passthrough", catalog.MethodList)
	if err == nil {
		t.Fatal("expected an error for passthrough")
	}
	if !strings.Contains(err.Error(), "unified passthrough") {
		t.Errorf("expected the passthrough command to be suggested in: %v", err)
	}
}

func TestResolveObjectUnsupportedMethod(t *testing.T) {
	_, err := resolveObject("ats_applicationstatus", catalog.MethodCreate)
	if err == nil {
		t.Fatal("expected an error for an unsupported method")
	}
	if !strings.Contains(err.Error(), "supported: list") {
		t.Errorf("expected the supported methods in: %v", err)
	}
}

func TestResolveObjectNoValidate(t *testing.T) {
	withNoValidate(t, true)

	// An object the catalog has not caught up with yet still resolves.
	obj, err := resolveObject("crm_newthing", catalog.MethodList)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obj.Category != "crm" || obj.Resource != "newthing" {
		t.Errorf("unexpected path segments: %s/%s", obj.Category, obj.Resource)
	}

	// An unsupported method is no longer an error either.
	if _, err := resolveObject("ats_applicationstatus", catalog.MethodCreate); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// A name that is not <category>_<resource> is still rejected: there is no
	// URL to build from it.
	if _, err := resolveObject("nounderscore", catalog.MethodList); err == nil {
		t.Error("expected an error for a name with no underscore")
	}
}

func TestValidateListParams(t *testing.T) {
	obj := catalog.Lookup("ats_candidate")
	if obj == nil {
		t.Fatal("expected ats_candidate in the catalog")
	}

	if err := validateListParams(obj, map[string]string{"limit": "10", "company_id": "CO1"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err := validateListParams(obj, map[string]string{"job_i": "J1"})
	if err == nil {
		t.Fatal("expected an error for an unknown list parameter")
	}
	if !strings.Contains(err.Error(), `"job_i"`) {
		t.Errorf("expected the offending parameter in: %v", err)
	}
	if !strings.Contains(err.Error(), "company_id") {
		t.Errorf("expected the accepted parameters in: %v", err)
	}
}

func TestValidateListParamsWithoutStandardParams(t *testing.T) {
	obj := catalog.Lookup("enrich_person")
	if obj == nil {
		t.Fatal("expected enrich_person in the catalog")
	}

	if err := validateListParams(obj, map[string]string{"email": "foo@bar.com"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := validateListParams(obj, map[string]string{"limit": "10"}); err == nil {
		t.Error("expected limit to be rejected on an endpoint without standard list parameters")
	}
}

func TestCompleteObjects(t *testing.T) {
	complete := completeObjects(catalog.MethodCreate)

	// The object is the second argument, so nothing is offered until the
	// connection ID is there.
	if got, _ := complete(nil, nil, "ats_"); got != nil {
		t.Errorf("expected no completions before the connection ID, got %v", got)
	}

	got, _ := complete(nil, []string{"conn1"}, "ats_")
	if len(got) == 0 {
		t.Fatal("expected ats completions")
	}
	for _, name := range got {
		if !strings.HasPrefix(name, "ats_") {
			t.Errorf("unexpected completion %q", name)
		}
		obj := catalog.Lookup(name)
		if obj == nil || !obj.Supports(catalog.MethodCreate) {
			t.Errorf("%q does not support create and should not be offered", name)
		}
	}
}
