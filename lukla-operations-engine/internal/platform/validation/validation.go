// Package validation provides reusable input-shape validation helpers.
// Domain-specific rules (e.g. client_id/order_id matching) belong in their owning module,
// not here.
package validation

import "strings"

// RequiredString returns an error string if v is empty after trimming, or empty string if ok.
func RequiredString(field, v string) string {
	if strings.TrimSpace(v) == "" {
		return field + " is required"
	}
	return ""
}

// Errors collects validation messages and exposes a convenience method.
type Errors []string

// Add appends a non-empty message to the list.
func (e *Errors) Add(msg string) {
	if msg != "" {
		*e = append(*e, msg)
	}
}

// HasErrors reports whether any error was recorded.
func (e Errors) HasErrors() bool { return len(e) > 0 }
