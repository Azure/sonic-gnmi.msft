package common

// SysEepromPyScript is the Python script that invokes the sonic_platform API
// to retrieve system EEPROM info.
var SysEepromPyScript = `
import sys
try:
    import sonic_platform
    eeprom = sonic_platform.platform.Platform().get_chassis().get_eeprom()
except Exception:
    eeprom = None
if not eeprom:
    sys.exit(1)
sys_eeprom_data = eeprom.read_eeprom()
eeprom.decode_eeprom(sys_eeprom_data)
`
