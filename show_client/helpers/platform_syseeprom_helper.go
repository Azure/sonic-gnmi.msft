package helpers

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

// --- EEPROM / syseeprom ---

const (
	eepromInfoTable = "EEPROM_INFO"

	// Special EEPROM_INFO keys (not TLV codes)
	eepromStateKey     = "State"
	eepromTlvHeaderKey = "TlvHeader"
	eepromChecksumKey  = "Checksum"
)

// Default TLV type codes (matching Python TlvInfoDecoder).
const (
	TlvCodeProductName   = 0x21
	TlvCodePartNumber    = 0x22
	TlvCodeSerialNumber  = 0x23
	TlvCodeMacBase       = 0x24
	TlvCodeManufDate     = 0x25
	TlvCodeDeviceVersion = 0x26
	TlvCodeLabelRevision = 0x27
	TlvCodePlatformName  = 0x28
	TlvCodeOnieVersion   = 0x29
	TlvCodeMacSize       = 0x2A
	TlvCodeManufName     = 0x2B
	TlvCodeManufCountry  = 0x2C
	TlvCodeVendorName    = 0x2D
	TlvCodeDiagVersion   = 0x2E
	TlvCodeServiceTag    = 0x2F
	TlvCodeVendorExt     = 0xFD
	TlvCodeCrc32         = 0xFE
)

// eepromTlvCodes lists TLV codes in display order.
// Built in init() to mirror Python: range(PRODUCT_NAME, SERVICE_TAG+1) + VENDOR_EXT + CRC_32.
var eepromTlvCodes []int

func init() {
	for code := TlvCodeProductName; code <= TlvCodeServiceTag; code++ {
		eepromTlvCodes = append(eepromTlvCodes, code)
	}
	eepromTlvCodes = append(eepromTlvCodes, TlvCodeVendorExt)
	eepromTlvCodes = append(eepromTlvCodes, TlvCodeCrc32)
}

// SysEepromInfo is the top-level JSON response for "show platform syseeprom".
type SysEepromInfo struct {
	TlvInfoHeader SysEepromHeader `json:"tlv_info_header"`
	TlvList       []SysEepromTlv  `json:"tlv_list"`
	ChecksumValid bool            `json:"checksum_valid"`
}

// SysEepromHeader holds the TlvInfo header fields.
type SysEepromHeader struct {
	IdString    string `json:"id_string"`
	Version     string `json:"version"`
	TotalLength string `json:"total_length"`
}

// SysEepromTlv represents one TLV entry in the EEPROM.
type SysEepromTlv struct {
	Name   string `json:"tlv_name"`
	Code   string `json:"code"`
	Length string `json:"length"`
	Value  string `json:"value"`
}

// PlatformsWithoutEepromDb lists platforms that do not support EEPROM DB
// (must fall back to platform API via nsenter).
var PlatformsWithoutEepromDb = []string{`(?i).*arista.*`, `(?i).*kvm.*`}

// PlatformsWithoutEeprom lists platforms that do not support EEPROM at all.
var PlatformsWithoutEeprom = []string{`(?i).*kvm.*`}

// MatchesPlatformPattern checks if platform matches any of the given regex patterns.
func MatchesPlatformPattern(platform string, patterns []string) bool {
	for _, p := range patterns {
		if matched, _ := regexp.MatchString(p, platform); matched {
			return true
		}
	}
	return false
}

// ReadEepromViaPlatformApi reads EEPROM data by invoking the sonic_platform
// Python API on the host via nsenter (for platforms without EEPROM DB support).
// Returns the raw text output from decode_eeprom() — format varies by vendor.
func ReadEepromViaPlatformApi() ([]byte, error) {
	escaped := strings.ReplaceAll(common.SysEepromPyScript, "'", `'\''`)
	pyCmd := fmt.Sprintf("python3 -c '%s'", escaped)
	output, err := common.GetDataFromHostCommand(pyCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to read EEPROM via platform API: %w", err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return nil, fmt.Errorf("empty response from platform EEPROM API")
	}

	// Wrap raw text in JSON since gNMI response uses JsonIetfVal
	result := map[string]string{"eeprom_raw": output}
	return json.Marshal(result)
}

