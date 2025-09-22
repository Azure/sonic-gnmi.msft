package show_client

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

func getAllPortsFromConfigDB() ([]string, error) {
	queries := [][]string{
		{"CONFIG_DB", "PORT"},
	}
	data, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get data from CONFIG_DB queries %v, got err: %v", queries, err)
		return nil, err
	}
	log.V(6).Infof("Data from CONFIG_DB: %v", data)

	ports := make([]string, 0, len(data))
	for iface := range data {
		ports = append(ports, iface)
	}
	return ports, nil
}

func getTransceiverErrorStatus(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf := args.At(0)

	var queries [][]string
	if intf == "" {
		queries = [][]string{
			{"STATE_DB", "TRANSCEIVER_STATUS_SW"},
		}
	} else {
		queries = [][]string{
			{"STATE_DB", "TRANSCEIVER_STATUS_SW", intf},
		}
	}

	data, err := GetDataFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get data from queries %v, got err: %v", queries, err)
		return nil, err
	}
	return data, nil
}

func getInterfaceTransceiverPresence(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf := args.At(0)

	// Get STATE_DB transceiver info
	queries := [][]string{
		{"STATE_DB", "TRANSCEIVER_INFO"},
	}
	data, err := common.GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Unable to get transceiver data from STATE_DB queries %v, got err: %v", queries, err)
		return nil, err
	}
	log.V(6).Infof("TRANSCEIVER_INFO Data from STATE_DB: %v", data)

	status := make(map[string]string)

	if intf != "" {
		// If specific interface provided, skip ConfigDB check
		if _, exist := data[intf]; exist {
			status[intf] = "Present"
		} else {
			status[intf] = "Not Present"
		}
	} else {
		// No specific interface provided, get all from ConfigDB
		ports, err := getAllPortsFromConfigDB()
		if err != nil {
			log.Errorf("Unable to get all ports from CONFIG_DB, %v", err)
			return nil, err
		}

		for _, port := range ports {
			if _, exist := data[port]; exist {
				status[port] = "Present"
			} else {
				status[port] = "Not Present"
			}
		}
	}

	log.V(6).Infof("Transceiver presence status: %v", status)
	return json.Marshal(status)
}

type portLpmode struct {
	Port   string `json:"Port"`
	Lpmode string `json:"Low-power Mode"`
}

func getInterfaceTransceiverLpMode(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf := args.At(0)
	cmdParts := []string{"sudo", "sfputil", "show", "lpmode"}
	if intf != "" {
		cmdParts = append(cmdParts, "-p", intf)
	}
	cmdStr := strings.Join(cmdParts, " ")

	output, err := common.GetDataFromHostCommand(cmdStr)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")
	entries := make([]portLpmode, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Port") || strings.HasPrefix(line, "---") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		port := fields[0]
		mode := fields[1]
		ml := strings.ToLower(mode)
		if ml == "on" || ml == "off" {
			mode = strings.Title(ml)
		}
		entries = append(entries, portLpmode{Port: port, Lpmode: mode})
	}

	return json.Marshal(entries)
}

// Converts VDM fields to legacy fields
func convertVdmFieldsToLegacyFields(intf string, dictToBeUpdated map[string]string, vdmToLegacyFieldMap map[string]string, vdmFieldType string) {
	if dictToBeUpdated == nil {
		return
	}
	if vdmFieldType != "FLAG" && vdmFieldType != "THRESHOLD" {
		return
	}

	// Fetch VDM dicts
	queries := [][]string{
		{"STATE_DB", fmt.Sprintf("TRANSCEIVER_VDM_HALARM_%s", vdmFieldType), intf},
	}
	vdmHAlarm, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get VDM_HALARM dict from STATE_DB: %v", err)
		return nil, err
	}
	queries = [][]string{
		{"STATE_DB", fmt.Sprintf("TRANSCEIVER_VDM_LALARM_%s", vdmFieldType), intf},
	}
	vdmLAlarm, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get VDM_HALARM dict from STATE_DB: %v", err)
		return nil, err
	}
	queries = [][]string{
		{"STATE_DB", fmt.Sprintf("TRANSCEIVER_VDM_HWARN_%s", vdmFieldType), intf},
	}
	vdmHWarn, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get VDM_HALARM dict from STATE_DB: %v", err)
		return nil, err
	}
	queries = [][]string{
		{"STATE_DB", fmt.Sprintf("TRANSCEIVER_VDM_LWARN_%s", vdmFieldType), intf},
	}
	vdmLWarn, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get VDM_HALARM dict from STATE_DB: %v", err)
		return nil, err
	}

	vdmThresholdTypes := map[string]map[string]string{
		"highalarm":   vdmHAlarm,
		"lowalarm":    vdmLAlarm,
		"highwarning": vdmHWarn,
		"lowwarning":  vdmLWarn,
	}

	for vdmField, legacyPrefix := range vdmToLegacyFieldMap {
		for thresholdType, vdmDict := range vdmThresholdTypes {
			if val, ok := vdmDict[vdmField]; ok {
				var legacyFieldName string
				if vdmFieldType == "FLAG" {
					legacyFieldName = fmt.Sprintf("%s%s_flag", legacyPrefix, thresholdType)
				} else {
					legacyFieldName = fmt.Sprintf("%s%s", legacyPrefix, thresholdType)
				}
				dictToBeUpdated[legacyFieldName] = val
			}
		}
	}
}

