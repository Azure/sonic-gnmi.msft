package common

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
)

// GetValueOrDefault returns the string form of values[key] or defaultValue when absent.
func GetValueOrDefault(values map[string]interface{}, key string, defaultValue string) string {
	if value, ok := values[key]; ok {
		return fmt.Sprint(value)
	}
	return defaultValue
}

// GetNonZeroValueOrEmpty returns the string form of values[key] if it parses as a non-zero integer; otherwise an empty string.
func GetNonZeroValueOrEmpty(values map[string]interface{}, key string) string {
	if value, ok := values[key]; ok {
		if intValue, err := strconv.ParseInt(fmt.Sprint(value), Base10, 64); err == nil && intValue != 0 {
			return fmt.Sprint(value)
		}
	}
	return ""
}

// GetFieldValueString fetches data[key][field] (assuming nested map[string]interface{}) or returns defaultValue.
func GetFieldValueString(data map[string]interface{}, key string, defaultValue string, field string) string {
	entry, ok := data[key].(map[string]interface{})
	if !ok {
		return defaultValue
	}

	value, ok := entry[field]
	if !ok {
		return defaultValue
	}
	return fmt.Sprint(value)
}

// GetSumFields sums integer (string or numeric) values under data[key][fieldN]; returns defaultValue on any failure.
func GetSumFields(data map[string]interface{}, key string, defaultValue string, fields ...string) (sum string) {
	defer func() {
		if r := recover(); r != nil {
			sum = defaultValue
		}
	}()
	var total int64
	for _, field := range fields {
		value := GetFieldValueString(data, key, defaultValue, field)
		if intValue, err := strconv.ParseInt(value, 10, 64); err != nil {
			return defaultValue
		} else {
			total += intValue
		}
	}
	return strconv.FormatInt(total, 10)
}

// GetSortedKeys returns the sorted keys of a map[string]interface{}.
func GetSortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GetOrDefault returns m[key] when present; otherwise returns def. Safe for nil maps.
func GetOrDefault[T any](m map[string]T, key string, def T) T {
	if v, ok := m[key]; ok {
		return v
	}
	return def
}

// ContainsString returns true if target is present in list.
func ContainsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

// Capitalize uppercases the first byte and lowercases the rest (ASCII-focused).
func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func ParseKey(key interface{}, delimiter string) (string, string) {
	keyStr, ok := key.(string)
	if !ok {
		return "", ""
	}

	parts := strings.Split(keyStr, delimiter)
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// SplitCompositeKey splits a two-part composite key using '|' or ':' delimiters.
// Returns left, right, true on success; empty strings and false otherwise.
// Examples:
//
//	"Vlan100|Ethernet0" -> ("Vlan100", "Ethernet0", true)
//	"PortChannel001:Ethernet4" -> ("PortChannel001", "Ethernet4", true)
func SplitCompositeKey(k string) (string, string, bool) {
	if parts := strings.Split(k, "|"); len(parts) == 2 {
		return parts[0], parts[1], true
	}
	if parts := strings.Split(k, ":"); len(parts) == 2 {
		return parts[0], parts[1], true
	}
	return "", "", false
}

// ParseIPv4 validates the input string and returns the parsed IPv4 address or an error.
func ParseIPv4(ipStr string) (net.IP, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil || ip.To4() == nil {
		return nil, fmt.Errorf("invalid IPv4 address: %s", ipStr)
	}
	return ip, nil
}

// ParseIPv6 validates the input string and returns the parsed IPv6 address or an error.
func ParseIPv6(ipStr string) (net.IP, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil || ip.To4() != nil {
		return nil, fmt.Errorf("invalid IPv6 address: %s", ipStr)
	}
	return ip, nil
}