// ReadEepromFromDb reads EEPROM data cached in STATE_DB by syseepromd.
func ReadEepromFromDb() ([]byte, error) {
	// Bulk-read all EEPROM_INFO entries from STATE_DB
	queries := [][]string{
		{common.StateDb, eepromInfoTable},
	}
	allData, err := common.GetMapFromQueries(queries)
	if err != nil {
		return nil, fmt.Errorf("failed to query EEPROM_INFO from STATE_DB: %w", err)
	}

	// Check initialization state
	stateData, ok := allData[eepromStateKey].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("EEPROM data not available - syseepromd may not have initialized yet")
	}
	initialized := common.GetValueOrDefault(stateData, "Initialized", "")
	if initialized != "1" {
		return nil, fmt.Errorf("EEPROM data not initialized (Initialized=%q) - syseepromd may still be starting", initialized)
	}

	// Parse TlvHeader
	header := SysEepromHeader{
		IdString:    "N/A",
		Version:     "N/A",
		TotalLength: "N/A",
	}
	if headerData, ok := allData[eepromTlvHeaderKey].(map[string]interface{}); ok {
		header.IdString = common.GetValueOrDefault(headerData, "Id String", "N/A")
		header.Version = common.GetValueOrDefault(headerData, "Version", "N/A")
		header.TotalLength = common.GetValueOrDefault(headerData, "Total Length", "N/A")
	}

	// Parse TLV entries in defined order
	tlvList := make([]SysEepromTlv, 0)
	for _, code := range eepromTlvCodes {
		codeStr := fmt.Sprintf("0x%02X", code)
		tlvData, ok := allData[strings.ToLower(codeStr)].(map[string]interface{})
		if !ok {
			continue
		}

		if code == TlvCodeVendorExt {
			// Vendor Extension: multiple sub-entries
			tlvList = append(tlvList, parseVendorExtensions(codeStr, tlvData)...)
		} else {
			tlv := SysEepromTlv{
				Code:   codeStr,
				Name:   common.GetValueOrDefault(tlvData, "Name", "N/A"),
				Length: common.GetValueOrDefault(tlvData, "Len", "N/A"),
				Value:  common.GetValueOrDefault(tlvData, "Value", "N/A"),
			}
			tlvList = append(tlvList, tlv)
		}
	}

	// Parse checksum validity
	checksumValid := false
	if checksumData, ok := allData[eepromChecksumKey].(map[string]interface{}); ok {
		validStr := common.GetValueOrDefault(checksumData, "Valid", "0")
		checksumValid = (validStr == "1")
	}

	result := SysEepromInfo{
		TlvInfoHeader: header,
		TlvList:       tlvList,
		ChecksumValid: checksumValid,
	}
	return json.Marshal(result)
}

// parseVendorExtensions expands the vendor extension TLV (0xFD) which can contain
// multiple sub-entries indexed by Name_0, Len_0, Value_0, Name_1, etc.
func parseVendorExtensions(codeStr string, data map[string]interface{}) []SysEepromTlv {
	numStr := common.GetValueOrDefault(data, "Num_vendor_ext", "0")
	numVendorExt, err := strconv.Atoi(numStr)
	if err != nil || numVendorExt <= 0 {
		log.V(2).Infof("parseVendorExtensions: no vendor extensions (Num_vendor_ext=%q)", numStr)
		return nil
	}

	tlvs := make([]SysEepromTlv, 0, numVendorExt)
	for i := 0; i < numVendorExt; i++ {
		name := common.GetValueOrDefault(data, fmt.Sprintf("Name_%d", i), "")
		length := common.GetValueOrDefault(data, fmt.Sprintf("Len_%d", i), "")
		value := common.GetValueOrDefault(data, fmt.Sprintf("Value_%d", i), "")
		if name == "" && length == "" && value == "" {
			log.V(2).Infof("parseVendorExtensions: skipping empty entry at index %d", i)
			continue
		}
		tlvs = append(tlvs, SysEepromTlv{
			Code:   codeStr,
			Name:   name,
			Length: length,
			Value:  value,
		})
	}
	return tlvs
}
