package show_client

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	log "github.com/golang/glog"
	"github.com/google/shlex"
	natural "github.com/maruel/natural"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
	"gopkg.in/yaml.v2"
)

const ConfigDBPortTable = "PORT"
const AppDBPortTable = "PORT_TABLE"
const StateDBPortTable = "PORT_TABLE"
const ConfigDBPortChannelTable = "PORTCHANNEL"
const AppDBPortChannelTable = "LAG_TABLE"
const DefaultEmptyString = ""
const StateDb = "STATE_DB"
const ConfigDb = "CONFIG_DB"
const ApplDb = "APPL_DB"
const ConfigDbPort = "PORT"
const FDBTable = "FDB_TABLE"
const VlanSubInterfaceSeparator = '.'
const SonicCliIfaceMode = "SONIC_CLI_IFACE_MODE"

const (
	dbIndex    = 0 // The first index for a query will be the DB
	tableIndex = 1 // The second index for a query will be the table

	minQueryLength = 2 // We need to support TARGET/TABLE as a minimum query
	maxQueryLength = 5 // We can support up to 5 elements in query (TARGET/TABLE/(2 KEYS)/FIELD)

	hostNamespace              = "1" // PID 1 is the host init process
	defaultMissingCounterValue = "N/A"
	base10                     = 10
	maxShowCommandPeriod       = 300 // Max time allotted for SHOW commands period argument
)

var countersDBSeparator string

func init() {
	var err error
	countersDBSeparator, err = sdc.GetTableKeySeparator("COUNTERS_DB", "")
	if err != nil {
		log.Warningf("Failed to get table key separator for COUNTERS DB: %v\nUsing the default separator ':'.", err)
		countersDBSeparator = ":"
	}
}

func GetDataFromHostCommand(command string) (string, error) {
	baseArgs := []string{
		"--target", hostNamespace,
		"--pid", "--mount", "--uts", "--ipc", "--net",
		"--",
	}
	commandParts, err := shlex.Split(command)
	if err != nil {
		return "", err
	}
	cmdArgs := append(baseArgs, commandParts...)
	cmd := exec.Command("nsenter", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func GetDataFromFile(fileName string) ([]byte, error) {
	fileContent, err := sdc.ImplIoutilReadFile(fileName)
	if err != nil {
		log.Errorf("Failed to read'%v', %v", fileName, err)
		return nil, err
	}
	log.V(4).Infof("getDataFromFile, output: %v", string(fileContent))
	return fileContent, nil
}

func GetMapFromQueries(queries [][]string) (map[string]interface{}, error) {
	tblPaths, err := CreateTablePathsFromQueries(queries)
	if err != nil {
		return nil, err
	}
	msi := make(map[string]interface{})
	for _, tblPath := range tblPaths {
		err := sdc.TableData2Msi(&tblPath, false, nil, &msi)
		if err != nil {
			return nil, err
		}
	}
	return msi, nil
}

func GetDataFromQueries(queries [][]string) ([]byte, error) {
	msi, err := GetMapFromQueries(queries)
	if err != nil {
		return nil, err
	}
	return sdc.Msi2Bytes(msi)
}

func CreateTablePathsFromQueries(queries [][]string) ([]sdc.TablePath, error) {
	var allPaths []sdc.TablePath

	// Create and validate gnmi path then create table path
	for _, q := range queries {
		queryLength := len(q)
		if queryLength < minQueryLength || queryLength > maxQueryLength {
			return nil, fmt.Errorf("invalid query %v: must support at least [DB, table] or at most [DB, table, key1, key2, field]", q)
		}

		// Build a gNMI path for validation:
		//   prefix = { Target: dbTarget }
		//   path   = { Elem: [ {Name:table}, {Name:key}, {Name:field} ] }

		dbTarget := q[dbIndex]
		prefix := &gnmipb.Path{Target: dbTarget}

		table := q[tableIndex]
		elems := []*gnmipb.PathElem{{Name: table}}

		// Additional elements like keys and fields
		for i := tableIndex + 1; i < queryLength; i++ {
			elems = append(elems, &gnmipb.PathElem{Name: q[i]})
		}

		path := &gnmipb.Path{Elem: elems}

		if tablePaths, err := sdc.PopulateTablePaths(prefix, path); err != nil {
			return nil, fmt.Errorf("query %v failed: %w", q, err)
		} else {
			allPaths = append(allPaths, tablePaths...)
		}
	}
	return allPaths, nil
}

func ReadYamlToMap(filePath string) (map[string]interface{}, error) {
	yamlFile, err := sdc.ImplIoutilReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}
	var data map[string]interface{}
	err = yaml.Unmarshal(yamlFile, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	return data, nil
}

func ReadConfToMap(filePath string) (map[string]interface{}, error) {
	dataBytes, err := sdc.ImplIoutilReadFile(filePath)

	if err != nil {
		return nil, fmt.Errorf("failed to read CONF: %w", err)
	}

	confData := make(map[string]interface{})

	content := string(dataBytes)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			confData[key] = value
		}
	}

	return confData, nil
}

