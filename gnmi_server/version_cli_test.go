package gnmi

// intf_cli_test.go

// Tests SHOW interface errors

import (
	"crypto/tls"
	"errors"
	"os"
	"testing"
	"time"

	pb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/agiledragon/gomonkey/v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"

	show_client "github.com/sonic-net/sonic-gnmi/show_client"
)

func saveEnv(key string) func() {
	originalValue, exists := os.LookupEnv(key)
	return func() {
		if exists {
			os.Setenv(key, originalValue)
		} else {
			os.Unsetenv(key)
		}
	}
}

func TestGetShowVersionWithoutEnv(t *testing.T) {
	cleanup := saveEnv("PLATFORM")
	t.Cleanup(cleanup)

	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()

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

	// Mock interface error data with some errors present
	versionInfo := `
build_version: test_branch.1-a8fbac59d
debian_version: 11.4
kernel_version: 5.10.0-18-2-amd64
asic_type: mellanox
asic_subtype: mellanox
commit_id: a8fbac59d
branch: test_branch
release: master
libswsscommon: 1.0.0
sonic_utilities: 1.2
`
	deviceMetadataFilename := "../testdata/VERSION_METADATA.txt"
	chassisDataFilename := "../testdata/VERSION_CHASSIS.txt"

	// Mock interface error data with no errors (all zeros)
	expectedOutputWithEmptyPlat := `
{
  "sonic_software_version": "SONiC.test_branch.1-a8fbac59d",
  "sonic_os_version": "\u003cnil\u003e",
  "distribution": "Debian 11.4",
  "kernel": "5.10.0-18-2-amd64",
  "build_commit": "a8fbac59d",
  "build_date": "\u003cnil\u003e",
  "built_by": "\u003cnil\u003e",
  "platform": "test_onie_platform",
  "hwsku": "",
  "asic": "mellanox",
  "asic_count": "N/A",
  "serial_number": "",
  "model_number": "",
  "hardware_revision": "",
  "uptime": "07:42:51 up 16 days, 14:51,  2 users,  load average: 0.00, 0.00, 0.00",
  "date": "Fri 18 Jul 2025 18:00:00",
  "docker_info": "[map[ID:sha256:372601148371578e886d545864469deca0ad8db5618e10ae6124429637f86a02 Repository:docker-mux Size:366MB Tag:20250510.14] map[ID:sha256:372601148371578e886d545864469deca0ad8db5618e10ae6124429637f86a02 Repository:docker-mux Size:366MB Tag:latest] map[ID:sha256:e035d4584c6c6bd7a48430abbba01e91737022f9ec42749af3de1f38751204af Repository:docker-dhcp-server Size:337MB Tag:20250510.14] map[ID:sha256:e035d4584c6c6bd7a48430abbba01e91737022f9ec42749af3de1f38751204af Repository:docker-dhcp-server Size:337MB Tag:latest] map[ID:sha256:4f8bfd4cbdd732ae3891bc34b49adf920804ec62eacc234917eefb0f6dfbe2f3 Repository:docker-macsec Size:347MB Tag:20250510.14] map[ID:sha256:4f8bfd4cbdd732ae3891bc34b49adf920804ec62eacc234917eefb0f6dfbe2f3 Repository:docker-macsec Size:347MB Tag:latest] map[ID:sha256:f33ca825cb610d1e0ab12f5d8c9a3b39b11c8e9945386c79fd36816d7441bfda Repository:docker-eventd Size:314MB Tag:20250510.14] map[ID:sha256:f33ca825cb610d1e0ab12f5d8c9a3b39b11c8e9945386c79fd36816d7441bfda Repository:docker-eventd Size:314MB Tag:latest] map[ID:sha256:8884f25155d4ba354712b6f21510cf75e0d0e7401cb2a61609eb940a4629b178 Repository:docker-sonic-telemetry Size:408MB Tag:20250510.14] map[ID:sha256:8884f25155d4ba354712b6f21510cf75e0d0e7401cb2a61609eb940a4629b178 Repository:docker-sonic-telemetry Size:408MB Tag:latest] map[ID:sha256:3922a17378ed1caf1b5c85c07fd73e5f8e7e85ad4a3c690e1c02cc3e4dae4c7b Repository:docker-gbsyncd-broncos Size:367MB Tag:20250510.14] map[ID:sha256:3922a17378ed1caf1b5c85c07fd73e5f8e7e85ad4a3c690e1c02cc3e4dae4c7b Repository:docker-gbsyncd-broncos Size:367MB Tag:latest] map[ID:sha256:5bf1e2f6561ebd6b5e84d91caab6a18776c1fa1be55b0d915d244cc5f528aa88 Repository:docker-gbsyncd-credo Size:341MB Tag:20250510.14] map[ID:sha256:5bf1e2f6561ebd6b5e84d91caab6a18776c1fa1be55b0d915d244cc5f528aa88 Repository:docker-gbsyncd-credo Size:341MB Tag:latest] map[ID:sha256:4c777e17558bded5ab09152c29d59f95073bca928a90c1cfacb9736d900daf72 Repository:docker-sonic-gnmi Size:408MB Tag:20250510.14] map[ID:sha256:4c777e17558bded5ab09152c29d59f95073bca928a90c1cfacb9736d900daf72 Repository:docker-sonic-gnmi Size:408MB Tag:latest] map[ID:sha256:44d4ca23b0f52ae9a056bea6f87e05a20de9e0677386e9af381252ae282cfc85 Repository:docker-orchagent Size:356MB Tag:20250510.14] map[ID:sha256:44d4ca23b0f52ae9a056bea6f87e05a20de9e0677386e9af381252ae282cfc85 Repository:docker-orchagent Size:356MB Tag:latest] map[ID:sha256:1d1ce9b15c8ca8bf5b1c8ccee1b9076ccdc5c52e5cd60b0c525d7edb75d94627 Repository:docker-dhcp-relay Size:325MB Tag:20250510.14] map[ID:sha256:1d1ce9b15c8ca8bf5b1c8ccee1b9076ccdc5c52e5cd60b0c525d7edb75d94627 Repository:docker-dhcp-relay Size:325MB Tag:latest] map[ID:sha256:6433be677e7a8aef06842a4f555ed0ddac0c30487e041abf08c206cb9db6dfff Repository:docker-fpm-frr Size:386MB Tag:20250510.14] map[ID:sha256:6433be677e7a8aef06842a4f555ed0ddac0c30487e041abf08c206cb9db6dfff Repository:docker-fpm-frr Size:386MB Tag:latest] map[ID:sha256:9c8ff651cc4c6c30b2f091fda457e464262cd2c39be9cb1576176d330f381e25 Repository:docker-syncd-brcm Size:799MB Tag:20250510.14] map[ID:sha256:9c8ff651cc4c6c30b2f091fda457e464262cd2c39be9cb1576176d330f381e25 Repository:docker-syncd-brcm Size:799MB Tag:latest] map[ID:sha256:49eb5cc18cff4f5c1a3f8594ed12981838fc60a62efc9ebcfcb31afc94d7b9a7 Repository:docker-dash-ha Size:357MB Tag:20250510.14] map[ID:sha256:49eb5cc18cff4f5c1a3f8594ed12981838fc60a62efc9ebcfcb31afc94d7b9a7 Repository:docker-dash-ha Size:357MB Tag:latest] map[ID:sha256:9b738a9e879a527849c16417862644691d84e5c8a128c51a5c9753b5e4e98252 Repository:docker-snmp Size:358MB Tag:20250510.14] map[ID:sha256:9b738a9e879a527849c16417862644691d84e5c8a128c51a5c9753b5e4e98252 Repository:docker-snmp Size:358MB Tag:latest] map[ID:sha256:071a45740d3b657b90665c8028e39687c89f41310a6b345dd395f27bcf7bb618 Repository:docker-platform-monitor Size:446MB Tag:20250510.14] map[ID:sha256:071a45740d3b657b90665c8028e39687c89f41310a6b345dd395f27bcf7bb618 Repository:docker-platform-monitor Size:446MB Tag:latest] map[ID:sha256:af98df3b20e54deb0039e923889f653bb320f41115d39dde9940ed5b2c2a8f7d Repository:docker-sysmgr Size:326MB Tag:20250510.14] map[ID:sha256:af98df3b20e54deb0039e923889f653bb320f41115d39dde9940ed5b2c2a8f7d Repository:docker-sysmgr Size:326MB Tag:latest] map[ID:sha256:9ee9b197541dc46f68c42512f4cb0d2f84114b95c8d57e09405bc8663089474f Repository:docker-teamd Size:344MB Tag:20250510.14] map[ID:sha256:9ee9b197541dc46f68c42512f4cb0d2f84114b95c8d57e09405bc8663089474f Repository:docker-teamd Size:344MB Tag:latest] map[ID:sha256:32b732da2119047c2c38c783e971df3ad60c8076730a86d34d83fe85e9b8ae65 Repository:docker-router-advertiser Size:314MB Tag:20250510.14] map[ID:sha256:32b732da2119047c2c38c783e971df3ad60c8076730a86d34d83fe85e9b8ae65 Repository:docker-router-advertiser Size:314MB Tag:latest] map[ID:sha256:44c6b595de678dc283e31c43abfbc73e45985f243b219d80a3e9ca2a06ac579c Repository:docker-sonic-restapi Size:333MB Tag:20250510.14] map[ID:sha256:44c6b595de678dc283e31c43abfbc73e45985f243b219d80a3e9ca2a06ac579c Repository:docker-sonic-restapi Size:333MB Tag:latest] map[ID:sha256:20f8142edc812a92fa9c563774b96c99fd8b95e23dc682be6984f83122382735 Repository:docker-gnmi-watchdog Size:314MB Tag:20250510.14] map[ID:sha256:20f8142edc812a92fa9c563774b96c99fd8b95e23dc682be6984f83122382735 Repository:docker-gnmi-watchdog Size:314MB Tag:latest] map[ID:sha256:333901845f00aaf8a4cd335584dc5fdea589aa7a9722ecb0b28e615215ce2d80 Repository:docker-lldp Size:360MB Tag:20250510.14] map[ID:sha256:333901845f00aaf8a4cd335584dc5fdea589aa7a9722ecb0b28e615215ce2d80 Repository:docker-lldp Size:360MB Tag:latest] map[ID:sha256:ea4d44010ec1ca9a5129842aebe5a36d177c51a585c4c54e83bd6c4325492af9 Repository:docker-database Size:321MB Tag:20250510.14] map[ID:sha256:ea4d44010ec1ca9a5129842aebe5a36d177c51a585c4c54e83bd6c4325492af9 Repository:docker-database Size:321MB Tag:latest] map[ID:sha256:4411bd9004debb98ad3329ccd50bb7fceee312b80e8fd5ba23db9f1017732f0c Repository:docker-dummyk8s Size:314MB Tag:20250510.14] map[ID:sha256:4411bd9004debb98ad3329ccd50bb7fceee312b80e8fd5ba23db9f1017732f0c Repository:docker-dummyk8s Size:314MB Tag:latest] map[ID:sha256:0253e833a6cb4472e2dfa5abb93a742e3d8a6b54c14c8d8730e1b8a498c0642f Repository:docker-sonic-bmp Size:315MB Tag:20250510.14] map[ID:sha256:0253e833a6cb4472e2dfa5abb93a742e3d8a6b54c14c8d8730e1b8a498c0642f Repository:docker-sonic-bmp Size:315MB Tag:latest] map[ID:sha256:4359efb75b869a8a097d02a5cfc7d825c5867f9fbe2aebd3347c73f7f8a60771 Repository:docker-bmp-watchdog Size:314MB Tag:20250510.14] map[ID:sha256:4359efb75b869a8a097d02a5cfc7d825c5867f9fbe2aebd3347c73f7f8a60771 Repository:docker-bmp-watchdog Size:314MB Tag:latest] map[ID:sha256:6c635884f18ca53c80dccf3f1a6a461f41f13eb65d2d2a3f3365d6c00869f2b9 Repository:docker-auditd-watchdog Size:314MB Tag:20250510.14] map[ID:sha256:6c635884f18ca53c80dccf3f1a6a461f41f13eb65d2d2a3f3365d6c00869f2b9 Repository:docker-auditd-watchdog Size:314MB Tag:latest] map[ID:sha256:7834b3b6b5652d63fd9b2e7144b4abfcf276252d38e4ee43d3242b2f1836babf Repository:docker-acms Size:365MB Tag:20250510.14] map[ID:sha256:7834b3b6b5652d63fd9b2e7144b4abfcf276252d38e4ee43d3242b2f1836babf Repository:docker-acms Size:365MB Tag:latest] map[ID:sha256:faf9d6c54d9cad4012e002576f1566106cfc3ddc838a39ce210fb508a1ef5a29 Repository:docker-auditd Size:314MB Tag:20250510.14] map[ID:sha256:faf9d6c54d9cad4012e002576f1566106cfc3ddc838a39ce210fb508a1ef5a29 Repository:docker-auditd Size:314MB Tag:latest] map[ID:sha256:ed210e3e4a5bae1237f1bb44d72a05a2f1e5c6bfe7a7e73da179e2534269c459 Repository:k8s.gcr.io/pause Size:683kB Tag:3.5]]"
}
`
	ResetDataSetsAndMappings(t)

	tests := []struct {
		desc           string
		pathTarget     string
		textPbPath     string
		wantRetCode    codes.Code
		wantRespVal    interface{}
		valTest        bool
		mockOutputFile map[string]string
		testTime       time.Time
		testInit       func()
	}{
		{
			desc:       "query SHOW version without evnironment variable",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "version" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutputWithEmptyPlat),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/VERSION_DOCKER_IMAGEDATA.txt",
				"uptime": "../testdata/VERSION_UPTIME.txt",
			},
			testTime: time.Date(2025, 7, 18, 18, 0, 0, 0, time.UTC),
			testInit: func() {
				MockReadFile(show_client.SonicVersionYamlPath, versionInfo, nil)
				MockEnvironmentVariable(t, "PLATFORM", "")
				AddDataSet(t, ConfigDbNum, deviceMetadataFilename)
				AddDataSet(t, chassisStateDbNum, chassisDataFilename)
			},
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}
		var patches *gomonkey.Patches
		if len(test.mockOutputFile) > 0 {
			patches = MockExecCmds(t, test.mockOutputFile)
		}

		testTime := test.testTime
		timepatch := gomonkey.ApplyFunc(time.Now, func() time.Time {
			return testTime
		})

		fileOpenPatch := gomonkey.ApplyFunc(show_client.ReadConfToMap, func(string) (map[string]interface{}, error) {
			data := map[string]interface{}{
				"onie_platform":  "test_onie_platform",
				"aboot_platform": "test_aboot_platform",
			}
			return data, nil
		})

		asicFilePatch := gomonkey.ApplyFunc(show_client.GetAsicConfFilePath, func() string {
			return "../testdata/version_test_asics_num.conf"
		})

		platformConfigFilePatch := gomonkey.ApplyFunc(show_client.GetPlatformEnvConfFilePath, func() string {
			return "../testdata/VERSION_TEST_PLATFORM.conf"
		})

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})

		if patches != nil {
			patches.Reset()
		}
		if timepatch != nil {
			timepatch.Reset()
		}
		if fileOpenPatch != nil {
			fileOpenPatch.Reset()
		}
		if asicFilePatch != nil {
			asicFilePatch.Reset()
		}
		if platformConfigFilePatch != nil {
			platformConfigFilePatch.Reset()
		}
	}
}