func BeautifyPmField(prefix string, field float64, unit string) string {
	if unit == "N/A" {
		// ignore unit
		unit = ""
	}

	if prefix == "prefec_ber" {
		if field != 0 {
			return fmt.Sprintf("%.2f%v", field, unit)
		} else {
			return fmt.Sprintf("0.0%v", unit)
		}
	} else {
		return fmt.Sprintf("%f%v", field, unit)
	}
}

const ZR_PM_NOT_APPLICABLE_STR = "Transceiver performance monitoring not applicable"

var ZR_PM_INFO_MAP = map[string]struct {
	Unit   string
	Prefix string
}{
	"Tx Power":        {"dBm", "tx_power"},
	"Rx Total Power":  {"dBm", "rx_tot_power"},
	"Rx Signal Power": {"dBm", "rx_sig_power"},
	"CD-short link":   {"ps/nm", "cd"},
	"PDL":             {"dB", "pdl"},
	"OSNR":            {"dB", "osnr"},
	"eSNR":            {"dB", "esnr"},
	"CFO":             {"MHz", "cfo"},
	"DGD":             {"ps", "dgd"},
	"SOPMD":           {"ps^2", "sopmd"},
	"SOP ROC":         {"krad/s", "soproc"},
	"Pre-FEC BER":     {"N/A", "prefec_ber"},
	"Post-FEC BER":    {"N/A", "uncorr_frames"},
	"EVM":             {"%", "evm"},
}
var ZR_PM_VALUE_KEY_SUFFIXS = []string{"min", "avg", "max"}
var ZR_PM_THRESHOLD_KEY_SUFFIXS = []string{"highalarm", "highwarning", "lowalarm", "lowwarning"}
var CCMIS_VDM_THRESHOLD_TO_LEGACY_DOM_THRESHOLD_MAP = map[string]string{
	"rxtotpower1":                     "rxtotpower",
	"rxsigpower1":                     "rxsigpower",
	"cdshort1":                        "cdshort",
	"pdl1":                            "pdl",
	"osnr1":                           "osnr",
	"esnr1":                           "esnr",
	"cfo1":                            "cfo",
	"dgd1":                            "dgd",
	"sopmd1":                          "sopmd",
	"soproc1":                         "soproc",
	"prefec_ber_avg_media_input1":     "prefecber",
	"errored_frames_avg_media_input1": "postfecber",
	"evm1":                            "evm",
}

func ConvertPmPrefixToThresholdPrefix(prefix string) string {
	if prefix == "uncorr_frames" {
		return "postfecber"
	} else if prefix == "cd" {
		return "cdshort"
	} else {
		return strings.Replace(prefix, "_", "", 1)
	}
}

