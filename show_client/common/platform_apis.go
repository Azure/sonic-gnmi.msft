package common

// SsdHealthPyScript is the Python script template that loads the platform-specific
// or generic SsdUtil and retrieves SSD health information as JSON.
// It expects two %s format parameters: platform path, then device path.
var SsdHealthPyScript = `
import sys, os, json
try:
    platform_plugins_path = os.path.join('%s', 'plugins')
    sys.path.append(os.path.abspath(platform_plugins_path))
    from ssd_util import SsdUtil
except ImportError as e:
    try:
        from sonic_platform_base.sonic_storage.ssd import SsdUtil
    except ImportError as e:
        raise e
s = SsdUtil('%s')
print(json.dumps({'model': str(s.get_model()), 'firmware': str(s.get_firmware()), 'serial': str(s.get_serial()), 'health': str(s.get_health()), 'temperature': str(s.get_temperature()), 'vendor_output': str(s.get_vendor_output())}))
`

// PcieInfoPyScript is the Python script template that loads the platform-specific
// or generic Pcie/PcieUtil and retrieves PCIe information as JSON.
// It expects two %s format parameters: platform path, then the API call (e.g., pcie.get_pcie_device() or pcie.get_pcie_check()).
var PcieInfoPyScript = `
import json
platform_path = %s
try:
    from sonic_platform.pcie import Pcie
    pcie = Pcie(platform_path)
except ImportError:
    from sonic_platform_base.sonic_pcie.pcie_common import PcieUtil
    pcie = PcieUtil(platform_path)
print(json.dumps(%s))
`
