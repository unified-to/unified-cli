package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestStripComments(t *testing.T) {
	src := `
setRoute(server, { object: 'crm_contact', methods: ['L'] });
// setRoute(server, { object: 'crm_ghost', methods: ['L'] });
/*
setRoute(server, { object: 'crm_blocked', methods: ['L'] });
*/
const url = 'https://example.com/not-a-comment';
`
	got := string(stripComments([]byte(src)))
	if !strings.Contains(got, "crm_contact") {
		t.Error("live registration was stripped")
	}
	if strings.Contains(got, "crm_ghost") {
		t.Error("line comment was not stripped")
	}
	if strings.Contains(got, "crm_blocked") {
		t.Error("block comment was not stripped")
	}
	if !strings.Contains(got, "https://example.com/not-a-comment") {
		t.Error("// inside a string literal was treated as a comment")
	}
}

func TestBraceBlock(t *testing.T) {
	src := `prefix {a: {b: 1}, c: 2} suffix`
	got, ok := braceBlock(src, strings.Index(src, "{"))
	if !ok {
		t.Fatal("expected a block")
	}
	if got != `{a: {b: 1}, c: 2}` {
		t.Errorf("unexpected block: %s", got)
	}

	if _, ok := braceBlock("{unterminated", 0); ok {
		t.Error("expected an unterminated block to fail")
	}
	if _, ok := braceBlock("no brace here", 0); ok {
		t.Error("expected a non-brace start to fail")
	}
}

func TestParseSetRoute(t *testing.T) {
	enums := map[string][]string{"AtsDocumentType": {"RESUME", "OTHER"}}
	block := `{
        object: 'ats_document',
        joiSchema: joiAtsDocument,
        methods: ['C', 'R', 'U', 'D', 'L'],
        list_params: ['candidate_id', 'type'],
        uploadFile: true,
        list_params_enums: {
            type: [...AtsDocumentType],
        },
    }`

	obj, err := parseSetRoute(block, enums)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obj.Category != "ats" || obj.Resource != "document" {
		t.Errorf("unexpected path segments: %s/%s", obj.Category, obj.Resource)
	}
	if !reflect.DeepEqual(obj.Methods, []string{"list", "get", "create", "update", "remove"}) {
		t.Errorf("unexpected methods: %v", obj.Methods)
	}
	if !obj.Standard {
		t.Error("setRoute objects accept the standard list parameters")
	}
	if len(obj.Filters) != 2 {
		t.Fatalf("expected 2 filters, got %v", obj.Filters)
	}
	if obj.Filters[1].Name != "type" || !reflect.DeepEqual(obj.Filters[1].Values, []string{"RESUME", "OTHER"}) {
		t.Errorf("enum was not resolved: %+v", obj.Filters[1])
	}
}

func TestParseSetRouteInlineEnumAndMulti(t *testing.T) {
	block := `{
        object: 'accounting_contact',
        methods: ['L'],
        list_params: ['type'],
        list_params_multi: ['type'],
        list_params_enums: { type: ['CUSTOMER', 'SUPPLIER'] },
    }`

	obj, err := parseSetRoute(block, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(obj.Filters) != 1 || !obj.Filters[0].Multi {
		t.Fatalf("expected one multi-valued filter, got %+v", obj.Filters)
	}
	if !reflect.DeepEqual(obj.Filters[0].Values, []string{"CUSTOMER", "SUPPLIER"}) {
		t.Errorf("unexpected values: %v", obj.Filters[0].Values)
	}
}

func TestParseSetRouteErrors(t *testing.T) {
	cases := map[string]string{
		"no object field":  `{ methods: ['L'] }`,
		"no methods field": `{ object: 'crm_contact' }`,
		"bad method":       `{ object: 'crm_contact', methods: ['Z'] }`,
		"bad name":         `{ object: 'contact', methods: ['L'] }`,
		"unresolved enum":  `{ object: 'crm_contact', methods: ['L'], list_params: ['type'], list_params_enums: { type: [...NoSuchEnum] } }`,
		"filters, no list": `{ object: 'crm_contact', methods: ['R'], list_params: ['type'] }`,
	}
	for name, block := range cases {
		if _, err := parseSetRoute(block, nil); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestReconcile(t *testing.T) {
	objects := map[string]object{"crm_contact": {Name: "crm_contact"}}

	if err := reconcile([]string{"crm_contact", "passthrough"}, objects); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err := reconcile([]string{"crm_contact", "crm_brandnew"}, objects)
	if err == nil || !strings.Contains(err.Error(), "crm_brandnew") {
		t.Errorf("expected the missing object to be reported, got %v", err)
	}

	err = reconcile([]string{}, objects)
	if err == nil || !strings.Contains(err.Error(), "not in ObjectType") {
		t.Errorf("expected the stray object to be reported, got %v", err)
	}
}
