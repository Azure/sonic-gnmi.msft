package helpers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

// ComponentInfo holds component data from Platform API
type ComponentInfo struct {
	Name            string
	FirmwareVersion string
	Description     string
}

// ModuleComponentInfo holds module component data
type ModuleComponentInfo struct {
	ModuleName      string
	Name            string
	FirmwareVersion string
	Description     string
}

// FirmwareData holds complete firmware information for a component
type FirmwareData struct {
	Chassis     string
	Module      string
	Component   string
	Version     string
	Description string
}

// GetAllFirmwareData retrieves complete firmware information using Platform API
func GetAllFirmwareData() ([]FirmwareData, error) {
	firmwareList := make([]FirmwareData, 0)

	// Get chassis name
	chassisName, err := GetChassisName()
	if err != nil {
		chassisName = "N/A"
	}

	// Check if modular chassis to determine module name logic
	isModularChassis := false
	moduleComponents, moduleErr := GetModuleComponents()
	if moduleErr == nil && len(moduleComponents) > 0 {
		isModularChassis = true
	}

	appendChassisName := true
	appendModuleNA := !isModularChassis // Show "N/A" for non-modular chassis

	// Get chassis components
	chassisComponents, err := GetChassisComponents()
	if err == nil {
		for _, component := range chassisComponents {
			moduleField := ""
			if appendModuleNA {
				moduleField = "N/A"
				appendModuleNA = false
			}

			firmware := FirmwareData{
				Chassis: func() string {
					if appendChassisName {
						appendChassisName = false
						return chassisName
					}
					return ""
				}(),
				Module:      moduleField,
				Component:   component.Name,
				Version:     component.FirmwareVersion,
				Description: component.Description,
			}
			firmwareList = append(firmwareList, firmware)
		}
	}

	// Get module components for modular chassis
	if isModularChassis {
		currentModuleName := ""
		appendModuleName := false

		for _, moduleComp := range moduleComponents {
			// New module - show module name for first component
			if moduleComp.ModuleName != currentModuleName {
				currentModuleName = moduleComp.ModuleName
				appendModuleName = true
			}

			moduleNameField := ""
			if appendModuleName {
				moduleNameField = moduleComp.ModuleName
				appendModuleName = false
			}

			firmware := FirmwareData{
				Chassis: func() string {
					if appendChassisName {
						appendChassisName = false
						return chassisName
					}
					return ""
				}(),
				Module:      moduleNameField,
				Component:   moduleComp.Name,
				Version:     moduleComp.FirmwareVersion,
				Description: moduleComp.Description,
			}
			firmwareList = append(firmwareList, firmware)
		}
	}

	return firmwareList, nil
}

// GetChassisName calls Platform API to get chassis name
func GetChassisName() (string, error) {
	// Query CHASSIS_INFO database table for chassis model
	queries := [][]string{
		{"STATE_DB", "CHASSIS_INFO"},
	}

	chassisData, err := common.GetMapFromQueries(queries)
	if err != nil {
		return "", err
	}

	// Extract chassis name (model) from database
	if chassisInfo, ok := chassisData["chassis 1"].(map[string]interface{}); ok {
		model := common.GetValueOrDefault(chassisInfo, "model", "N/A")
		return model, nil
	}

	return "N/A", nil
}

// GetChassisComponents calls Platform API to get chassis components
func GetChassisComponents() ([]ComponentInfo, error) {
	pythonScript := `
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
	escaped := strings.ReplaceAll(pythonScript, "'", `'\''`)
	command := fmt.Sprintf("python3 -c '%s'", escaped)

	output, err := common.GetDataFromHostCommand(command)
	if err != nil {
		return nil, err
	}

	var rawComponents []map[string]string
	if err := json.Unmarshal([]byte(output), &rawComponents); err != nil {
		return nil, err
	}

	// Parse into Go structs
	components := make([]ComponentInfo, 0, len(rawComponents))
	for _, raw := range rawComponents {
		components = append(components, ComponentInfo{
			Name:            raw["name"],
			FirmwareVersion: raw["firmware_version"],
			Description:     raw["description"],
		})
	}

	return components, nil
}

// GetModuleComponents calls Platform API to get module components
func GetModuleComponents() ([]ModuleComponentInfo, error) {
	pythonScript := `
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
	escaped := strings.ReplaceAll(pythonScript, "'", `'\''`)
	command := fmt.Sprintf("python3 -c '%s'", escaped)

	output, err := common.GetDataFromHostCommand(command)
	if err != nil {
		return nil, err
	}

	var rawComponents []map[string]string
	if err := json.Unmarshal([]byte(output), &rawComponents); err != nil {
		return nil, err
	}

	// Parse into Go structs
	components := make([]ModuleComponentInfo, 0, len(rawComponents))
	for _, raw := range rawComponents {
		components = append(components, ModuleComponentInfo{
			ModuleName:      raw["module_name"],
			Name:            raw["name"],
			FirmwareVersion: raw["firmware_version"],
			Description:     raw["description"],
		})
	}

	return components, nil
}
