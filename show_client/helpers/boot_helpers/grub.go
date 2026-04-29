package helpers

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

type GrubBootloader struct{}

func (g *GrubBootloader) Name() string { return "grub" }

func (g *GrubBootloader) GetCurrentImage() (string, error) {
	return getCurrentImageFromCmdline(`loop=(\S+)/fs\.squashfs`)
}

func (g *GrubBootloader) GetInstalledImages() ([]string, error) {
	data, err := os.ReadFile(GrubCfgPath)
	if err != nil {
		return nil, err
	}

	var images []string
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "menuentry") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				image := strings.Trim(parts[1], "'\"")
				if strings.Contains(image, ImagePrefix) {
					images = append(images, image)
				}
			}
		}
	}

	return images, nil
}

func (g *GrubBootloader) GetNextImage() (string, error) {
	images, err := g.GetInstalledImages()
	if err != nil {
		return "", err
	}

	if len(images) == 0 {
		return "", fmt.Errorf("no installed images found")
	}

	command := fmt.Sprintf("/usr/bin/grub-editenv %s list", GrubEnvPath)
	output, err := common.GetDataFromHostCommand(command)
	if err != nil {
		return images[0], nil
	}

	nextImageIndex := 0

	re := regexp.MustCompile(`next_entry=(\d+)`)
	if m := re.FindStringSubmatch(output); len(m) >= 2 {
		if idx, err := strconv.Atoi(m[1]); err == nil {
			nextImageIndex = idx
		}
	} else {
		re = regexp.MustCompile(`saved_entry=(\d+)`)
		if m := re.FindStringSubmatch(output); len(m) >= 2 {
			if idx, err := strconv.Atoi(m[1]); err == nil {
				nextImageIndex = idx
			}
		}
	}

	if nextImageIndex < 0 || nextImageIndex >= len(images) {
		nextImageIndex = 0
	}

	return images[nextImageIndex], nil
}
