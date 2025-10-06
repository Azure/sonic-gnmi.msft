package common

const redactedString = "[REDACTED]"

// Redaction is performed at the top level only and only supports when map values are primitive types only.
func RedactSensitiveData(msi map[string]interface{}, keys []string) (map[string]interface{}) {
	if msi == nil {
		return nil
	}

	result := make(map[string]interface{}, len(msi))
	for key, value := range msi {
		result[key] = value
	}

	if len(keys) == 0 {
		return result
	}

	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}

	for key := range result {
		if keySet[key] {
			result[key] = redactedString
		}
	}

	return result
}
