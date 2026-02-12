package show_client

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// Constants for mirror session tables
const (
	CFGMirrorSessionTable   = "MIRROR_SESSION"
	StateMirrorSessionTable = "MIRROR_SESSION_TABLE"
)

// MirrorSessionOutput represents the complete output structure
type MirrorSessionOutput struct {
	ERSPANSessions []ERSPANSession `json:"erspan_sessions"`
	SPANSessions   []SPANSession   `json:"span_sessions"`
}

// ERSPANSession represents an ERSPAN session for table output
type ERSPANSession struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	SrcIP       string `json:"src_ip"`
	DstIP       string `json:"dst_ip"`
	GRE         string `json:"gre"`
	DSCP        string `json:"dscp"`
	TTL         string `json:"ttl"`
	Queue       string `json:"queue"`
	Policer     string `json:"policer"`
	MonitorPort string `json:"monitor_port"`
	SrcPort     string `json:"src_port"`
	Direction   string `json:"direction"`
}

// SPANSession represents a SPAN session for table output
type SPANSession struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	DstPort   string `json:"dst_port"`
	SrcPort   string `json:"src_port"`
	Direction string `json:"direction"`
	Queue     string `json:"queue"`
	Policer   string `json:"policer"`
}

func getMirrorSession(args sdc.CmdArgs, options sdc.OptionMap) ([]byte, error) {
	var sessionName string
	if len(args) > 0 {
		sessionName = args[0]
		fmt.Printf("DEBUG: Filtering for session: %s\n", sessionName)
	}

	// Get sessions info (config + state data merged)
	sessions, err := readSessionsInfo()
	if err != nil {
		fmt.Printf("DEBUG: readSessionsInfo failed: %v\n", err)
		return nil, fmt.Errorf("failed to read sessions info: %v", err)
	}

	output, err := processMirrorSessionData(sessions, sessionName)
	if err != nil {
		return nil, err
	}

	result, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}
	fmt.Printf("DEBUG: Generated JSON with %d ERSPAN and %d SPAN sessions\n", len(output.ERSPANSessions), len(output.SPANSessions))
	return result, nil
}

// readSessionsInfo replicates the Python AclLoader.read_sessions_info() method
// For single ASIC only (no multi-npu support)
func readSessionsInfo() (map[string]map[string]interface{}, error) {
	// First, get CONFIG_DB data
	configQueries := [][]string{
		{common.ConfigDb, CFGMirrorSessionTable},
	}
	fmt.Printf("DEBUG: Executing query for CONFIG_DB.%s\n", CFGMirrorSessionTable)

	configResult, err := common.GetMapFromQueries(configQueries)
	if err != nil {
		fmt.Printf("DEBUG: GetMapFromQueries for CONFIG_DB failed: %v\n", err)
		return nil, fmt.Errorf("failed to get config data: %v", err)
	}
	fmt.Printf("DEBUG: Found %d sessions in CONFIG_DB\n", len(configResult))

	// Initialize sessions with config data
	sessions := make(map[string]map[string]interface{})
	for sessionName, sessionData := range configResult {
		if sessionMap, ok := sessionData.(map[string]interface{}); ok {
			sessions[sessionName] = sessionMap
			fmt.Printf("DEBUG: Added config for session %s\n", sessionName)
		}
	}

	// Now enrich with STATE_DB data for each session
	// Python logic: state_db_info = self.statedb.get_all(self.statedb.STATE_DB, "{}|{}".format(self.STATE_MIRROR_SESSION_TABLE, key))
	for sessionName := range sessions {
		stateQueries := [][]string{
			{common.StateDb, StateMirrorSessionTable, sessionName},
		}
		fmt.Printf("DEBUG: Querying STATE_DB for session %s\n", sessionName)
		
		stateResult, err := common.GetMapFromQueries(stateQueries)
		if err != nil {
			fmt.Printf("DEBUG: GetMapFromQueries for STATE_DB session %s failed: %v\n", sessionName, err)
			// Set default values as per Python logic: "error" if state_db_info doesn't exist
			sessions[sessionName]["status"] = "error"
			sessions[sessionName]["monitor_port"] = ""
			continue
		}

		// When querying a specific key, GetMapFromQueries may return data directly or wrapped with session name
		// Try both approaches
		var stateMap map[string]interface{}
		var hasStateData bool
		
		// First try: check if result has session name as key
		if stateData, exists := stateResult[sessionName]; exists {
			if sm, ok := stateData.(map[string]interface{}); ok {
				stateMap = sm
				hasStateData = true
			}
		} else if len(stateResult) > 0 {
			// Second try: result might be the state data directly
			stateMap = stateResult
			hasStateData = true
		}

		if hasStateData {
			// Python logic: state_db_info.get("status", "inactive") if state_db_info else "error"
			if status, hasStatus := stateMap["status"]; hasStatus {
				sessions[sessionName]["status"] = status
			} else {
				sessions[sessionName]["status"] = "inactive"
			}

			// Python logic: state_db_info.get("monitor_port", "") if state_db_info else ""
			if monitorPort, hasMonitorPort := stateMap["monitor_port"]; hasMonitorPort {
				sessions[sessionName]["monitor_port"] = monitorPort
			} else {
				sessions[sessionName]["monitor_port"] = ""
			}
			fmt.Printf("DEBUG: Added state data for session %s (status=%v, monitor_port=%v)\n", sessionName, stateMap["status"], stateMap["monitor_port"])
		} else {
			// No state data found, set defaults as per Python logic
			sessions[sessionName]["status"] = "error"
			sessions[sessionName]["monitor_port"] = ""
			fmt.Printf("DEBUG: No state data found for session %s, set status=error\n", sessionName)
		}
	}

	fmt.Printf("DEBUG: Total sessions with merged data: %d\n", len(sessions))
	return sessions, nil
}

