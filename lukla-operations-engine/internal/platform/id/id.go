// Package id provides ID generation utilities.
// All primary keys use UUID v4 unless otherwise specified.
package id

import "github.com/google/uuid"

// New generates a new random UUID string.
func New() string {
	return uuid.New().String()
}

// Validate returns true if s is a valid UUID.
func Validate(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
