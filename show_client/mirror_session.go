package show_client

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// Constants for mirror session tables
const (
	CFGMirrorSessionTable   = "MIRROR_SESSION"
	StateMirrorSessionTable = "MIRROR_SESSION_TABLE"
)

// MirrorSession represents a mirror session entry
type MirrorSession struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type,omitempty"`
	Status      interface{}           `json:"status,omitempty"` // Can be string or map[string]string for multi-ASIC
	SrcIP       string                 `json:"src_ip,omitempty"`
	DstIP       string                 `json:"dst_ip,omitempty"`
	GreType     string                 `json:"gre_type,omitempty"`
	DSCP        string                 `json:"dscp,omitempty"`
	TTL         string                 `json:"ttl,omitempty"`
	Queue       string                 `json:"queue,omitempty"`
	Policer     string                 `json:"policer,omitempty"`
	MonitorPort interface{}           `json:"monitor_port,omitempty"` // Can be string or map[string]string for multi-ASIC
	SrcPort     string                 `json:"src_port,omitempty"`
	DstPort     string                 `json:"dst_port,omitempty"`
	Direction   string                 `json:"direction,omitempty"`
}

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

	// CONFIG DB
	configQueries := [][]string{
		{common.ConfigDb, CFGMirrorSessionTable},
	}

	configData, err := common.GetMapFromQueries(configQueries)
	if err != nil {
		return nil, fmt.Errorf("failed to get config data: %v", err)
	}

	// STATE DB
	stateQueries := [][]string{
		{common.StateDb, StateMirrorSessionTable},
	}

	stateData, err := common.GetMapFromQueries(stateQueries)
	if err != nil {
		return nil, fmt.Errorf("failed to get state data: %v", err)
	}

	sessions, err := processMirrorSessionData(configData, stateData, sessionName)
	if err != nil {
		return nil, err
	}

	return json.Marshal(sessions)
}

func processMirrorSessionData(configData, stateData map[string]interface{}, sessionFilter string) (*MirrorSessionOutput, error) {
	output := &MirrorSessionOutput{
		ERSPANSessions: []ERSPANSession{},
		SPANSessions:   []SPANSession{},
	}

	// get config data
	var configSessions map[string]interface{}
	if cfg, ok := configData[common.ConfigDb]; ok {
		if table, ok := cfg.(map[string]interface{})[CFGMirrorSessionTable]; ok {
			if sessions, ok := table.(map[string]interface{}); ok {
				configSessions = sessions
			}
		}
	}

	// get state data 
	var stateSessions map[string]interface{}
	if state, ok := stateData[common.StateDb]; ok {
		if table, ok := state.(map[string]interface{})[StateMirrorSessionTable]; ok {
			if sessions, ok := table.(map[string]interface{}); ok {
				stateSessions = sessions
			}
		}
	}

	// Process each session in sorted order
	sortedSessionNames := common.GetSortedKeys(configSessions)
	for _, sessionName := range sortedSessionNames {
		sessionConfig := configSessions[sessionName]
		
		// if session name
		if sessionFilter != "" && sessionName != sessionFilter {
			continue
		}

		// parse config
		config, ok := sessionConfig.(map[string]interface{})
		if !ok {
			continue
		}

		session := MirrorSession{
			Name:      sessionName,
			Type:      common.GetValueOrDefault(config, "type", ""),
			SrcIP:     common.GetValueOrDefault(config, "src_ip", ""),
			DstIP:     common.GetValueOrDefault(config, "dst_ip", ""),
			GreType:   common.GetValueOrDefault(config, "gre_type", ""),
			DSCP:      common.GetValueOrDefault(config, "dscp", ""),
			TTL:       common.GetValueOrDefault(config, "ttl", ""),
			Queue:     common.GetValueOrDefault(config, "queue", ""),
			Policer:   common.GetValueOrDefault(config, "policer", ""),
			SrcPort:   common.GetValueOrDefault(config, "src_port", ""),
			DstPort:   common.GetValueOrDefault(config, "dst_port", ""),
			Direction: strings.ToLower(common.GetValueOrDefault(config, "direction", "")),
		}

		// get state info
		stateKey := StateMirrorSessionTable + "|" + sessionName
		if stateInfo, ok := stateSessions[stateKey]; ok {
			if stateMap, ok := stateInfo.(map[string]interface{}); ok {
				session.Status = common.GetValueOrDefault(stateMap, "status", "inactive")
				session.MonitorPort = common.GetValueOrDefault(stateMap, "monitor_port", "")
			}
		} else {
			session.Status = "error"
			session.MonitorPort = ""
		}

		// categorize by session type
		if session.Type == "SPAN" {
			spanSession := SPANSession{
				Name:      session.Name,
				Status:    fmt.Sprintf("%v", session.Status),
				DstPort:   session.DstPort,
				SrcPort:   session.SrcPort,
				Direction: session.Direction,
				Queue:     session.Queue,
				Policer:   session.Policer,
			}
			output.SPANSessions = append(output.SPANSessions, spanSession)
		} else {
			erspaSession := ERSPANSession{
				Name:        session.Name,
				Status:      fmt.Sprintf("%v", session.Status),
				SrcIP:       session.SrcIP,
				DstIP:       session.DstIP,
				GRE:         session.GreType,
				DSCP:        session.DSCP,
				TTL:         session.TTL,
				Queue:       session.Queue,
				Policer:     session.Policer,
				MonitorPort: fmt.Sprintf("%v", session.MonitorPort),
				SrcPort:     session.SrcPort,
				Direction:   session.Direction,
			}
			output.ERSPANSessions = append(output.ERSPANSessions, erspaSession)
		}
	}

	return output, nil
}
