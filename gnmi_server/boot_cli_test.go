package gnmi

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/agiledragon/gomonkey/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	"github.com/sonic-net/sonic-gnmi/show_client/helpers/boot_helpers"
)

// Mock bootloader for testing
type mockBootloader struct {
	name            string
	currentImage    string
	nextImage       string
	installedImages []string
	currentErr      error
	nextErr         error
	installedErr    error
}

func (m *mockBootloader) Name() string {
	return m.name
}

func (m *mockBootloader) GetCurrentImage() (string, error) {
	return m.currentImage, m.currentErr
}

func (m *mockBootloader) GetNextImage() (string, error) {
	return m.nextImage, m.nextErr
}

func (m *mockBootloader) GetInstalledImages() ([]string, error) {
	return m.installedImages, m.installedErr
}

func TestGetShowBoot(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}

	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dialing to %q failed: %v", TargetAddr, err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal []byte
		valTest     bool
		setupFunc   func() *gomonkey.Patches
	}{
		{
			desc:       "query SHOW boot success",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "boot" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"current":"SONiC-20240101.01","next":"SONiC-20240201.01","available":["SONiC-20240101.01","SONiC-20240201.01","SONiC-20240301.01"]}`),
			valTest:     true,
			setupFunc: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				mockBL := &mockBootloader{
					name:            "grub",
					currentImage:    "SONiC-20240101.01",
					nextImage:       "SONiC-20240201.01",
					installedImages: []string{"SONiC-20240101.01", "SONiC-20240201.01", "SONiC-20240301.01"},
				}
				patches.ApplyFunc(helpers.DetectBootloader, func() (helpers.Bootloader, error) {
					return mockBL, nil
				})
				return patches
			},
		},
		{
			desc:       "query SHOW boot with empty installed images",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "boot" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"current":"SONiC-20240101.01","next":"SONiC-20240201.01","available":[]}`),
			valTest:     true,
			setupFunc: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				mockBL := &mockBootloader{
					name:            "aboot",
					currentImage:    "SONiC-20240101.01",
					nextImage:       "SONiC-20240201.01",
					installedImages: nil, // Should be converted to empty array
				}
				patches.ApplyFunc(helpers.DetectBootloader, func() (helpers.Bootloader, error) {
					return mockBL, nil
				})
				return patches
			},
		},
		{
			desc:       "query SHOW boot with uboot",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "boot" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"current":"SONiC-uboot.01","next":"SONiC-uboot.02","available":["SONiC-uboot.01","SONiC-uboot.02"]}`),
			valTest:     true,
			setupFunc: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				mockBL := &mockBootloader{
					name:            "uboot",
					currentImage:    "SONiC-uboot.01",
					nextImage:       "SONiC-uboot.02",
					installedImages: []string{"SONiC-uboot.01", "SONiC-uboot.02"},
				}
				patches.ApplyFunc(helpers.DetectBootloader, func() (helpers.Bootloader, error) {
					return mockBL, nil
				})
				return patches
			},
		},
		{
			desc:       "query SHOW boot detect bootloader error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "boot" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			setupFunc: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				patches.ApplyFunc(helpers.DetectBootloader, func() (helpers.Bootloader, error) {
					return nil, errors.New("no supported bootloader detected")
				})
				return patches
			},
		},
		{
			desc:       "query SHOW boot get current image error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "boot" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			setupFunc: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				mockBL := &mockBootloader{
					name:       "grub",
					currentErr: errors.New("failed to get current image"),
				}
				patches.ApplyFunc(helpers.DetectBootloader, func() (helpers.Bootloader, error) {
					return mockBL, nil
				})
				return patches
			},
		},
		{
			desc:       "query SHOW boot get next image error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "boot" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			setupFunc: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				mockBL := &mockBootloader{
					name:         "grub",
					currentImage: "SONiC-20240101.01",
					nextErr:      errors.New("failed to get next image"),
				}
				patches.ApplyFunc(helpers.DetectBootloader, func() (helpers.Bootloader, error) {
					return mockBL, nil
				})
				return patches
			},
		},
		{
			desc:       "query SHOW boot get installed images error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "boot" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     false,
			setupFunc: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				mockBL := &mockBootloader{
					name:         "grub",
					currentImage: "SONiC-20240101.01",
					nextImage:    "SONiC-20240201.01",
					installedErr: errors.New("failed to get installed images"),
				}
				patches.ApplyFunc(helpers.DetectBootloader, func() (helpers.Bootloader, error) {
					return mockBL, nil
				})
				return patches
			},
		},
		{
			desc:       "query SHOW boot with special characters",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "boot" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"current":"SONiC-OS.2024-01-01","next":"SONiC-OS.2024-02-01","available":["SONiC-OS.2024-01-01","SONiC-OS.2024-02-01"]}`),
			valTest:     true,
			setupFunc: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				mockBL := &mockBootloader{
					name:            "grub",
					currentImage:    "SONiC-OS.2024-01-01",
					nextImage:       "SONiC-OS.2024-02-01",
					installedImages: []string{"SONiC-OS.2024-01-01", "SONiC-OS.2024-02-01"},
				}
				patches.ApplyFunc(helpers.DetectBootloader, func() (helpers.Bootloader, error) {
					return mockBL, nil
				})
				return patches
			},
		},
		{
			desc:       "query SHOW boot with single image",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "boot" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"current":"SONiC-single","next":"SONiC-single","available":["SONiC-single"]}`),
			valTest:     true,
			setupFunc: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				mockBL := &mockBootloader{
					name:            "grub",
					currentImage:    "SONiC-single",
					nextImage:       "SONiC-single",
					installedImages: []string{"SONiC-single"},
				}
				patches.ApplyFunc(helpers.DetectBootloader, func() (helpers.Bootloader, error) {
					return mockBL, nil
				})
				return patches
			},
		},
		{
			desc:       "query SHOW boot with large image list",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "boot" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"current":"SONiC-20240101.00","next":"SONiC-20240101.01","available":["SONiC-20240101.00","SONiC-20240101.01","SONiC-20240101.02","SONiC-20240101.03","SONiC-20240101.04"]}`),
			valTest:     true,
			setupFunc: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				// Create a list of 5 images for testing
				imageList := make([]string, 5)
				for i := 0; i < 5; i++ {
					imageList[i] = fmt.Sprintf("SONiC-20240101.%02d", i)
				}
				mockBL := &mockBootloader{
					name:            "grub",
					currentImage:    "SONiC-20240101.00",
					nextImage:       "SONiC-20240101.01",
					installedImages: imageList,
				}
				patches.ApplyFunc(helpers.DetectBootloader, func() (helpers.Bootloader, error) {
					return mockBL, nil
				})
				return patches
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var patches *gomonkey.Patches
			if test.setupFunc != nil {
				patches = test.setupFunc()
				defer patches.Reset()
			}
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
	}
}

