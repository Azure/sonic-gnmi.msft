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

// ChassisComponentsPyScript retrieves all chassis components via Platform API
var ChassisComponentsPyScript = `
import json
try:
    from sonic_platform.platform import Platform
    chassis = Platform().get_chassis()
    components = []
    
    if hasattr(chassis, 'get_all_components'):
        for component in chassis.get_all_components():
            try:
                components.append({
                    'name': component.get_name() if hasattr(component, 'get_name') else 'N/A',
                    'firmware_version': component.get_firmware_version() if hasattr(component, 'get_firmware_version') else 'N/A',
                    'description': component.get_description() if hasattr(component, 'get_description') else 'N/A'
                })
            except Exception:
                continue
    
    print(json.dumps(components))
except Exception:
    print('[]')
`

// ModuleComponentsPyScript retrieves all module components via Platform API
var ModuleComponentsPyScript = `
import json
try:
    from sonic_platform.platform import Platform
    chassis = Platform().get_chassis()
    components = []
    
    if hasattr(chassis, 'get_all_modules'):
        for module in chassis.get_all_modules():
            try:
                module_name = module.get_name() if hasattr(module, 'get_name') else 'N/A'
                
                if hasattr(module, 'get_all_components'):
                    for component in module.get_all_components():
                        try:
                            components.append({
                                'module_name': module_name,
                                'name': component.get_name() if hasattr(component, 'get_name') else 'N/A',
                                'firmware_version': component.get_firmware_version() if hasattr(component, 'get_firmware_version') else 'N/A',
                                'description': component.get_description() if hasattr(component, 'get_description') else 'N/A'
                            })
                        except Exception:
                            continue
            except Exception:
                continue
    
    print(json.dumps(components))
except Exception:
    print('[]')
`

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
