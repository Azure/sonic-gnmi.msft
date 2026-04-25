package gnmi

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	helpers "github.com/sonic-net/sonic-gnmi/show_client/helpers/boot_helpers"
)

func TestBootHelperDetectBootloader(t *testing.T) {
	tests := []struct {
		name           string
		mockCmdline    string
		mockCmdlineErr error
		grubCfgExists  bool
		expectedType   string
		expectError    bool
	}{
		{
			name:          "Aboot detection",
			mockCmdline:   "console=ttyS0,9600 Aboot=Aboot-veos-8.0.0",
			grubCfgExists: false,
			expectedType:  "aboot",
			expectError:   false,
		},
		{
			name:          "GRUB detection",
			mockCmdline:   "BOOT_IMAGE=/vmlinuz-4.19.0-12-amd64",
			grubCfgExists: true,
			expectedType:  "grub",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			// Mock file operations for proc/cmdline
			patches.ApplyFunc(os.ReadFile, func(name string) ([]byte, error) {
				if name == "/proc/cmdline" {
					if tt.mockCmdlineErr != nil {
						return nil, tt.mockCmdlineErr
					}
					return []byte(tt.mockCmdline), nil
				}
				return nil, os.ErrNotExist
			})

			// Mock os.Stat for GRUB detection
			patches.ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				if strings.Contains(name, "grub.cfg") && tt.grubCfgExists {
					return &mockFileInfo{name: "grub.cfg", isDir: false}, nil
				}
				return nil, os.ErrNotExist
			})

			bl, err := helpers.DetectBootloader()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if bl.Name() != tt.expectedType {
				t.Errorf("Expected bootloader %s, got %s", tt.expectedType, bl.Name())
			}
		})
	}
}

// Mock FileInfo for testing
type mockFileInfo struct {
	name  string
	isDir bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0 }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

func TestBootHelperAbootBootloader(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Mock file operations for simplified Aboot implementation
	patches.ApplyFunc(os.ReadFile, func(name string) ([]byte, error) {
		if name == "/proc/cmdline" {
			return []byte("loop=image-20240101.01/fs.squashfs"), nil
		}
		if name == "/host/boot-config" {
			return []byte("# Boot configuration\nSWI=flash:/image-20240201.01/sonic.swi\n"), nil
		}
		return nil, os.ErrNotExist
	})

	patches.ApplyFunc(os.ReadDir, func(name string) ([]os.DirEntry, error) {
		if name == "/host" {
			return []os.DirEntry{
				&mockDirEntry{name: "image-20240101.01", isDir: true},
				&mockDirEntry{name: "image-20240201.01", isDir: true},
			}, nil
		}
		return nil, os.ErrNotExist
	})

	bl := &helpers.AbootBootloader{}

	// Test Name()
	if bl.Name() != "aboot" {
		t.Errorf("Expected name 'aboot', got %q", bl.Name())
	}

	// Test GetCurrentImage()
	current, err := bl.GetCurrentImage()
	if err != nil {
		t.Fatalf("GetCurrentImage error: %v", err)
	}
	expected := "SONiC-OS-20240101.01"
	if current != expected {
		t.Errorf("Expected current image %q, got %q", expected, current)
	}

	// Test GetInstalledImages()
	images, err := bl.GetInstalledImages()
	if err != nil {
		t.Fatalf("GetInstalledImages error: %v", err)
	}
	if len(images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(images))
	}

	// Test GetNextImage()
	next, err := bl.GetNextImage()
	if err != nil {
		t.Fatalf("GetNextImage error: %v", err)
	}
	expected = "SONiC-OS-20240201.01"
	if next != expected {
		t.Errorf("Expected next image %q, got %q", expected, next)
	}
}

type mockDirEntry struct {
	name  string
	isDir bool
}

func (m *mockDirEntry) Name() string               { return m.name }
func (m *mockDirEntry) IsDir() bool                { return m.isDir }
func (m *mockDirEntry) Type() os.FileMode          { return 0 }
func (m *mockDirEntry) Info() (os.FileInfo, error) { return nil, nil }