func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func RemapAliasToPortName(portData map[string]interface{}) map[string]interface{} {
	aliasMap := sdc.AliasToPortNameMap()
	remapped := make(map[string]interface{})

	needRemap := false

	for key := range portData {
		if _, isAlias := aliasMap[key]; isAlias {
			needRemap = true
			break
		}
	}

	if !needRemap { // Not an alias keyed map, no-op
		return portData
	}

	for alias, val := range portData {
		if portName, ok := aliasMap[alias]; ok {
			remapped[portName] = val
		}
	}
	return remapped
}

func RemapAliasToPortNameForQueues(queueData map[string]interface{}) map[string]interface{} {
	aliasMap := sdc.AliasToPortNameMap()
	remapped := make(map[string]interface{})

	for key, val := range queueData {
		port, queueIdx, found := strings.Cut(key, countersDBSeparator)
		if !found {
			log.Warningf("Ignoring the invalid queue '%v'", key)
			continue
		}
		if sonicPortName, ok := aliasMap[port]; ok {
			remapped[sonicPortName+countersDBSeparator+queueIdx] = val
		} else {
			remapped[key] = val
		}
	}

	return remapped
}

func GetValueOrDefault(values map[string]interface{}, key string, defaultValue string) string {
	if value, ok := values[key]; ok {
		return fmt.Sprint(value)
	}
	return defaultValue
}

func GetNonZeroValueOrEmpty(values map[string]interface{}, key string) string {
	if value, ok := values[key]; ok {
		if intValue, err := strconv.ParseInt(fmt.Sprint(value), base10, 64); err == nil && intValue != 0 {
			return fmt.Sprint(value)
		}
	}
	return ""
}

func GetFieldValueString(data map[string]interface{}, key string, defaultValue string, field string) string {
	entry, ok := data[key].(map[string]interface{})
	if !ok {
		return defaultValue
	}

	value, ok := entry[field]
	if !ok {
		return defaultValue
	}
	return fmt.Sprint(value)
}

func GetSumFields(data map[string]interface{}, key string, defaultValue string, fields ...string) (sum string) {
	defer func() {
		if r := recover(); r != nil {
			sum = defaultValue
		}
	}()
	var total int64
	for _, field := range fields {
		value := GetFieldValueString(data, key, defaultValue, field)
		if intValue, err := strconv.ParseInt(value, base10, 64); err != nil {
			return defaultValue
		} else {
			total += intValue
		}
	}
	return strconv.FormatInt(total, base10)
}

