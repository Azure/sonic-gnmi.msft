package common

import (
	"strings"

	log "github.com/golang/glog"
)

// CheckFeatureSupported checks if a feature is supported by reading the given
// field from the specified db, table, and key. Returns true if the field value
// is "true".
func CheckFeatureSupported(db, table, key, field string) (bool, error) {
	queries := [][]string{
		{db, table, key},
	}
	data, err := GetMapFromQueries(queries)
	if err != nil {
		log.V(2).Infof("Unable to query %s|%s: %v", table, key, err)
		return false, err
	}
	if val, ok := data[field]; ok {
		if strVal, isStr := val.(string); isStr && strings.EqualFold(strVal, "true") {
			return true, nil
		}
	}
	return false, nil
}

