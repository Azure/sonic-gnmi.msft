package common

func RedactSensitiveData(msi map[string]interface{}, keys []string, redactedString string) map[string]interface{} {
	if msi == nil || len(keys) == 0 {
		return msi
	}

	keySet := make(map[string]bool, len(keys))
	for _, k := range keys {
		keySet[k] = true
	}

	for key := range msi {
		if keySet[key] {
			msi[key] = redactedString
		}
	}

	return msi
}
