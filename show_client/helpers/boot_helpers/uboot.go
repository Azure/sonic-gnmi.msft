package helpers

import (
	"fmt"
	"strings"

	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

type UbootBootloader struct{}

func (u *UbootBootloader) Name() string { return "uboot" }

func (u *UbootBootloader) GetCurrentImage() (string, error) {
	return getCurrentImageFromCmdline(`loop=(\S+)/fs\.squashfs`)
}

func (u *UbootBootloader) GetInstalledImages() ([]string, error) {
	var images []string

	for i := 1; i <= 2; i++ {
		cmd := fmt.Sprintf("/usr/bin/fw_printenv -n sonic_version_%d", i)
		if output, err := common.GetDataFromHostCommand(cmd); err == nil {
			image := strings.TrimSpace(output)
			if strings.Contains(image, ImagePrefix) {
				images = append(images, image)
			}
		}
	}

	return images, nil
}

func (u *UbootBootloader) GetNextImage() (string, error) {
	images, err := u.GetInstalledImages()
	if err != nil {
		return "", err
	}

	output, err := common.GetDataFromHostCommand("/usr/bin/fw_printenv -n boot_next")
	if err != nil {
		if len(images) > 0 {
			return images[0], nil
		}
		return "", err
	}

	bootNext := strings.TrimSpace(output)
	if strings.Contains(bootNext, "sonic_image_2") && len(images) == 2 {
		return images[1], nil
	}

	if len(images) > 0 {
		return images[0], nil
	}

	return "", nil
}
