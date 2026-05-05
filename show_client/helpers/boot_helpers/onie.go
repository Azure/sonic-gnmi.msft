package helpers

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func readProcCmdline() (string, error) {
	data, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func getCurrentImageFromCmdline(regexPattern string) (string, error) {
	cmdline, err := readProcCmdline()
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(regexPattern)
	m := re.FindStringSubmatch(cmdline)
	if len(m) < 2 {
		return "", fmt.Errorf("loop mount pattern not found in cmdline")
	}

	current := m[1]
	result := strings.Replace(current, ImageDirPrefix, ImagePrefix, 1)

	return result, nil
}
