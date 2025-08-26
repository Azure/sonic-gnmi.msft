//go:build go1.21
// +build go1.21

package ipinterfaces

import "slices"

// containsString checks if s exists in xs using Go 1.21+ slices package.
func containsString(xs []string, s string) bool {
	return slices.Contains(xs, s)
}
