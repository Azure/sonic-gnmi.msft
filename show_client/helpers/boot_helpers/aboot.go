package helpers

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

type AbootBootloader struct{}

func (a *AbootBootloader) Name() string { return "aboot" }

func (a *AbootBootloader) GetCurrentImage() (string, error) {
	// Use aboot-specific regex pattern
	return getCurrentImageFromCmdline(`loop=/*(\S+)/`)
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
	configData, err := common.ReadConfToMap(AbootBootConfigPath)
	if err != nil {
		return "", err
	}

	swiInterface, exists := configData["SWI"]
	if !exists {
		return "", fmt.Errorf("SWI not found in boot config")
	}

	swi, ok := swiInterface.(string)
	if !ok {
		return "", fmt.Errorf("SWI value is not a string")
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
