package show_client

import (
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const (
	showCmdOptionUnimplementedDesc = "UNIMPLEMENTED"
	showCmdOptionDisplayDesc       = "[display=all] No-op since no-multi-asic support"
	showCmdOptionVerboseDesc       = "[verbose=true] Enable verbose output"
	showCmdOptionInterfacesDesc    = "[interfaces=TEXT] Filter by interfaces name"
	showCmdOptionInterfaceDesc     = "[interface=TEXT] Filter by single interface name"
	showCmdOptionPeriodDesc        = "[period=INTEGER] Display statistics over a specified period (in seconds)"
	showCmdOptionJsonDesc          = "[json=true] No-op since response is in json format"
	showCmdOptionVlanDesc          = "[vlan=INTEGER] Filter by VLAN ID"
	showCmdOptionPortDesc          = "[port=TEXT] Filter by interface or PortChannel name"
	showCmdOptionAddressDesc       = "[address=MAC] Filter by MAC address"
	showCmdOptionMacTypeDesc       = "[type=TEXT] Filter by MAC entry type: dynamic|static"
	showCmdOptionCountDesc         = "[count=true] Return only count"
)

var (
	showCmdOptionVerbose = sdc.NewShowCmdOption(
		"verbose",
		showCmdOptionVerboseDesc,
		sdc.BoolValue,
	)

	showCmdOptionNamespace = sdc.NewShowCmdOption(
		"namespace",
		showCmdOptionUnimplementedDesc,
		sdc.StringValue,
	)

	showCmdOptionDisplay = sdc.NewShowCmdOption(
		"display",
		showCmdOptionDisplayDesc,
		sdc.StringValue,
	)

	showCmdOptionInterfaces = sdc.NewShowCmdOption(
		"interfaces",
		showCmdOptionInterfacesDesc,
		sdc.StringSliceValue,
	)

	showCmdOptionPeriod = sdc.NewShowCmdOption(
		"period",
		showCmdOptionPeriodDesc,
		sdc.IntValue,
	)

	showCmdOptionJson = sdc.NewShowCmdOption(
		"json",
		showCmdOptionJsonDesc,
		sdc.BoolValue,
	)

	showCmdOptionInterface = sdc.NewShowCmdOption(
		"interface",
		showCmdOptionInterfaceDesc,
		sdc.StringValue,
	)

	showCmdOptionFetchFromHW = sdc.NewShowCmdOption(
		"fetch-from-hardware",
		showCmdOptionUnimplementedDesc,
		sdc.StringValue,
	)

	// MAC table specific options
	showCmdOptionVlan = sdc.NewShowCmdOption(
		"vlan",
		showCmdOptionVlanDesc,
		sdc.IntValue,
	)

	showCmdOptionPort = sdc.NewShowCmdOption(
		"port",
		showCmdOptionPortDesc,
		sdc.StringValue,
	)

	showCmdOptionAddress = sdc.NewShowCmdOption(
		"address",
		showCmdOptionAddressDesc,
		sdc.StringValue,
	)

	showCmdOptionMacType = sdc.NewShowCmdOption(
		"type",
		showCmdOptionMacTypeDesc,
		sdc.StringValue,
	)

	showCmdOptionCount = sdc.NewShowCmdOption(
		"count",
		showCmdOptionCountDesc,
		sdc.BoolValue,
	)
)