func TestGetShowVersion(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()

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

	// Mock interface error data with some errors present
	versionInfo := `
build_version: test_branch.1-a8fbac59d
debian_version: 11.4
kernel_version: 5.10.0-18-2-amd64
asic_type: mellanox
asic_subtype: mellanox
commit_id: a8fbac59d
branch: test_branch
release: master
libswsscommon: 1.0.0
sonic_utilities: 1.2
`
	deviceMetadataFilename := "../testdata/VERSION_METADATA.txt"
	chassisDataFilename := "../testdata/VERSION_CHASSIS.txt"

	// Mock interface error data with no errors (all zeros)
	expectedOutput := `
{
  "sonic_software_version": "SONiC.test_branch.1-a8fbac59d",
  "sonic_os_version": "\u003cnil\u003e",
  "distribution": "Debian 11.4",
  "kernel": "5.10.0-18-2-amd64",
  "build_commit": "a8fbac59d",
  "build_date": "\u003cnil\u003e",
  "built_by": "\u003cnil\u003e",
  "platform": "test_onie_platform",
  "hwsku": "",
  "asic": "mellanox",
  "asic_count": "N/A",
  "serial_number": "",
  "model_number": "",
  "hardware_revision": "",
  "uptime": "07:42:51 up 16 days, 14:51,  2 users,  load average: 0.00, 0.00, 0.00",
  "date": "Fri 18 Jul 2025 18:00:00",
  "docker_info": "[map[ID:sha256:372601148371578e886d545864469deca0ad8db5618e10ae6124429637f86a02 Repository:docker-mux Size:366MB Tag:20250510.14] map[ID:sha256:372601148371578e886d545864469deca0ad8db5618e10ae6124429637f86a02 Repository:docker-mux Size:366MB Tag:latest] map[ID:sha256:e035d4584c6c6bd7a48430abbba01e91737022f9ec42749af3de1f38751204af Repository:docker-dhcp-server Size:337MB Tag:20250510.14] map[ID:sha256:e035d4584c6c6bd7a48430abbba01e91737022f9ec42749af3de1f38751204af Repository:docker-dhcp-server Size:337MB Tag:latest] map[ID:sha256:4f8bfd4cbdd732ae3891bc34b49adf920804ec62eacc234917eefb0f6dfbe2f3 Repository:docker-macsec Size:347MB Tag:20250510.14] map[ID:sha256:4f8bfd4cbdd732ae3891bc34b49adf920804ec62eacc234917eefb0f6dfbe2f3 Repository:docker-macsec Size:347MB Tag:latest] map[ID:sha256:f33ca825cb610d1e0ab12f5d8c9a3b39b11c8e9945386c79fd36816d7441bfda Repository:docker-eventd Size:314MB Tag:20250510.14] map[ID:sha256:f33ca825cb610d1e0ab12f5d8c9a3b39b11c8e9945386c79fd36816d7441bfda Repository:docker-eventd Size:314MB Tag:latest] map[ID:sha256:8884f25155d4ba354712b6f21510cf75e0d0e7401cb2a61609eb940a4629b178 Repository:docker-sonic-telemetry Size:408MB Tag:20250510.14] map[ID:sha256:8884f25155d4ba354712b6f21510cf75e0d0e7401cb2a61609eb940a4629b178 Repository:docker-sonic-telemetry Size:408MB Tag:latest] map[ID:sha256:3922a17378ed1caf1b5c85c07fd73e5f8e7e85ad4a3c690e1c02cc3e4dae4c7b Repository:docker-gbsyncd-broncos Size:367MB Tag:20250510.14] map[ID:sha256:3922a17378ed1caf1b5c85c07fd73e5f8e7e85ad4a3c690e1c02cc3e4dae4c7b Repository:docker-gbsyncd-broncos Size:367MB Tag:latest] map[ID:sha256:5bf1e2f6561ebd6b5e84d91caab6a18776c1fa1be55b0d915d244cc5f528aa88 Repository:docker-gbsyncd-credo Size:341MB Tag:20250510.14] map[ID:sha256:5bf1e2f6561ebd6b5e84d91caab6a18776c1fa1be55b0d915d244cc5f528aa88 Repository:docker-gbsyncd-credo Size:341MB Tag:latest] map[ID:sha256:4c777e17558bded5ab09152c29d59f95073bca928a90c1cfacb9736d900daf72 Repository:docker-sonic-gnmi Size:408MB Tag:20250510.14] map[ID:sha256:4c777e17558bded5ab09152c29d59f95073bca928a90c1cfacb9736d900daf72 Repository:docker-sonic-gnmi Size:408MB Tag:latest] map[ID:sha256:44d4ca23b0f52ae9a056bea6f87e05a20de9e0677386e9af381252ae282cfc85 Repository:docker-orchagent Size:356MB Tag:20250510.14] map[ID:sha256:44d4ca23b0f52ae9a056bea6f87e05a20de9e0677386e9af381252ae282cfc85 Repository:docker-orchagent Size:356MB Tag:latest] map[ID:sha256:1d1ce9b15c8ca8bf5b1c8ccee1b9076ccdc5c52e5cd60b0c525d7edb75d94627 Repository:docker-dhcp-relay Size:325MB Tag:20250510.14] map[ID:sha256:1d1ce9b15c8ca8bf5b1c8ccee1b9076ccdc5c52e5cd60b0c525d7edb75d94627 Repository:docker-dhcp-relay Size:325MB Tag:latest] map[ID:sha256:6433be677e7a8aef06842a4f555ed0ddac0c30487e041abf08c206cb9db6dfff Repository:docker-fpm-frr Size:386MB Tag:20250510.14] map[ID:sha256:6433be677e7a8aef06842a4f555ed0ddac0c30487e041abf08c206cb9db6dfff Repository:docker-fpm-frr Size:386MB Tag:latest] map[ID:sha256:9c8ff651cc4c6c30b2f091fda457e464262cd2c39be9cb1576176d330f381e25 Repository:docker-syncd-brcm Size:799MB Tag:20250510.14] map[ID:sha256:9c8ff651cc4c6c30b2f091fda457e464262cd2c39be9cb1576176d330f381e25 Repository:docker-syncd-brcm Size:799MB Tag:latest] map[ID:sha256:49eb5cc18cff4f5c1a3f8594ed12981838fc60a62efc9ebcfcb31afc94d7b9a7 Repository:docker-dash-ha Size:357MB Tag:20250510.14] map[ID:sha256:49eb5cc18cff4f5c1a3f8594ed12981838fc60a62efc9ebcfcb31afc94d7b9a7 Repository:docker-dash-ha Size:357MB Tag:latest] map[ID:sha256:9b738a9e879a527849c16417862644691d84e5c8a128c51a5c9753b5e4e98252 Repository:docker-snmp Size:358MB Tag:20250510.14] map[ID:sha256:9b738a9e879a527849c16417862644691d84e5c8a128c51a5c9753b5e4e98252 Repository:docker-snmp Size:358MB Tag:latest] map[ID:sha256:071a45740d3b657b90665c8028e39687c89f41310a6b345dd395f27bcf7bb618 Repository:docker-platform-monitor Size:446MB Tag:20250510.14] map[ID:sha256:071a45740d3b657b90665c8028e39687c89f41310a6b345dd395f27bcf7bb618 Repository:docker-platform-monitor Size:446MB Tag:latest] map[ID:sha256:af98df3b20e54deb0039e923889f653bb320f41115d39dde9940ed5b2c2a8f7d Repository:docker-sysmgr Size:326MB Tag:20250510.14] map[ID:sha256:af98df3b20e54deb0039e923889f653bb320f41115d39dde9940ed5b2c2a8f7d Repository:docker-sysmgr Size:326MB Tag:latest] map[ID:sha256:9ee9b197541dc46f68c42512f4cb0d2f84114b95c8d57e09405bc8663089474f Repository:docker-teamd Size:344MB Tag:20250510.14] map[ID:sha256:9ee9b197541dc46f68c42512f4cb0d2f84114b95c8d57e09405bc8663089474f Repository:docker-teamd Size:344MB Tag:latest] map[ID:sha256:32b732da2119047c2c38c783e971df3ad60c8076730a86d34d83fe85e9b8ae65 Repository:docker-router-advertiser Size:314MB Tag:20250510.14] map[ID:sha256:32b732da2119047c2c38c783e971df3ad60c8076730a86d34d83fe85e9b8ae65 Repository:docker-router-advertiser Size:314MB Tag:latest] map[ID:sha256:44c6b595de678dc283e31c43abfbc73e45985f243b219d80a3e9ca2a06ac579c Repository:docker-sonic-restapi Size:333MB Tag:20250510.14] map[ID:sha256:44c6b595de678dc283e31c43abfbc73e45985f243b219d80a3e9ca2a06ac579c Repository:docker-sonic-restapi Size:333MB Tag:latest] map[ID:sha256:20f8142edc812a92fa9c563774b96c99fd8b95e23dc682be6984f83122382735 Repository:docker-gnmi-watchdog Size:314MB Tag:20250510.14] map[ID:sha256:20f8142edc812a92fa9c563774b96c99fd8b95e23dc682be6984f83122382735 Repository:docker-gnmi-watchdog Size:314MB Tag:latest] map[ID:sha256:333901845f00aaf8a4cd335584dc5fdea589aa7a9722ecb0b28e615215ce2d80 Repository:docker-lldp Size:360MB Tag:20250510.14] map[ID:sha256:333901845f00aaf8a4cd335584dc5fdea589aa7a9722ecb0b28e615215ce2d80 Repository:docker-lldp Size:360MB Tag:latest] map[ID:sha256:ea4d44010ec1ca9a5129842aebe5a36d177c51a585c4c54e83bd6c4325492af9 Repository:docker-database Size:321MB Tag:20250510.14] map[ID:sha256:ea4d44010ec1ca9a5129842aebe5a36d177c51a585c4c54e83bd6c4325492af9 Repository:docker-database Size:321MB Tag:latest] map[ID:sha256:4411bd9004debb98ad3329ccd50bb7fceee312b80e8fd5ba23db9f1017732f0c Repository:docker-dummyk8s Size:314MB Tag:20250510.14] map[ID:sha256:4411bd9004debb98ad3329ccd50bb7fceee312b80e8fd5ba23db9f1017732f0c Repository:docker-dummyk8s Size:314MB Tag:latest] map[ID:sha256:0253e833a6cb4472e2dfa5abb93a742e3d8a6b54c14c8d8730e1b8a498c0642f Repository:docker-sonic-bmp Size:315MB Tag:20250510.14] map[ID:sha256:0253e833a6cb4472e2dfa5abb93a742e3d8a6b54c14c8d8730e1b8a498c0642f Repository:docker-sonic-bmp Size:315MB Tag:latest] map[ID:sha256:4359efb75b869a8a097d02a5cfc7d825c5867f9fbe2aebd3347c73f7f8a60771 Repository:docker-bmp-watchdog Size:314MB Tag:20250510.14] map[ID:sha256:4359efb75b869a8a097d02a5cfc7d825c5867f9fbe2aebd3347c73f7f8a60771 Repository:docker-bmp-watchdog Size:314MB Tag:latest] map[ID:sha256:6c635884f18ca53c80dccf3f1a6a461f41f13eb65d2d2a3f3365d6c00869f2b9 Repository:docker-auditd-watchdog Size:314MB Tag:20250510.14] map[ID:sha256:6c635884f18ca53c80dccf3f1a6a461f41f13eb65d2d2a3f3365d6c00869f2b9 Repository:docker-auditd-watchdog Size:314MB Tag:latest] map[ID:sha256:7834b3b6b5652d63fd9b2e7144b4abfcf276252d38e4ee43d3242b2f1836babf Repository:docker-acms Size:365MB Tag:20250510.14] map[ID:sha256:7834b3b6b5652d63fd9b2e7144b4abfcf276252d38e4ee43d3242b2f1836babf Repository:docker-acms Size:365MB Tag:latest] map[ID:sha256:faf9d6c54d9cad4012e002576f1566106cfc3ddc838a39ce210fb508a1ef5a29 Repository:docker-auditd Size:314MB Tag:20250510.14] map[ID:sha256:faf9d6c54d9cad4012e002576f1566106cfc3ddc838a39ce210fb508a1ef5a29 Repository:docker-auditd Size:314MB Tag:latest] map[ID:sha256:ed210e3e4a5bae1237f1bb44d72a05a2f1e5c6bfe7a7e73da179e2534269c459 Repository:k8s.gcr.io/pause Size:683kB Tag:3.5]]"
}
`
	ResetDataSetsAndMappings(t)

	tests := []struct {
		desc           string
		pathTarget     string
		textPbPath     string
		wantRetCode    codes.Code
		wantRespVal    interface{}
		valTest        bool
		mockOutputFile map[string]string
		testTime       time.Time
		testInit       func()
	}{
		{
			desc:       "query SHOW version with evnironment variable",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "version" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutput),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/VERSION_DOCKER_IMAGEDATA.txt",
				"uptime": "../testdata/VERSION_UPTIME.txt",
			},
			testTime: time.Date(2025, 7, 18, 18, 0, 0, 0, time.UTC),
			testInit: func() {
				MockReadFile(show_client.SonicVersionYamlPath, versionInfo, nil)
				MockEnvironmentVariable(t, "PLATFORM", "dummy_platform")
				AddDataSet(t, ConfigDbNum, deviceMetadataFilename)
				AddDataSet(t, chassisStateDbNum, chassisDataFilename)
			},
		},
		{
			desc:       "query SHOW version with file yaml error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "version" >
			`,
			wantRetCode: codes.NotFound,
			wantRespVal: nil,
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/VERSION_DOCKER_IMAGEDATA.txt",
				"uptime": "../testdata/VERSION_UPTIME.txt",
			},
			testTime: time.Date(2025, 7, 18, 18, 0, 0, 0, time.UTC),
			testInit: func() {
				MockReadFile(show_client.SonicVersionYamlPath, versionInfo, errors.New("test error."))
				AddDataSet(t, ConfigDbNum, deviceMetadataFilename)
				AddDataSet(t, chassisStateDbNum, chassisDataFilename)
			},
		},
		{
			desc:       "query SHOW version with env variable and yaml error",
			pathTarget: "SHOW",
			textPbPath: `
				elem: <name: "version" >
			`,
			wantRetCode: codes.OK,
			wantRespVal: []byte(expectedOutput),
			valTest:     true,
			mockOutputFile: map[string]string{
				"docker": "../testdata/VERSION_DOCKER_IMAGEDATA.txt",
				"uptime": "../testdata/VERSION_UPTIME.txt",
			},
			testTime: time.Date(2025, 7, 18, 18, 0, 0, 0, time.UTC),
			testInit: func() {
				MockReadFile(show_client.SonicVersionYamlPath, versionInfo, nil)
				MockEnvironmentVariable(t, "PLATFORM", "dummy_platform")
				AddDataSet(t, ConfigDbNum, deviceMetadataFilename)
				AddDataSet(t, chassisStateDbNum, chassisDataFilename)
			},
		},
	}

	for _, test := range tests {
		if test.testInit != nil {
			test.testInit()
		}
		var patches *gomonkey.Patches
		if len(test.mockOutputFile) > 0 {
			patches = MockExecCmds(t, test.mockOutputFile)
		}

		testTime := test.testTime
		timepatch := gomonkey.ApplyFunc(time.Now, func() time.Time {
			return testTime
		})

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, test.pathTarget, test.textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
		if patches != nil {
			patches.Reset()
		}
		if timepatch != nil {
			timepatch.Reset()
		}
	}
}
