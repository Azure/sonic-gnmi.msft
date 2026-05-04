package helpers

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Shared constants
const (
	HostPath       = "/host"
	ImageDirPrefix = "image-"
	ImagePrefix    = "SONiC-OS-"

	// Aboot
	AbootBootConfigPath = "/host/boot-config"

	// GRUB
	GrubCfgPath = "/host/grub/grub.cfg"
	GrubEnvPath = "/host/grub/grubenv"
)

type Bootloader interface {
	Name() string
	GetCurrentImage() (string, error)
	GetNextImage() (string, error)
	GetInstalledImages() ([]string, error)
}

func DetectBootloader() (Bootloader, error) {
	// 1. Check for Aboot
	cmdline, err := readProcCmdline()
	if err == nil && strings.Contains(cmdline, "Aboot=") {
		return &AbootBootloader{}, nil
	}

	// 2. Check for GRUB
	if _, err := os.Stat(GrubCfgPath); err == nil {
		return &GrubBootloader{}, nil
	}

	// 3. Check for U-Boot
	if runtime.GOARCH == "arm" || runtime.GOARCH == "arm64" {
		return &UbootBootloader{}, nil
	}

	return nil, fmt.Errorf("No supported bootloader detected")
}