// Test edge cases and bootloader-specific behaviors
func TestGetShowBootEdgeCases(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}

	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dialing to %q failed: %v", TargetAddr, err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	tests := []struct {
		desc        string
		pathTarget  string
		textPbPath  string
		wantRetCode codes.Code
		wantRespVal []byte
		valTest     bool
		setupFunc   func() *gomonkey.Patches
	}{
		{
			desc:       "query SHOW boot with empty strings",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "boot" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"current":"","next":"","available":[]}`),
			valTest:     true,
			setupFunc: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				mockBL := &mockBootloader{
					name:            "test",
					currentImage:    "",
					nextImage:       "",
					installedImages: []string{},
				}
				patches.ApplyFunc(helpers.DetectBootloader, func() (helpers.Bootloader, error) {
					return mockBL, nil
				})
				return patches
			},
		},
		{
			desc:       "query SHOW boot with very long image names",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "boot" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(`{"current":"SONiC-very-long-image-name-with-multiple-components-2024.01.01.build.123456","next":"SONiC-very-long-image-name-with-multiple-components-2024.02.01.build.234567","available":["SONiC-very-long-image-name-with-multiple-components-2024.01.01.build.123456","SONiC-very-long-image-name-with-multiple-components-2024.02.01.build.234567"]}`),
			valTest:     true,
			setupFunc: func() *gomonkey.Patches {
				patches := gomonkey.NewPatches()
				mockBL := &mockBootloader{
					name:            "grub",
					currentImage:    "SONiC-very-long-image-name-with-multiple-components-2024.01.01.build.123456",
					nextImage:       "SONiC-very-long-image-name-with-multiple-components-2024.02.01.build.234567",
					installedImages: []string{"SONiC-very-long-image-name-with-multiple-components-2024.01.01.build.123456", "SONiC-very-long-image-name-with-multiple-components-2024.02.01.build.234567"},
				}
				patches.ApplyFunc(helpers.DetectBootloader, func() (helpers.Bootloader, error) {
					return mockBL, nil
				})
				return patches
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var patches *gomonkey.Patches
			if test.setupFunc != nil {
				patches = test.setupFunc()
				defer patches.Reset()
			}
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
	}
}

// Test different bootloader types
func TestGetShowBootDifferentBootloaders(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()
	defer ResetDataSetsAndMappings(t)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}

	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dialing to %q failed: %v", TargetAddr, err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	bootloaderTests := []struct {
		name            string
		bootloaderName  string
		currentImage    string
		nextImage       string
		installedImages []string
		expectedJSON    string
	}{
		{
			name:            "Aboot",
			bootloaderName:  "aboot",
			currentImage:    "SONiC-aboot.01",
			nextImage:       "SONiC-aboot.02",
			installedImages: []string{"SONiC-aboot.01", "SONiC-aboot.02"},
			expectedJSON:    `{"current":"SONiC-aboot.01","next":"SONiC-aboot.02","available":["SONiC-aboot.01","SONiC-aboot.02"]}`,
		},
		{
			name:            "GRUB",
			bootloaderName:  "grub",
			currentImage:    "SONiC-grub.01",
			nextImage:       "SONiC-grub.02",
			installedImages: []string{"SONiC-grub.01", "SONiC-grub.02"},
			expectedJSON:    `{"current":"SONiC-grub.01","next":"SONiC-grub.02","available":["SONiC-grub.01","SONiC-grub.02"]}`,
		},
		{
			name:            "U-Boot",
			bootloaderName:  "uboot",
			currentImage:    "SONiC-uboot.01",
			nextImage:       "SONiC-uboot.02",
			installedImages: []string{"SONiC-uboot.01", "SONiC-uboot.02"},
			expectedJSON:    `{"current":"SONiC-uboot.01","next":"SONiC-uboot.02","available":["SONiC-uboot.01","SONiC-uboot.02"]}`,
		},
	}

	for _, bt := range bootloaderTests {
		t.Run("SHOW boot with "+bt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			mockBL := &mockBootloader{
				name:            bt.bootloaderName,
				currentImage:    bt.currentImage,
				nextImage:       bt.nextImage,
				installedImages: bt.installedImages,
			}

			patches.ApplyFunc(helpers.DetectBootloader, func() (helpers.Bootloader, error) {
				return mockBL, nil
			})

			runTestGet(t, ctx, gClient, "SHOW", `elem: <name: "boot" >`, codes.OK, []byte(bt.expectedJSON), true)
		})
	}
}
