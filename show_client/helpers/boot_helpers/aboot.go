package helpers

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type AbootBootloader struct{}

func (a *AbootBootloader) Name() string { return "aboot" }

func (a *AbootBootloader) GetCurrentImage() (string, error) {
	cmdline, err := readProcCmdline()
	if err != nil {
		return "", err
	}
	return currentImageFromCmdline(cmdline)
}

func (a *AbootBootloader) GetInstalledImages() ([]string, error) {
	files, err := os.ReadDir(HostPath)
	if err != nil {
		return nil, err
	}

	var images []string
	for _, file := range files {
		if file.IsDir() && strings.HasPrefix(file.Name(), ImageDirPrefix) {
			image := strings.Replace(file.Name(), ImageDirPrefix, ImagePrefix, 1)
			images = append(images, image)
		}
	}

	return images, nil
}

func (a *AbootBootloader) GetNextImage() (string, error) {
	// Read boot config
	config, err := a.bootConfigRead()
	if err != nil {
		return "", err
	}

	swi := config["SWI"]
	if swi == "" {
		return "", fmt.Errorf("SWI not found in boot config")
	}

	re := regexp.MustCompile(`flash:/*(\S+)/`)
	m := re.FindStringSubmatch(swi)
	if len(m) >= 2 {
		return strings.Replace(m[1], ImageDirPrefix, ImagePrefix, 1), nil
	}

	// Fallback: swi.split(':', 1)[-1]
	parts := strings.SplitN(swi, ":", 2)
	if len(parts) == 2 {
		return parts[1], nil
	}

	return swi, nil
}

func (a *AbootBootloader) bootConfigRead() (map[string]string, error) {
	config := make(map[string]string)

	data, err := os.ReadFile(AbootBootConfigPath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			config[key] = value
		}
	}

	return config, nil
}
