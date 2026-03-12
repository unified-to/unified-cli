package parser

import "testing"

func TestSplitObject(t *testing.T) {
	tests := []struct {
		input    string
		category string
		object   string
		wantErr  bool
	}{
		{"ats_candidate", "ats", "candidate", false},
		{"crm_contact", "crm", "contact", false},
		{"hris_employee", "hris", "employee", false},
		{"commerce_item", "commerce", "item", false},
		{"ats_application", "ats", "application", false},
		{"accounting_credit_memo", "accounting", "credit_memo", false},
		{"", "", "", true},
		{"nocategory", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			category, object, err := SplitObject(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SplitObject(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("SplitObject(%q) unexpected error: %v", tt.input, err)
				return
			}
			if category != tt.category {
				t.Errorf("SplitObject(%q) category = %q, want %q", tt.input, category, tt.category)
			}
			if object != tt.object {
				t.Errorf("SplitObject(%q) object = %q, want %q", tt.input, object, tt.object)
			}
		})
	}
}
