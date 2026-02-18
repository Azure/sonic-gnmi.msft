package show_client

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	log "github.com/golang/glog"
	natural "github.com/maruel/natural"
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
	}

	// Get sessions info (config + state data merged)
	sessions, err := readSessionsInfo()
	if err != nil {
		log.Errorf("Failed to read sessions info: %v", err)
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
	return result, nil
}

// readSessionsInfo replicates the Python AclLoader.read_sessions_info() method
// For single ASIC only (no multi-npu support)
func readSessionsInfo() (map[string]map[string]interface{}, error) {
	// First, get CONFIG_DB data
	configQueries := [][]string{
		{common.ConfigDb, CFGMirrorSessionTable},
	}

	configResult, err := common.GetMapFromQueries(configQueries)
	if err != nil {
		log.Errorf("Failed to get config data from CONFIG_DB: %v", err)
		return nil, fmt.Errorf("failed to get config data: %v", err)
	}

	// Initialize sessions with config data
	sessions := make(map[string]map[string]interface{})
	for sessionName, sessionData := range configResult {
		if sessionMap, ok := sessionData.(map[string]interface{}); ok {
			sessions[sessionName] = sessionMap
		}
	}

	// Add STATE_DB data for each session
	for sessionName := range sessions {
		stateQueries := [][]string{
			{common.StateDb, StateMirrorSessionTable, sessionName},
		}

		stateResult, err := common.GetMapFromQueries(stateQueries)
		if err != nil {
			log.Errorf("Failed to get state data for session %s: %v", sessionName, err)
			// Set default values if state_db_info doesn't exist
			sessions[sessionName]["status"] = "error"
			sessions[sessionName]["monitor_port"] = ""
			continue
		}

		// When querying a specific key, GetMapFromQueries may return data directly or wrapped with session name
		var stateMap map[string]interface{}
		var hasStateData bool
		
		// Check if result has session name as key
		if stateData, exists := stateResult[sessionName]; exists {
			if sm, ok := stateData.(map[string]interface{}); ok {
				stateMap = sm
				hasStateData = true
			}
		} else if len(stateResult) > 0 {
			// If result is the state data directly
			stateMap = stateResult
			hasStateData = true
		}

		if hasStateData {
			if status, hasStatus := stateMap["status"]; hasStatus {
				sessions[sessionName]["status"] = status
			} else {
				sessions[sessionName]["status"] = "inactive"
			}

			if monitorPort, hasMonitorPort := stateMap["monitor_port"]; hasMonitorPort {
				sessions[sessionName]["monitor_port"] = monitorPort
			} else {
				sessions[sessionName]["monitor_port"] = ""
			}
		} else {
			sessions[sessionName]["status"] = "error"
			sessions[sessionName]["monitor_port"] = ""
		}
	}

	return sessions, nil
}

func processMirrorSessionData(sessions map[string]map[string]interface{}, sessionFilter string) (*MirrorSessionOutput, error) {
	output := &MirrorSessionOutput{
		ERSPANSessions: []ERSPANSession{},
		SPANSessions:   []SPANSession{},
	}

	// Get natsorted session names
	sessionNames := make([]string, 0, len(sessions))
	for name := range sessions {
		sessionNames = append(sessionNames, name)
	}
	sort.Sort(natural.StringSlice(sessionNames))

	for _, sessionName := range sessionNames {
		sessionInfo := sessions[sessionName]

		// Python: if session_name and key != session_name: continue
		if sessionFilter != "" && sessionName != sessionFilter {
			continue
		}

		// Extract values
		sessionType := common.GetValueOrDefault(sessionInfo, "type", "")
		status := common.GetValueOrDefault(sessionInfo, "status", "")

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
		} else {
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
		}
	}

	return output, nil
}
