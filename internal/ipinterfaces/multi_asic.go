package ipinterfaces

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	namespacePathGlob     = "/run/netns/*"
	asicNamePrefix        = "asic"
	defaultNamespace      = ""
	machineConfPath       = "/host/machine.conf"
	containerPlatformPath = "/usr/share/sonic/platform"
	hostDevicePath        = "/usr/share/sonic/device"
	asicConfFilename      = "asic.conf"
)

// getPlatform retrieves the device's platform identifier.
// This is a port of the logic from sonic_py_common/device_info.py
func getPlatform() string {
	// 1. Check PLATFORM environment variable
	if platformEnv := os.Getenv("PLATFORM"); platformEnv != "" {
		return platformEnv
	}

	// 2. Check machine.conf
	file, err := os.Open(machineConfPath)
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
				key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
				if key == "onie_platform" || key == "aboot_platform" {
					return value
				}
			}
		}
	}

	// 3. Fallback to ConfigDB via injected DBQuery if available
	if DBQuery == nil {
		return ""
	}
	queries := [][]string{{"CONFIG_DB", "DEVICE_METADATA", "localhost"}}
	msi, err := DBQuery(queries)
	if err != nil || msi == nil {
		return ""
	}
	entry, ok := msi["DEVICE_METADATA|localhost"].(map[string]interface{})
	if !ok {
		return ""
	}
	if platform, ok := entry["platform"].(string); ok {
		return platform
	}
	return ""
}

// getAsicConfFilePath retrieves the path to the ASIC configuration file.
// This is a port of the logic from sonic_py_common/device_info.py
func getAsicConfFilePath() string {
	// Candidate 1: /usr/share/sonic/platform/asic.conf
	candidate1 := filepath.Join(containerPlatformPath, asicConfFilename)
	if _, err := os.Stat(candidate1); err == nil {
		return candidate1
	}

	// Candidate 2: /usr/share/sonic/device/<platform>/asic.conf
	platform := getPlatform()
	if platform != "" {
		candidate2 := filepath.Join(hostDevicePath, platform, asicConfFilename)
		if _, err := os.Stat(candidate2); err == nil {
			return candidate2
		}
	}

	return "" // No file found
}

// GetNumASICs retrieves the number of ASICs present on the platform.
// It reads the asic.conf file and counts the number of lines.
func GetNumASICs() (int, error) {
	asicConfPath := getAsicConfFilePath()
	if asicConfPath == "" {
		// If no asic.conf file is found, assume a single ASIC platform.
		return 1, nil
	}

	file, err := os.Open(asicConfPath)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, it's a single ASIC platform.
			return 1, nil
		}
		return 0, fmt.Errorf("failed to open asic config file %s: %w", asicConfPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			if strings.ToLower(key) == "num_asic" {
				num, err := strconv.Atoi(value)
				if err != nil {
					return 0, fmt.Errorf("invalid num_asic value '%s': %w", value, err)
				}
				return num, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading asic config file %s: %w", asicConfPath, err)
	}

	// If num_asic is not found in the file, assume 1.
	return 1, nil
}

// IsMultiASIC checks if the device is a multi-ASIC platform.
func IsMultiASIC() (bool, error) {
	numAsics, err := GetNumASICs()
	if err != nil {
		return false, err
	}
	return numAsics > 1, nil
}

// GetAllNamespaces returns a slice of all network namespace names.
// On a single-ASIC system, it returns a slice with one empty string ""
// which represents the default (host) namespace.
func GetAllNamespaces() (*NamespacesByRole, error) {
	numAsics, err := GetNumASICs()
	if err != nil {
		return nil, err
	}

	if numAsics <= 1 {
		// Single ASIC platform, return the default namespace in the frontend role.
		return &NamespacesByRole{Frontend: []string{defaultNamespace}}, nil
	}

	// Multi-ASIC platform, discover namespaces and their roles from ConfigDB.
	namespaces := NamespacesByRole{}
	for i := 0; i < numAsics; i++ {
		ns := fmt.Sprintf("%s%d", asicNamePrefix, i)
		dbTarget := fmt.Sprintf("CONFIG_DB/%s", ns)
		queries := [][]string{{dbTarget, "DEVICE_METADATA", "localhost"}}

		if DBQuery == nil {
			fmt.Printf("Warning: DBQuery not configured; skipping namespace '%s' role detection\n", ns)
			continue
		}
		msi, err := DBQuery(queries)
		if err != nil {
			// Log warning but continue, one failing namespace shouldn't stop the whole process.
			fmt.Printf("Warning: could not get metadata for namespace '%s': %v\n", ns, err)
			continue
		}

		key := fmt.Sprintf("DEVICE_METADATA|localhost")
		entry, ok := msi[key].(map[string]interface{})
		if !ok {
			fmt.Printf("Warning: could not parse metadata for namespace '%s'\n", ns)
			continue
		}

		if subRole, ok := entry["sub_role"].(string); ok {
			switch subRole {
			case "Frontend":
				namespaces.Frontend = append(namespaces.Frontend, ns)
			case "Backend":
				namespaces.Backend = append(namespaces.Backend, ns)
			case "Fabric":
				namespaces.Fabric = append(namespaces.Fabric, ns)
			}
		}
	}

	return &namespaces, nil
}
