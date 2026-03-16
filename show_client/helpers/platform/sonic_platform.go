/*
	sonic_platform.go

	Factory for creating the correct platform-specific Chassis object based on
	the current platform name. This abstracts platform detection so callers
	simply call NewChassis() and get the right implementation.

	Ported from:
	  sonic-buildimage/src/sonic-platform-common/sonic_platform/chassis.py
	  (the import logic that selects the correct vendor module)
*/
package platform

import (
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	"github.com/sonic-net/sonic-gnmi/show_client/helpers/platform/mellanox"
)

func NewChassis() ChassisBase {
	/* NewChassis detects the current platform and returns the appropriate
		ChassisBase implementation. Returns nil if the platform is not supported.
		Currently supported platforms:
		  - Mellanox / NVIDIA (platform name contains "mlnx" or "nvidia")*/
	platformName := strings.ToLower(common.GetPlatform())

	if strings.Contains(platformName, "mlnx") || strings.Contains(platformName, "nvidia") {
		return &mellanox.Chassis{}
	}

	log.Warningf("Platform '%s' is not supported for chassis LED control, skipping", platformName)
	return nil
}
