package parser

import (
	"fmt"
	"strings"
)

// SplitObject splits an object name like "ats_candidate" into category and object.
// Splits on the first underscore only, so "accounting_credit_memo" becomes
// category="accounting", object="credit_memo".
func SplitObject(input string) (category string, object string, err error) {
	if input == "" {
		return "", "", fmt.Errorf("object name cannot be empty")
	}
	idx := strings.Index(input, "_")
	if idx == -1 {
		return "", "", fmt.Errorf("invalid object name %q: expected format like 'ats_candidate'", input)
	}
	return input[:idx], input[idx+1:], nil
}
