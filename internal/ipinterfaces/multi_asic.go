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
	asicNamePrefix        = "asic"
	defaultNamespace      = ""
	machineConfPath       = "/host/machine.conf"
	containerPlatformPath = "/usr/share/sonic/platform"
	hostDevicePath        = "/usr/share/sonic/device"
	asicConfFilename      = "asic.conf"
)

// getPlatform retrieves the device's platform identifier by checking the environment,
// machine.conf, and the config DB in that order.
func getPlatform(dbQuery DBQueryFunc) string {
	if platform := getPlatformFromEnv(); platform != "" {
		return platform
	}

	if platform, err := getPlatformFromMachineConf(); err == nil && platform != "" {
		return platform
	}

	if platform, err := getPlatformFromConfigDB(dbQuery); err == nil && platform != "" {
		return platform
	}

	return ""
}

// getPlatformFromEnv reads the platform from the "PLATFORM" environment variable.
func getPlatformFromEnv() string {
	return os.Getenv("PLATFORM")
}

// getPlatformFromMachineConf reads the platform from onie_platform or aboot_platform in machine.conf.
func getPlatformFromMachineConf() (string, error) {
	file, err := os.Open(machineConfPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			if key == "onie_platform" || key == "aboot_platform" {
				return value, nil
			}
		}
	}
	return "", scanner.Err()
}

// getPlatformFromConfigDB reads the platform from DEVICE_METADATA in the ConfigDB.
func getPlatformFromConfigDB(dbQuery DBQueryFunc) (string, error) {
	if dbQuery == nil {
		return "", nil // Not an error, just unavailable.
	}
	queries := [][]string{{"CONFIG_DB", "DEVICE_METADATA", "localhost"}}
	msi, err := dbQuery(queries)
	if err != nil {
		return "", err
	}
	if msi == nil {
		return "", nil
	}
	entry, ok := msi["DEVICE_METADATA|localhost"].(map[string]interface{})
	if !ok {
		return "", nil
	}
	if platform, ok := entry["platform"].(string); ok {
		return platform, nil
	}
	return "", nil
}

// getAsicConfFilePath retrieves the path to the ASIC configuration file.
// This is a port of the logic from sonic_py_common/device_info.py
func getAsicConfFilePath(dbQuery DBQueryFunc) string {
	// Candidate 1: /usr/share/sonic/platform/asic.conf
	candidate1 := filepath.Join(containerPlatformPath, asicConfFilename)
	if _, err := os.Stat(candidate1); err == nil {
		return candidate1
	}

	// Candidate 2: /usr/share/sonic/device/<platform>/asic.conf
	platform := getPlatform(dbQuery)
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
func GetNumASICs(dbQuery DBQueryFunc) (int, error) {
	asicConfPath := getAsicConfFilePath(dbQuery)
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
func IsMultiASIC(dbQuery DBQueryFunc) (bool, error) {
	numAsics, err := GetNumASICs(dbQuery)
	if err != nil {
		return false, err
	}
	return numAsics > 1, nil
}

// GetAllNamespaces returns a slice of all network namespace names.
// On a single-ASIC system, it returns a slice with one empty string ""
// which represents the default (host) namespace.

func GetAllNamespaces(logger Logger, dbQuery DBQueryFunc) (*NamespacesByRole, error) {
	numAsics, err := GetNumASICs(dbQuery)
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

		if dbQuery == nil {
			logger.Warnf("DBQuery not configured; skipping namespace '%s' role detection", ns)
			continue
		}
		msi, err := dbQuery(queries)
		if err != nil {
			// Log warning but continue, one failing namespace shouldn't stop the whole process.
			logger.Warnf("could not get metadata for namespace '%s': %v", ns, err)
			continue
		}

		key := "DEVICE_METADATA|localhost"
		entry, ok := msi[key].(map[string]interface{})
		if !ok {
			logger.Warnf("could not parse metadata for namespace '%s'", ns)
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