func calculateDiffCounters(oldCounter string, newCounter string, defaultValue string) string {
	if oldCounter == defaultValue || newCounter == defaultValue {
		return defaultValue
	}
	oldCounterValue, err := strconv.ParseInt(oldCounter, base10, 64)
	if err != nil {
		return defaultValue
	}
	newCounterValue, err := strconv.ParseInt(newCounter, base10, 64)
	if err != nil {
		return defaultValue
	}
	return strconv.FormatInt(newCounterValue-oldCounterValue, base10)
}

func natsortInterfaces(interfaces []string) []string {
	// Naturally sort the port list
	sort.Sort(natural.StringSlice(interfaces))
	return interfaces
}

// toString converts any value to string, returning the value directly if it is already a string.
func toString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		return fmt.Sprint(v)
	}
}

func GetSortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func ParseKey(key interface{}, delimiter string) (string, string) {
	keyStr, ok := key.(string)
	if !ok {
		log.Info("parse Key failure to convert key as string.")
		return "", ""
	}

	parts := strings.Split(keyStr, delimiter)
	if len(parts) < 2 {
		log.Info("Unable to parse the string")
		return "", ""
	}
	return parts[0], parts[1]
}

// GetInterfaceNameForDisplay returns alias when SONIC_CLI_IFACE_MODE=alias; otherwise the name.
// It also preserves VLAN sub-interface suffix like Ethernet0.100.
func GetInterfaceNameForDisplay(name string) string {
	if name == "" {
		return name
	}
	if interfaceNamingMode := GetInterfaceNamingMode(); interfaceNamingMode != "alias" {
		return name
	}

	nameToAlias := sdc.PortToAliasNameMap()

	base, suffix := name, ""
	if i := strings.IndexByte(name, VlanSubInterfaceSeparator); i >= 0 {
		base, suffix = name[:i], name[i:] // keep .<vlan>
	}

	if alias, ok := nameToAlias[base]; ok {
		return alias + suffix
	}
	return name
}

// GetInterfaceSwitchportMode returns the switchport mode.
func GetInterfaceSwitchportMode(
	portTbl, portChannelTbl, vlanMemberTbl map[string]interface{},
	name string,
) string {
	if m := GetFieldValueString(portTbl, name, "", "mode"); m != "" {
		return m
	}
	if m := GetFieldValueString(portChannelTbl, name, "", "mode"); m != "" {
		return m
	}
	for k := range vlanMemberTbl {
		_, member, ok := SplitCompositeKey(k)
		if ok && member == name {
			return "trunk"
		}
	}
	return "routed"
}

// SplitCompositeKey splits a two-part composite key using '|' or ':' delimiters.
// Returns left, right, true on success; empty strings and false otherwise.
// Examples:
//
//	"Vlan100|Ethernet0" -> ("Vlan100", "Ethernet0", true)
//	"PortChannel001:Ethernet4" -> ("PortChannel001", "Ethernet4", true)
func SplitCompositeKey(k string) (string, string, bool) {
	if parts := strings.Split(k, "|"); len(parts) == 2 {
		return parts[0], parts[1], true
	}
	if parts := strings.Split(k, ":"); len(parts) == 2 {
		return parts[0], parts[1], true
	}
	return "", "", false
}

// getOrDefault returns m[key] when present; otherwise returns def.
// Safe to call with a nil map. Handy for nested map lookups with explicit defaults.
func getOrDefault[T any](m map[string]T, key string, def T) T {
	if v, ok := m[key]; ok {
		return v
	}
	return def
}

// matchIPFamily checks if prefix belongs to desired family
func matchIPFamily(prefix string, wantIPv6 bool) bool {
	host := prefix
	if i := strings.IndexByte(prefix, '/'); i >= 0 {
		host = prefix[:i]
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	isV6 := ip.To4() == nil
	return isV6 == wantIPv6
}

// ContainsString returns true if target is present in list.
func ContainsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func GetInterfaceNamingMode() string {
	if mode := os.Getenv(SonicCliIfaceMode); mode != "" {
		return mode
	}
	return "default"
}