func TestBootHelperGrubBootloader(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Mock file operations for simplified GRUB implementation
	patches.ApplyFunc(os.ReadFile, func(name string) ([]byte, error) {
		if name == "/proc/cmdline" {
			return []byte("loop=image-grub-test.01/fs.squashfs"), nil
		}
		if strings.Contains(name, "grub.cfg") {
			return []byte(`
menuentry 'SONiC-OS-20240101.01' {
    linux /vmlinuz
}
menuentry 'SONiC-OS-20240201.01' {
    linux /vmlinuz
}
`), nil
		}
		return nil, os.ErrNotExist
	})

	// Mock common.GetDataFromHostCommand for grub-editenv
	patches.ApplyFunc(common.GetDataFromHostCommand, func(command string) (string, error) {
		if strings.Contains(command, "grub-editenv") && strings.Contains(command, "list") {
			return "next_entry=1\nsaved_entry=0\n", nil
		}
		return "", os.ErrNotExist
	})

	bl := &helpers.GrubBootloader{}

	// Test Name()
	if bl.Name() != "grub" {
		t.Errorf("Expected name 'grub', got %q", bl.Name())
	}

	// Test GetCurrentImage()
	current, err := bl.GetCurrentImage()
	if err != nil {
		t.Fatalf("GetCurrentImage error: %v", err)
	}
	expected := "SONiC-OS-grub-test.01"
	if current != expected {
		t.Errorf("Expected current image %q, got %q", expected, current)
	}

	// Test GetInstalledImages()
	images, err := bl.GetInstalledImages()
	if err != nil {
		t.Fatalf("GetInstalledImages error: %v", err)
	}
	if len(images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(images))
	}

	// Test GetNextImage()
	next, err := bl.GetNextImage()
	if err != nil {
		t.Fatalf("GetNextImage error: %v", err)
	}
	expected = "SONiC-OS-20240201.01"
	if next != expected {
		t.Errorf("Expected next image %q, got %q", expected, next)
	}
}

func TestBootHelperUbootBootloader(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Mock file operations for /proc/cmdline
	patches.ApplyFunc(os.ReadFile, func(name string) ([]byte, error) {
		if name == "/proc/cmdline" {
			return []byte("loop=image-uboot-test.01/fs.squashfs"), nil
		}
		return nil, os.ErrNotExist
	})

	// Mock common.GetDataFromHostCommand for fw_printenv
	patches.ApplyFunc(common.GetDataFromHostCommand, func(command string) (string, error) {
		if strings.Contains(command, "fw_printenv") {
			if strings.Contains(command, "sonic_version_1") {
				return "SONiC-OS-20240101.01", nil
			}
			if strings.Contains(command, "sonic_version_2") {
				return "SONiC-OS-20240201.01", nil
			}
			if strings.Contains(command, "boot_next") {
				return "sonic_image_2", nil
			}
		}
		return "", os.ErrNotExist
	})

	bl := &helpers.UbootBootloader{}

	// Test Name()
	if bl.Name() != "uboot" {
		t.Errorf("Expected name 'uboot', got %q", bl.Name())
	}

	// Test GetCurrentImage()
	current, err := bl.GetCurrentImage()
	if err != nil {
		t.Fatalf("GetCurrentImage error: %v", err)
	}
	expected := "SONiC-OS-uboot-test.01"
	if current != expected {
		t.Errorf("Expected current image %q, got %q", expected, current)
	}

	// Test GetInstalledImages()
	images, err := bl.GetInstalledImages()
	if err != nil {
		t.Fatalf("GetInstalledImages error: %v", err)
	}
	if len(images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(images))
	}

	// Test GetNextImage()
	next, err := bl.GetNextImage()
	if err != nil {
		t.Fatalf("GetNextImage error: %v", err)
	}
	expected = "SONiC-OS-20240201.01"
	if next != expected {
		t.Errorf("Expected next image %q, got %q", expected, next)
	}
}

func TestBootHelperIntegration(t *testing.T) {
	// Test that DetectBootloader returns valid interface in current environment
	if runtime.GOARCH == "amd64" {
		_, err := helpers.DetectBootloader()
		// We expect an error in test environment, which is fine
		if err != nil {
			t.Logf("Expected error in test environment: %v", err)
		}
	}
}
