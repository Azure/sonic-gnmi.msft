package helpers

import (
	"strings"

	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

type UbootBootloader struct{}

func (u *UbootBootloader) Name() string { return "uboot" }

func (u *UbootBootloader) GetCurrentImage() (string, error) {
	cmdline, err := readProcCmdline()
	if err != nil {
		return "", err
	}
	return currentImageFromCmdline(cmdline)
}

func (u *UbootBootloader) GetInstalledImages() ([]string, error) {
	var images []string

	if output, err := common.GetDataFromHostCommand("/usr/bin/fw_printenv -n sonic_version_1"); err == nil {
		image := strings.TrimSpace(output)
		if strings.Contains(image, ImagePrefix) {
			images = append(images, image)
		}
	}

	if output, err := common.GetDataFromHostCommand("/usr/bin/fw_printenv -n sonic_version_2"); err == nil {
		image := strings.TrimSpace(output)
		if strings.Contains(image, ImagePrefix) {
			images = append(images, image)
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
