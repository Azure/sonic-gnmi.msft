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

func currentImageFromCmdline(cmdline string) (string, error) {
	re := regexp.MustCompile(`loop=(\S+)/fs\.squashfs`)
	m := re.FindStringSubmatch(cmdline)
	if len(m) < 2 {
		return "", fmt.Errorf("loop mount with fs.squashfs not found in cmdline")
	}

	current := m[1]
	result := strings.Replace(current, ImageDirPrefix, ImagePrefix, 1)

	return result, nil
}
