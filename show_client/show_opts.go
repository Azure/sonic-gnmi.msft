package show_client

import (
	"github.com/sonic-net/sonic-gnmi/show_client/common"
	sdc "github.com/sonic-net/sonic-gnmi/sonic_data_client"
)

const (
	showCmdOptionUnimplementedDesc     = "UNIMPLEMENTED"
	showCmdOptionDisplayDesc           = "[display=all] No-op since no-multi-asic support"
	showCmdOptionVerboseDesc           = "[verbose=true] Enable verbose output"
	showCmdOptionQueueInterfacesDesc   = "[interfaces=TEXT] Filter by interfaces name"
	showCmdOptionInterfacesDesc        = "[interface=TEXT] Filter by interfaces name"
	showCmdOptionInterfaceDesc         = "[interface=TEXT] Filter by single interface name"
	showCmdOptionPortDesc              = "[port=TEXT] Filter by single port name"
	showCmdOptionVlanDesc              = "[vlan=INTEGER] Filter by VLAN ID"
	showCmdOptionAddressDesc           = "[address=TEXT] Filter by MAC address"
	showCmdOptionTypeDesc              = "[type=TEXT] Filter by MAC type (static/dynamic)"
	showCmdOptionCountDesc             = "[count=true] Only show the count of MAC addresses"
	showCmdOptionDomDesc               = "[dom=false] Also display Digital Optical Monitoring (DOM) data"
	showCmdOptionPeriodDesc            = "[period=INTEGER] Display statistics over a specified period (in seconds)"
	showCmdOptionJsonDesc              = "[json=true] No-op since response is in json format"
	showCmdOptionSidDesc               = "[sid=TEXT] Filter by SRv6 SID"
	showCmdOptionNonzeroDesc           = "[nonzero=true] Display only non-zero values"
	showCmdOptionTrimDesc              = "[trim=true] Display only trim counters"
	showCmdOptionGroupDesc             = "[group=TEXT] Filter by logical counter group (eg RX_DROPS, TX_ERR)"
	showCmdOptionCounterTypeDesc       = "[counter_type=TEXT] Filter by counter type (eg PORT_INGRESS_DROPS, SWITCH_EGRESS_DROPS)"
	showCmdOptionIPAddressDesc         = "[ipaddress=TEXT] Filter by single IP address"
	showCmdOptionIPV6AddressDesc       = "[ipaddress=TEXT] Filter by IPv6 address"
	showCmdOptionInfoTypeDesc          = "[info_type=TEXT] Filter by information type"
	showCmdOptionSonicCliIfaceModeDesc = "[SONIC_CLI_IFACE_MODE=TEXT] Filter by sonic interface naming mode (eg alias/default)"
	showCmdOptionPrintAllDesc          = "[printall=true] Show all counters"
	showCmdOptionAllDesc               = "[all=true] No-op since all queue counters are shown by default"
	showCmdOptionIfaceDesc             = "[iface=TEXT] Filter by interface name"
	showCmdOptionLinesDesc             = "[lines=INTEGER] Number of lines to show (default: 10)"
)

// Option keys
const (
	OptionKeyVerbose  = "verbose"
	SonicCliIfaceMode = "SONIC_CLI_IFACE_MODE"
)

var (
	showCmdOptionVerbose = sdc.NewShowCmdOption(
		OptionKeyVerbose,
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

	showCmdOptionQueueInterfaces = sdc.NewShowCmdOption(
		"interfaces",
		showCmdOptionQueueInterfacesDesc,
		sdc.StringSliceValue,
	)

	showCmdOptionInterfaces = sdc.NewShowCmdOption(
		"interface",
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

	showCmdOptionPort = sdc.NewShowCmdOption(
		"port",
		showCmdOptionPortDesc,
		sdc.StringValue,
	)

	// MAC-show specific options
	showCmdOptionVlan = sdc.NewShowCmdOption(
		"vlan",
		showCmdOptionVlanDesc,
		sdc.IntValue,
	)

	showCmdOptionAddress = sdc.NewShowCmdOption(
		"address",
		showCmdOptionAddressDesc,
		sdc.StringValue,
	)

	showCmdOptionType = sdc.NewShowCmdOption(
		"type",
		showCmdOptionTypeDesc,
		sdc.StringValue,
	)

	showCmdOptionCount = sdc.NewShowCmdOption(
		"count",
		showCmdOptionCountDesc,
		sdc.BoolValue,
	)

	showCmdOptionDom = sdc.NewShowCmdOption(
		"dom",
		showCmdOptionDomDesc,
		sdc.BoolValue,
	)

	showCmdOptionFetchFromHW = sdc.NewShowCmdOption(
		"fetch-from-hardware",
		showCmdOptionUnimplementedDesc,
		sdc.StringValue,
	)

	showCmdOptionSid = sdc.NewShowCmdOption(
		"sid",
		showCmdOptionSidDesc,
		sdc.StringValue,
	)

	showCmdOptionNonzero = sdc.NewShowCmdOption(
		"nonzero",
		showCmdOptionNonzeroDesc,
		sdc.BoolValue,
	)

	showCmdOptionTrim = sdc.NewShowCmdOption(
		"trim",
		showCmdOptionTrimDesc,
		sdc.BoolValue,
	)

	showCmdOptionGroup = sdc.NewShowCmdOption(
		"group",
		showCmdOptionGroupDesc,
		sdc.StringValue,
	)

	showCmdOptionCounterType = sdc.NewShowCmdOption(
		"counter_type",
		showCmdOptionCounterTypeDesc,
		sdc.StringValue,
	)

	showCmdOptionIPAddress = sdc.NewShowCmdOption(
		"ipaddress",
		showCmdOptionIPAddressDesc,
		sdc.StringValue,
	)

	showCmdOptionInfoTypeForBgpNetwork = sdc.NewShowCmdOption(
		"info_type",
		showCmdOptionInfoTypeDesc,
		sdc.StringValue,
	)

	showCmdOptionSonicCliIfaceMode = sdc.NewShowCmdOption(
		SonicCliIfaceMode,
		showCmdOptionSonicCliIfaceModeDesc,
		sdc.EnumValue,
		common.Default.String(),
		common.Alias.String(),
	)

	showCmdOptionPrintAll = sdc.NewShowCmdOption(
		"printall",
		showCmdOptionPrintAllDesc,
		sdc.BoolValue,
	)

	showCmdOptionAll = sdc.NewShowCmdOption(
		"all",
		showCmdOptionAllDesc,
		sdc.BoolValue,
	)

	showCmdOptionVoq = sdc.NewShowCmdOption(
		"voq",
		showCmdOptionUnimplementedDesc,
		sdc.BoolValue,
	)

	showCmdOptionIface = sdc.NewShowCmdOption(
		"iface",
		showCmdOptionIfaceDesc,
		sdc.StringValue,
	)

	showCmdOptionLines = sdc.NewShowCmdOption(
		"lines",
		showCmdOptionLinesDesc,
		sdc.IntValue,
	)
)

