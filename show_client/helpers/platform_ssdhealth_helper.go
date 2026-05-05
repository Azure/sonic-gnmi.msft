package helpers

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	"golang.org/x/sys/unix"
)

const (
	defaultSsdDevice = "/dev/sda"
	hostMountPoint   = "/host"
	// lsblk -J -o NAME,MAJ:MIN,TYPE,TRAN outputs JSON with block device info
	lsblkJSONCmd = "lsblk -J -o NAME,MAJ:MIN,TYPE,TRAN"
)

// lsblkOutput represents the top-level JSON output of lsblk -J.
type lsblkOutput struct {
	BlockDevices []lsblkDevice `json:"blockdevices"`
}

// lsblkDevice represents a single block device entry from lsblk -J.
type lsblkDevice struct {
	Name   string  `json:"name"`
	MajMin string  `json:"maj:min"`
	Type   string  `json:"type"`
	Tran   *string `json:"tran"`
}

// SsdInfo holds the SSD information retrieved from the Python SsdUtil API.
type SsdInfo struct {
	DiskType     string `json:"disk_type"`
	Model        string `json:"model"`
	Firmware     string `json:"firmware"`
	Serial       string `json:"serial"`
	Health       string `json:"health"`
	Temperature  string `json:"temperature"`
	VendorOutput string `json:"vendor_output"`
}

// GetDefaultDisk discovers the host disk device and transport type.
func GetDefaultDisk() (defaultDevice string, diskType string) {
	defaultDevice = defaultSsdDevice
	diskType = ""

	partitions, err := disk.Partitions(false)
	if err != nil || partitions == nil {
		log.V(4).Infof("Could not get disk partitions: %v", err)
		return defaultDevice, diskType
	}

	var hostPartition disk.PartitionStat
	found := false
	for _, parts := range partitions {
		if parts.Mountpoint == hostMountPoint {
			hostPartition = parts
			found = true
			break
		}
	}
	if !found {
		log.V(4).Infof("No %s mount found in disk partitions", hostMountPoint)
		return defaultDevice, diskType
	}

	var statResult unix.Stat_t
	if err := unix.Stat(hostPartition.Device, &statResult); err != nil {
		log.V(4).Infof("Could not stat %s: %v", hostPartition.Device, err)
		return defaultDevice, diskType
	}
	diskMajor := unix.Major(uint64(statResult.Rdev))

	majMinFilter := fmt.Sprintf("%d:0", diskMajor)

	// (replicated via lsblk -J since Go has no blkinfo equivalent)
	lsblkRaw, err := common.GetDataFromHostCommand(lsblkJSONCmd)
	if err != nil {
		log.V(4).Infof("Could not run lsblk: %v", err)
		return defaultDevice, diskType
	}

	var lsblkResult lsblkOutput
	if err := json.Unmarshal([]byte(lsblkRaw), &lsblkResult); err != nil {
		log.V(4).Infof("Could not parse lsblk JSON: %v", err)
		return defaultDevice, diskType
	}

	var matchedDisk lsblkDevice
	diskFound := false
	for _, dev := range lsblkResult.BlockDevices {
		if dev.Type == "disk" && dev.MajMin == majMinFilter {
			matchedDisk = dev
			diskFound = true
			break
		}
	}
	if !diskFound {
		log.V(4).Infof("No parent disk found with maj:min %s", majMinFilter)
		return defaultDevice, diskType
	}

	defaultDevice = filepath.Join("/dev", matchedDisk.Name)
	if matchedDisk.Tran != nil {
		diskType = *matchedDisk.Tran
	}

	if len(diskType) == 0 && strings.Contains(hostPartition.Device, "mmcblk") {
		diskType = "eMMC"
	}

	return defaultDevice, diskType
}

// ImportSsdApi loads the SSD utility (platform-specific or generic) via nsenter
// and retrieves SSD health information for the given device.
func ImportSsdApi(device string) (*SsdInfo, error) {
	platformPath, _ := common.GetPathsToPlatformAndHwskuDirsOnHost()
	pyScript := fmt.Sprintf(common.SsdHealthPyScript, platformPath, device)
	shlexSafeScript := strings.ReplaceAll(pyScript, "'", `'\''`)
	pyCmd := fmt.Sprintf("python3 -c '%s'", shlexSafeScript)

	output, err := common.GetDataFromHostCommand(pyCmd)
	if err != nil {
		log.Errorf("ImportSsdApi: command failed for %s: %v", device, err)
		return nil, fmt.Errorf("failed to get SSD info for %s: %w", device, err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		log.Errorf("ImportSsdApi: empty response for device %s", device)
		return nil, fmt.Errorf("empty response from SSD utility for %s", device)
	}

	log.V(4).Infof("ImportSsdApi: raw output for %s: %s", device, output)

	var info SsdInfo
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		log.Errorf("ImportSsdApi: JSON parse failed for %s: %v", device, err)
		return nil, fmt.Errorf("failed to parse SSD info JSON: %w", err)
	}

	log.V(4).Infof("ImportSsdApi: successfully parsed SSD info for %s", device)
	return &info, nil
}

func GetSsdHealthData(device string) (*SsdInfo, error) {
	/* GetSsdHealthData resolves the SSD device and retrieves health data.
	 1. Use explicit device if provided (from args)
	 2. Try platform.json: chassis.disk.device
	 3. Call GetDefaultDisk() to discover /host partition's parent disk
	 4. Call ImportSsdApi to get SSD info
	Returns ssdInfo (with DiskType set) or error if SSD not detected. */
	if device == "" {
		platformData, err := common.GetPlatformJsonData()
		if err == nil && platformData != nil {
			device = common.GetNestedString(platformData, "chassis", "disk", "device")
			if device != "" {
				log.V(4).Infof("GetSsdHealthData: device from platform.json: %s", device)
			}
		} else if err != nil {
			log.V(4).Infof("GetSsdHealthData: platform.json not available: %v", err)
		}
	}

	defaultDevice, diskType := GetDefaultDisk()
	log.V(4).Infof("GetSsdHealthData: default disk=%s, disk type=%s", defaultDevice, diskType)

	if device == "" {
		device = defaultDevice
	}

	ssdInfo, err := ImportSsdApi(device)
	if err != nil {
		log.Errorf("GetSsdHealthData: failed to get SSD info for %s: %v", device, err)
		return nil, err
	}

	ssdInfo.DiskType = diskType
	return ssdInfo, nil
}

// IsNumber checks if a string can be parsed as a float.
func IsNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// FormatHealth formats the health value with % suffix if numeric.
func FormatHealth(health string) string {
	if IsNumber(health) {
		return health + "%"
	}
	return health
}

// FormatTemperature formats the temperature value with C suffix if numeric.
func FormatTemperature(temperature string) string {
	if IsNumber(temperature) {
		return temperature + "C"
	}
	return temperature
}
