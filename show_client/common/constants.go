package common

const (
	DefaultMissingCounterValue = "N/A"
	Base10                     = 10
	MaxShowCommandPeriod       = 300 // Max time allotted for SHOW commands period argument
	ExternalPort               = "Ext"
	InternalPort               = "Int"
	InbandPort                 = "Inb"
	RecircPort                 = "Rec"
	DpuConnectPort             = "Dpc"
	RJ45PortType               = "RJ45"
	OptionKeyVerbose           = "verbose"
)

var SonicInterfacePrefixes = map[string]string{
	"Ethernet-FrontPanel": "Ethernet",
	"PortChannel":         "PortChannel",
	"Vlan":                "Vlan",
	"Loopback":            "Loopback",
	"Ethernet-Backplane":  "Ethernet-BP",
	"Ethernet-Inband":     "Ethernet-IB",
	"Ethernet-Recirc":     "Ethernet-Rec",
	"Ethernet-SubPort":    "Eth",
	"PortChannel-SubPort": "Po",
}
