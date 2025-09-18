package show_client

import (
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// All SHOW path and getters are defined here
func init() {
	// SHOW/buffer_pool
	sdc.RegisterCliPath(
		[]string{"SHOW", "buffer_pool", "persistent-watermark"},
		getBufferPoolPersistentWatermark,
		"SHOW/buffer_pool/persistent-watermark[OPTIONS]: Show persistent WM for buffer pools",
		0,
		0,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "buffer_pool", "watermark"},
		getBufferPoolWatermark,
		"SHOW/buffer_pool/watermark[OPTIONS]: Show user WM for buffer pools",
		0,
		0,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
	)

	// SHOW/clock
	sdc.RegisterCliPath(
		[]string{"SHOW", "clock"},
		getDate,
		"SHOW/clock[OPTIONS]: Show date and time",
		0,
		0,
		map[string]string{
			"timezones": "show/clock/timezones: List of available timezones",
		},
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "clock", "timezones"},
		getDateTimezone,
		"SHOW/clock/timezones[OPTIONS]: List of available timezones",
		0,
		0,
		nil,
		showCmdOptionVerbose,
	)

	// SHOW/dropcounters
	sdc.RegisterCliPath(
		[]string{"SHOW", "dropcounters", "capabilities"},
		getDropcountersCapabilities,
		"SHOW/dropcounters/capabilities[OPTIONS]: Show device drop counters capabilities",
		0,
		0,
		nil,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "dropcounters", "configuration"},
		getDropCountersConfiguration,
		"SHOW/dropcounters/configuration[OPTIONS]: Show current drop counter configuration",
		0,
		0,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionGroup,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "dropcounters", "counts"},
		getDropCounters,
		"SHOW/dropcounters/counts[OPTIONS]: Show drop counts",
		0,
		0,
		nil,
		showCmdOptionGroup,
		showCmdOptionCounterType,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionVerbose,
	)

	// SHOW/headroom-pool
	sdc.RegisterCliPath(
		[]string{"SHOW", "headroom-pool", "persistent-watermark"},
		getHeadroomPoolPersistentWatermark,
		"SHOW/headroom-pool/persistent-watermark[OPTIONS]: Show persistent WM for headroom pool",
		0,
		0,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "headroom-pool", "watermark"},
		getHeadroomPoolWatermark,
		"SHOW/headroom-pool/watermark[OPTIONS]: Show user WM for headroom pool",
		0,
		0,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
	)

	// SHOW/interfaces
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "alias"},
		getInterfaceAlias,
		"SHOW/interfaces/alias/{INTERFACENAME}[OPTIONS]: Show Interface Name/Alias Mapping",
		0,
		1,
		nil,
		showCmdOptionSonicCliIfaceMode,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionDisplay,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "counters"},
		getInterfaceCounters,
		"SHOW/interfaces/counters[OPTIONS]: Show interface counters",
		0,
		0,
		map[string]string{
			"detailed":      "show/interfaces/counters/detailed: Show interface counters detailed",
			"errors":        "show/interfaces/counters/errors: Show interface counters errors",
			"fec-histogram": "show/interfaces/counters/fec-histogram: Show interface counters fec-histogram",
			"fec-stats":     "show/interfaces/counters/fec-stats: Show interface counters rates",
			"rates":         "show/interfaces/counters/rates: Show interface counters rates",
			"rif":           "show/interfaces/counters/rif: Show interface counters rif",
			"trim":          "show/interfaces/counters/trim: Show interface counters trim",
		},
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionDisplay,
		showCmdOptionInterfaces,
		showCmdOptionPeriod,
		showCmdOptionJson,
		showCmdOptionVerbose,
	)

	// SHOW/interfaces/counters/rif
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "counters", "rif"},
		getInterfaceRifCounters,
		"SHOW/interfaces/counters/rif/{INTERFACENAME}[OPTIONS]",
		0,
		1,
		nil,
		showCmdOptionPeriod,
		showCmdOptionJson,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "description"},
		getInterfacesDescription,
		"SHOW/interfaces/description/{INTERFACENAME}[OPTIONS]: Show interface status, protocol and description",
		0,
		1,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		sdc.UnimplementedOption(showCmdOptionDisplay),
		showCmdOptionInterface, // TODO
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "errors"},
		getInterfaceErrors,
		"SHOW/interfaces/errors/{INTERFACENAME}[OPTIONS]: Show Interface Errors <interfacename>",
		1,
		1,
		nil,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "fec", "status"},
		getInterfaceFecStatus,
		"SHOW/interfaces/fec/status/{INTERFACENAME}[OPTIONS]: Show interface fec status",
		0,
		1,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionDisplay,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "flap"},
		getInterfaceFlap,
		"SHOW/interfaces/flap/{INTERFACENAME}[OPTIONS]: Show Interface Flap Information",
		0,
		1,
		nil,
		showCmdOptionSonicCliIfaceMode,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "neighbor", "expected"},
		getInterfaceNeighborExpected,
		"SHOW/interfaces/neighbor/expected/{INTERFACENAME}[OPTIONS]: Show expected neighbor information by interfaces",
		0,
		1,
		nil,
		showCmdOptionInterface, // TODO: Take as arg not option
		showCmdOptionSonicCliIfaceMode,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "naming_mode"},
		getInterfaceNamingMode,
		"SHOW/interfaces/naming_mode[OPTIONS]: Show interface naming_mode status",
		0,
		0,
		nil,
		showCmdOptionVerbose,
		showCmdOptionSonicCliIfaceMode,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "status"},
		getInterfaceStatus,
		"SHOW/interfaces/status/{INTERFACENAME}[OPTIONS]: Show Interface status information",
		0,
		1,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionDisplay,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "switchport", "config"},
		getInterfaceSwitchportConfig,
		"SHOW/interfaces/switchport/config[OPTIONS]: Show interface switchport config information",
		0,
		0,
		nil,
		showCmdOptionSonicCliIfaceMode,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "switchport", "status"},
		getInterfaceSwitchportStatus,
		"SHOW/interfaces/switchport/status[OPTIONS]: Show interface switchport status information",
		0,
		0,
		nil,
		showCmdOptionSonicCliIfaceMode,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "transceiver", "error-status"},
		getTransceiverErrorStatus,
		"SHOW/interfaces/transceiver/error-status/{INTERFACENAME}[OPTIONS]: Show transceiver error-status",
		0,
		1,
		nil,
		showCmdOptionVerbose,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		sdc.UnimplementedOption(showCmdOptionFetchFromHW),
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "transceiver", "presence"},
		getInterfaceTransceiverPresence,
		"SHOW/interfaces/transceiver/presence/{INTERFACENAME}[OPTIONS]: Show interface transceiver presence",
		0,
		1,
		nil,
		showCmdOptionVerbose,
		sdc.UnimplementedOption(showCmdOptionNamespace),
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "transceiver", "lpmode"},
		getInterfaceTransceiverLpMode,
		"SHOW/interfaces/transceiver/lpmode/{INTERFACENAME}[OPTIONS]: Show interface transceiver low-power mode",
		0,
		1,
		nil,
		showCmdOptionInterface, // TODO
	)

	// SHOW/ipv6
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "bgp", "neighbors"},
		getIPv6BGPNeighborsHandler,
		"SHOW/ipv6/bgp/neighbors/{IPADDRESS}/{routes|advertised-routes|received-routes}[OPTIONS]: Show IPv6 BGP neighbors",
		0,
		2,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "bgp", "network"},
		getIPv6BGPNetwork,
		"SHOW/ipv6/bgp/network/{ipv6-address|ipv6-prefix}/{bestpath|json|longer-prefixes|multipath}[OPTIONS]: Show BGP ipv6 network",
		0,
		2,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "bgp", "summary"},
		getIPv6BGPSummary,
		"SHOW/ipv6/bgp/summary[OPTIONS]: Show summarized information of IPv6 BGP state",
		0,
		0,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionDisplay,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "fib"},
		getIPv6Fib,
		"SHOW/ipv6/fib/{IPADDRESS}[OPTIONS]: Show IP FIB table",
		0,
		1,
		nil,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "interfaces"},
		getIPv6Interfaces,
		"SHOW/ipv6/interfaces[OPTIONS]: Show ipv6 interfaces",
		0,
		0,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionDisplay,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "link-local-mode"},
		getPortsIpv6LinkLocalMode,
		"SHOW/ipv6/link-local-mode[OPTIONS]: Show ipv6 link-local-mode",
		0,
		0,
		nil,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "protocol"},
		getIPv6Protocol,
		"SHOW/ipv6/protocol[OPTIONS]: Show IPv6 protocol information",
		0,
		0,
		nil,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "route"},
		getIPv6Route,
		"SHOW/ipv6/route/{IPADDRESS}/{VRF NAME}{...}[OPTIONS]: Show IPv6 routing table",
		0,
		-1,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionDisplay,
	)

	// SHOW/lldp
	sdc.RegisterCliPath(
		[]string{"SHOW", "lldp", "neighbors"},
		getLLDPNeighbors,
		"SHOW/lldp/neighbors/{INTERFACENAME}[OPTIONS]: Show LLDP neighbors",
		0,
		1,
		nil,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "lldp", "table"},
		getLLDPTable,
		"SHOW/lldp/table[OPTIONS]: Show LLDP neighbors in a tabular format",
		0,
		0,
		nil,
		showCmdOptionVerbose,
	)

	// SHOW/mac
	sdc.RegisterCliPath(
		[]string{"SHOW", "mac"},
		getMacTable,
		"SHOW/mac[OPTIONS]: Show MAC (FDB) entries",
		0,
		0,
		map[string]string{
			"aging-time": "show/mac/aging-time",
		},
		showCmdOptionVlan,
		showCmdOptionPort,
		showCmdOptionAddress,
		showCmdOptionType,
		showCmdOptionCount,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "mac", "aging-time"},
		getMacAgingTime,
		"SHOW/mac/aging-time[OPTIONS]: Show mac aging-time",
		0,
		0,
		nil,
	)

	// SHOW/mmu
	sdc.RegisterCliPath(
		[]string{"SHOW", "mmu"},
		getMmuConfig,
		"SHOW/mmu[OPTIONS]: Show mmu configuration",
		0,
		0,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionVerbose,
	)

	// SHOW/processes
	sdc.RegisterCliPath(
		[]string{"SHOW", "processes"},
		getProcessesRoot,
		"SHOW/processes/COMMAND[OPTIONS]: Show process information",
		0,
		0,
		map[string]string{
			"summary": "show/processes/summary: Show processses info",
			"cpu":     "show/processes/cpu: Show processes CPU info",
			"memory":  "show/processes/memory: Show processes information sorted by memory usage",
		},
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "processes", "memory"},
		getTopMemoryUsage,
		"SHOW/processes/memory[OPTIONS]: Show processes information sorted by memory usage",
		0,
		0,
		nil,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "processes", "summary"},
		getProcessesSummary,
		"SHOW/processes/summary[OPTIONS]: Show processes info",
		0,
		0,
		nil,
		showCmdOptionVerbose,
	)

	// SHOW/queue
	sdc.RegisterCliPath(
		[]string{"SHOW", "queue", "counters"},
		getQueueCounters,
		"SHOW/queue/counters/{INTERFACENAME}[OPTIONS]: Show queue counters",
		0,
		1,
		nil,
		showCmdOptionInterfaces,
		showCmdOptionDisplay,
		showCmdOptionNonzero,
		showCmdOptionAll,
		showCmdOptionTrim,
		sdc.UnimplementedOption(showCmdOptionVoq),
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionVerbose,
		showCmdOptionJson,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "queue", "wredcounters"},
		getQueueWredCounters,
		"SHOW/queue/wredcounters/{INTERFACENAME}[OPTIONS]: Show queue WRED counters",
		0,
		1,
		nil,
		showCmdOptionInterfaces,
		showCmdOptionDisplay,
		showCmdOptionNonzero,
		sdc.UnimplementedOption(showCmdOptionVoq),
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionVerbose,
		showCmdOptionJson,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "queue", "watermark"},
		getQueueUserWatermarks,
		"SHOW/queue/watermark/COMMAND[OPTIONS]: Show user WM for queues",
		0,
		0,
		map[string]string{
			"all":       "show/queue/watermark/all",
			"unicast":   "show/queue/watermark/unicast",
			"multicast": "show/queue/watermark/multicast",
		},
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "queue", "watermark", "all"},
		getQueueUserWatermarksAll,
		"SHOW/queue/watermark/all[OPTIONS]: Show user WM for unicast and multicast queues",
		0,
		0,
		nil,
		showCmdOptionInterfaces,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionJson,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "queue", "watermark", "unicast"},
		getQueueUserWatermarksUnicast,
		"SHOW/queue/watermark/unicast[OPTIONS]: Show user WM for unicast queues",
		0,
		0,
		nil,
		showCmdOptionInterfaces,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionJson,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "queue", "watermark", "multicast"},
		getQueueUserWatermarksMulticast,
		"SHOW/queue/watermark/multicast[OPTIONS]: Show user WM for multicast queues",
		0,
		0,
		nil,
		showCmdOptionInterfaces,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionJson,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "prefix-list"},
		getIPv6PrefixList,
		"SHOW/ipv6/prefix-list/{prefix_list_name}[OPTIONS]: Show IPv6 prefix-lists",
		0,
		1,
		nil,
		showCmdOptionVerbose,
	)
	// SHOW/reboot-cause
	sdc.RegisterCliPath(
		[]string{"SHOW", "reboot-cause"},
		getPreviousRebootCause,
		"SHOW/reboot-cause[OPTIONS]: Show cause of most recent reboot",
		0,
		0,
		map[string]string{
			"history": "show/reboot-cause/history: Show history of reboot-cause",
		},
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "reboot-cause", "history"},
		getRebootCauseHistory,
		"SHOW/reboot-cause/history[OPTIONS]: Show history of reboot-cause",
		0,
		0,
		nil,
	)

	// SHOW/services
	sdc.RegisterCliPath(
		[]string{"SHOW", "services"},
		getServices,
		"SHOW/services[OPTIONS]: Show all daemon services",
		0,
		0,
		nil,
	)

	// SHOW/srv6
	sdc.RegisterCliPath(
		[]string{"SHOW", "srv6", "stats"},
		getSRv6Stats,
		"SHOW/srv6/stats/{SID}[OPTIONS]: Show SRv6 counters statistics",
		0,
		1,
		nil,
		showCmdOptionVerbose,
	)

	// SHOW/system-memory
	sdc.RegisterCliPath(
		[]string{"SHOW", "system-memory"},
		getSystemMemory,
		"SHOW/system-memory[OPTIONS]: Show memory information",
		0,
		0,
		nil,
		showCmdOptionVerbose,
	)

	// SHOW/uptime
	sdc.RegisterCliPath(
		[]string{"SHOW", "uptime"},
		getUptime,
		"SHOW/uptime[OPTIONS]: Show system uptime",
		0,
		0,
		nil,
		showCmdOptionVerbose,
	)

	// SHOW/version
	sdc.RegisterCliPath(
		[]string{"SHOW", "version"},
		getVersion,
		"SHOW/version[OPTIONS]: Show version information",
		0,
		0,
		nil,
		showCmdOptionVerbose,
	)

	// SHOW/vlan
	sdc.RegisterCliPath(
		[]string{"SHOW", "vlan", "brief"},
		getVlanBrief,
		"SHOW/vlan/brief[OPTIONS]: Show all bridge information",
		0,
		0,
		nil,
		showCmdOptionVerbose,
	)

	// SHOW/watermark
	sdc.RegisterCliPath(
		[]string{"SHOW", "watermark", "telemetry", "interval"},
		getWatermarkTelemetryInterval,
		"SHOW/watermark/telemetry/interval[OPTIONS]: Show telemetry interval",
		0,
		0,
		nil,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interfaces", "portchannel"},
		getInterfacePortchannel,
		"SHOW/interfaces/portchannel[OPTIONS]: Show interface portchannel",
		0,
		0,
		nil,
		showCmdOptionVerbose,
		showCmdOptionSonicCliIfaceMode,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ecn"},
		getEcnProfiles,
		"SHOW/ecn[OPTIONS]: Show ECN profiles",
		0,
		0,
		nil,
		showCmdOptionVerbose,
	)
}