func querySfpPM(intf string) map[string]string {
	firstSubport := getFirstSubPort(intf)
	if firstSubport == "" {
		log.Errorf("Unable to get first subport for %v while converting SFP status", intf)
		return map[string]string{
			"interface":   intf,
			"description": ZR_PM_NOT_APPLICABLE_STR,
		}
	}

	// Query PM info from STATE_DB
	queries := [][]string{
		{"STATE_DB", "TRANSCEIVER_PM", intf},
	}
	sfpPMDict, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get PM dict from STATE_DB: %v", err)
		return nil, err
	}

	// Query threshold info from STATE_DB
	queries = [][]string{
		{"STATE_DB", "TRANSCEIVER_DOM_THRESHOLD", intf},
	}
	sfpThresholdDict, err := GetMapFromQueries(queries)
	if err != nil {
		log.Errorf("Failed to get PM dict from STATE_DB: %v", err)
		return nil, err
	}

	convertVdmFieldsToLegacyFields(firstSubport, sfpThresholdDict, CCMIS_VDM_THRESHOLD_TO_LEGACY_DOM_THRESHOLD_MAP, "THRESHOLD")

	if len(sfpPMDict) > 0 {
		output := map[string]string{
			"interface":   intf,
			"description": "Min,Avg,Max,Threshold High Alarm,Threshold High Warning,Threshold Crossing Alert-High,Threshold Low Alarm,Threshold Low Warning,Threshold Crossing Alert-Low",
		}
		for paramName, info := range ZR_PM_INFO_MAP {
			unit := info.Unit
			prefix := info.Prefix
			row := ""

			// Collect values
			var values = make([]string, len(ZR_PM_VALUE_KEY_SUFFIXS))
			for _, suffix := range ZR_PM_VALUE_KEY_SUFFIXS {
				key := prefix + "_" + suffix
				if val, ok := sfpPMDict[key]; ok {
					if f, err := strconv.ParseFloat(val, 64); err == nil {
						values = append(values, BeautifyPmField(prefix, f, unit))
					} else {
						values = append(values, "N/A")
					}
				} else {
					values = append(values, "N/A")
				}
			}

			// Collect thresholds
			var thresholds = make([]string, len(ZR_PM_THRESHOLD_KEY_SUFFIXS))
			for _, suffix := range ZR_PM_THRESHOLD_KEY_SUFFIXS {
				key := ConvertPmPrefixToThresholdPrefix(prefix) + suffix
				if val, ok := sfpThresholdDict[key]; ok && val != "N/A" {
					if f, err := strconv.ParseFloat(val, 64); err == nil {
						thresholds = append(thresholds, BeautifyPmField(prefix, f, unit))
					} else {
						thresholds = append(thresholds, "N/A")
					}
				} else {
					thresholds = append(thresholds, "N/A")
				}
			}

			// TCA checks
			var tcaHigh, tcaLow bool
			if len(values) > 2 && len(thresholds) > 0 && thresholds[0] != 0 {
				tcaHigh = values[2] > thresholds[0]
			}
			if len(values) > 0 && len(thresholds) > 2 && thresholds[2] != 0 {
				tcaLow = values[0] < thresholds[2]
			}

			// Append fields
			for _, field := range append(values, thresholds[0:2]...) {
				row += field
				row += ","
			}
			row += fmt.Sprintf("%v,", tcaHigh)
			for _, field := range thresholds[2:] {
				row += field
				row += ","
			}
			row += fmt.Sprintf("%v", tcaLow)

			output[paramName] = row
		}

		return output
	} else {
		return map[string]string{
			"name":        intf,
			"description": ZR_PM_NOT_APPLICABLE_STR,
		}
	}
}

func getInterfaceTransceiverPM(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intf := args.At(0)

	result := make([]map[string]string, 0)
	if intf != "" {
		if ok, _ := isValidPhysicalPort(intf); !ok {
			log.Errorf("Invalid interface: %v!", intf)
			return nil, fmt.Errorf("Invalid interface: %v!", intf)
		}
		result = append(result, querySfpPM(intf))
	} else {
		queries := [][]string{
			{"APPL_DB", AppDBPortTable},
		}
		portTable, err := GetMapFromQueries(queries)
		if err != nil {
			log.Errorf("Failed to get interface list from APPL_DB: %v", err)
			return nil, err
		}

		ports := make([]string, 0, len(portTable))
		for key := range portTable {
			ports = append(ports, key)
		}
		ports = NatsortInterfaces(ports)

		for _, p := range ports {
			if ok, _ := isValidPhysicalPort(p); ok {
				result = append(result, querySfpPM(p))
			}
		}
    
	return json.Marshal(result)
}

func getInterfaceTransceiverStatus(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	intfArg := args.At(0)
	namingMode, _ := options[SonicCliIfaceMode].String()

	// APPL_DB PORT_TABLE -> determine valid ports
	portTable, err := GetMapFromQueries([][]string{{ApplDb, AppDBPortTable}})
	if err != nil {
		return nil, fmt.Errorf("failed to read PORT_TABLE: %w", err)
	}

	var ports []string
	if intfArg != "" {
		interfaceName, err := TryConvertInterfaceNameFromAlias(intfArg, namingMode)
		if err != nil {
			return nil, fmt.Errorf("alias conversion failed for %s: %w", intfArg, err)
		}
		if _, ok := portTable[interfaceName]; !ok {
			return nil, fmt.Errorf("invalid interface name %s", intfArg)
		}
		ports = []string{interfaceName}
	} else {
		for p := range portTable {
			ports = append(ports, p)
		}
		ports = NatsortInterfaces(ports)
	}

	result := make(map[string]string, len(ports))

	for _, p := range ports {
		if ok, _ := common.IsValidPhysicalPort(p); !ok {
			continue
		}
		statusStr := convertInterfaceSfpStatusToCliOutputString(p)
		result[p] = statusStr
	}

	return json.Marshal(result)
}
