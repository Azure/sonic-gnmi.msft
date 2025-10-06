package common

// RedactSensitiveData creates a deep copy of the input map and redacts specified keys.
// Redaction is performed at the top level only - nested values are not traversed.
// Assumes map values are primitive types (string, number, bool, nil) only.
// Returns (nil, nil) for nil input, and a deep copy with no redactions for empty/nil keys.
func RedactSensitiveData(msi map[string]interface{}, keys []string) (map[string]interface{}, error) {
	// Handle nil input
	if msi == nil {
		return nil, nil
	}

	// Create shallow copy (sufficient for primitive values)
	result := make(map[string]interface{}, len(msi))
	for key, value := range msi {
		result[key] = value
	}

	// If no keys provided, return the copy with no redactions
	if len(keys) == 0 {
		return result, nil
	}

	// Build key lookup map for O(1) checking
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}

	// Redact only at top level
	for key := range result {
		if keySet[key] {
			result[key] = "[REDACTED]"
		}
	}

	return result, nil
}
