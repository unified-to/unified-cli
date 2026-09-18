package parser

import (
	"fmt"
	"strings"
)

// SplitObject splits an object name like "ats_candidate" into category and
// object. It splits on the first underscore only, so a name with more than one
// underscore keeps the rest in the object segment.
//
// Prefer the catalog package, which knows the objects the API actually exposes.
// This is the fallback for names the catalog does not have.
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