func processMirrorSessionData(sessions map[string]map[string]interface{}, sessionFilter string) (*MirrorSessionOutput, error) {
	fmt.Printf("DEBUG: Processing %d sessions, filter: '%s'\n", len(sessions), sessionFilter)
	output := &MirrorSessionOutput{
		ERSPANSessions: []ERSPANSession{},
		SPANSessions:   []SPANSession{},
	}

	// Get sorted session names for consistent output (matches Python's natsorted)
	sessionNames := make([]string, 0, len(sessions))
	for name := range sessions {
		sessionNames = append(sessionNames, name)
	}
	sort.Strings(sessionNames)

	for _, sessionName := range sessionNames {
		sessionInfo := sessions[sessionName]
		
		// Log all fields for debugging
		fmt.Printf("DEBUG: Session %s fields:\n", sessionName)
		for key, value := range sessionInfo {
			fmt.Printf("  %s = %v (type: %T)\n", key, value, value)
		}
		
		// Python: if session_name and key != session_name: continue
		if sessionFilter != "" && sessionName != sessionFilter {
			continue
		}

		// Extract values - Python uses val.get("field", "") with .lower() for direction
		sessionType := common.GetValueOrDefault(sessionInfo, "type", "")
		status := common.GetValueOrDefault(sessionInfo, "status", "")
		
		fmt.Printf("DEBUG: Processing session %s, type: %s, status: %s\n", sessionName, sessionType, status)
		
		// Python: if val.get("type") == "SPAN":
		if sessionType == "SPAN" {
			spanSession := SPANSession{
				Name:      sessionName,
				Status:    status,
				DstPort:   common.GetValueOrDefault(sessionInfo, "dst_port", ""),
				SrcPort:   common.GetValueOrDefault(sessionInfo, "src_port", ""),
				Direction: strings.ToLower(common.GetValueOrDefault(sessionInfo, "direction", "")),
				Queue:     common.GetValueOrDefault(sessionInfo, "queue", ""),
				Policer:   common.GetValueOrDefault(sessionInfo, "policer", ""),
			}
			output.SPANSessions = append(output.SPANSessions, spanSession)
			fmt.Printf("DEBUG: Added SPAN session %s\n", sessionName)
		} else {
			// Python: else: (defaults to ERSPAN for any non-SPAN type)
			erspanSession := ERSPANSession{
				Name:        sessionName,
				Status:      status,
				SrcIP:       common.GetValueOrDefault(sessionInfo, "src_ip", ""),
				DstIP:       common.GetValueOrDefault(sessionInfo, "dst_ip", ""),
				GRE:         common.GetValueOrDefault(sessionInfo, "gre_type", ""),
				DSCP:        common.GetValueOrDefault(sessionInfo, "dscp", ""),
				TTL:         common.GetValueOrDefault(sessionInfo, "ttl", ""),
				Queue:       common.GetValueOrDefault(sessionInfo, "queue", ""),
				Policer:     common.GetValueOrDefault(sessionInfo, "policer", ""),
				MonitorPort: common.GetValueOrDefault(sessionInfo, "monitor_port", ""),
				SrcPort:     common.GetValueOrDefault(sessionInfo, "src_port", ""),
				Direction:   strings.ToLower(common.GetValueOrDefault(sessionInfo, "direction", "")),
			}
			output.ERSPANSessions = append(output.ERSPANSessions, erspanSession)
			fmt.Printf("DEBUG: Added ERSPAN session %s\n", sessionName)
		}
	}

	fmt.Printf("DEBUG: Final output - ERSPAN: %d, SPAN: %d\n", len(output.ERSPANSessions), len(output.SPANSessions))
	return output, nil
}
