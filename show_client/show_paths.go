package show_client

import (
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

// All SHOW path and getters are defined here
func init() {
	sdc.RegisterCliPath(
		[]string{"SHOW", "reboot-cause"},
		getPreviousRebootCause,
		map[string]string{
			"history": "show/reboot-cause/history: Show history of reboot-cause",
		},
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "reboot-cause", "history"},
		getRebootCauseHistory,
		nil,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "clock"},
		getDate,
		map[string]string{
			"timezones": "show/clock/timezones: List of available timezones",
		},
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "clock", "timezones"},
		getDateTimezone,
		nil,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "bgp", "summary"},
		getIPv6BGPSummary,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionDisplay,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "bgp", "neighbors"},
		getIPv6BGPNeighborsHandler,
		nil,
		showCmdOptionIPAddress,
		showCmdOptionInfoType,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interface", "counters"},
		getInterfaceCounters,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionDisplay,
		showCmdOptionInterfaces,
		showCmdOptionPeriod,
		showCmdOptionJson,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interface", "errors"},
		getInterfaceErrors,
		nil,
		sdc.RequiredOption(showCmdOptionInterface),
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interface", "fec", "status"},
		getInterfaceFecStatus,
		nil,
		showCmdOptionInterface,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		sdc.UnimplementedOption(showCmdOptionDisplay),
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "watermark", "telemetry", "interval"},
		getWatermarkTelemetryInterval,
		nil,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interface", "transceiver", "error-status"},
		getTransceiverErrorStatus,
		nil,
		showCmdOptionVerbose,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		sdc.UnimplementedOption(showCmdOptionFetchFromHW),
		showCmdOptionInterface,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interface", "transceiver", "presence"},
		getInterfaceTransceiverPresence,
		nil,
		showCmdOptionInterface,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interface", "status"},
		getInterfaceStatus,
		nil,
		showCmdOptionInterface,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		sdc.UnimplementedOption(showCmdOptionDisplay),
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "headroom-pool", "watermark"},
		getHeadroomPoolWatermark,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "headroom-pool", "persistent-watermark"},
		getHeadroomPoolPersistentWatermark,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "buffer_pool", "watermark"},
		getBufferPoolWatermark,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "buffer_pool", "persistent-watermark"},
		getBufferPoolPersistentWatermark,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "mmu"},
		getMmuConfig,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "mac", "aging-time"},
		getMacAgingTime,
		nil,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "interfaces"},
		getIPv6Interfaces,
		nil,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionDisplay,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "lldp", "table"},
		getLLDPTable,
		nil,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interface", "alias"},
		getInterfaceAlias,
		nil,
		showCmdOptionInterface,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "srv6", "stats"},
		getSRv6Stats,
		nil,
		showCmdOptionSid,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "queue", "counters"},
		getQueueCounters,
		nil,
		showCmdOptionInterfaces,
		showCmdOptionNonzero,
		showCmdOptionTrim,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "vlan", "brief"},
		getVlanBrief,
		nil,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "version"},
		getVersion,
		nil,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "protocol"},
		getIPv6Protocol,
		nil,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "link-local-mode"},
		getPortsIpv6LinkLocalMode,
		nil,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "fib"},
		getIPv6Fib,
		nil,
		showCmdOptionIPV6Address,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "mac"},
		getMacTable,
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
		[]string{"SHOW", "interface", "switchport", "config"},
		getInterfaceSwitchportConfig,
		nil,
		showCmdOptionInterface,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interface", "switchport", "status"},
		getInterfaceSwitchportStatus,
		nil,
		showCmdOptionInterface,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "dropcounters", "counts"},
		getDropCounters,
		nil,
		showCmdOptionGroup,
		showCmdOptionCounterType,
		sdc.UnimplementedOption(showCmdOptionNamespace),
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "dropcounters", "capabilities"},
		getDropcountersCapabilities,
		nil,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "system-memory"},
		getSystemMemory,
		nil,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "lldp", "neighbors"},
		getLLDPNeighbors,
		nil,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "processes"},
		getProcessesRoot,
		map[string]string{
			"summary": "show/processes/summary",
			"cpu":     "show/processes/cpu",
			"mem":     "show/processes/mem",
		},
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "processes", "summary"},
		getProcessesSummary,
		nil,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "ipv6", "bgp", "network"},
		getIPv6BGPNetwork,
		nil,
		showCmdOptionIPV6Address,
		showCmdOptionInfoTypeForBgpNetwork,
		sdc.UnimplementedOption(showCmdOptionNamespace),
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "uptime"},
		getUptime,
		nil,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interface", "flap"},
		getInterfaceFlap,
		nil,
		showCmdOptionInterface,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interface", "neighbor", "expected"},
		getInterfaceNeighborExpected,
		nil,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "interface", "naming_mode"},
		getInterfaceNamingMode,
		nil,
		showCmdOptionVerbose,
	)
	sdc.RegisterCliPath(
		[]string{"SHOW", "services"},
		getServices,
		nil,
	)
}
