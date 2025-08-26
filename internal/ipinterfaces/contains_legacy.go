//go:build !go1.21
// +build !go1.21

package ipinterfaces

// containsString checks if s exists in xs using a simple loop for older Go versions.
func containsString(xs []string, s string) bool {
	for _, v := range xs {
		if v == s {
			return true
		}
	}
	return false
}
