package client

type OptionType int
type ValueType int

type ShowCmdOption struct {
	optName     string
	optType     OptionType // 0 means required, 1 means optional, -1 means unimplemented, all other values means invalid argument
	description string     // will be used in help output
	valueType   ValueType
	enumValues  []string // valid only when valueType is EnumValue
	hidden      bool     // when true, exclude from help output
}

type OptionValue struct {
	value interface{}
}

type CmdArgs []string

type OptionMap map[string]OptionValue

type DataGetter func(args CmdArgs, options OptionMap) ([]byte, error)

type TablePath = tablePath

type ShowPathConfig struct {
	dataGetter  DataGetter
	options     map[string]ShowCmdOption
	description map[string]map[string]string
	minArgs     int // 0 means no args required, all numbers greater are required
	maxArgs     int // 0 means no args allowed, -1 means any number of args
	regLen      int // length of registered prefix
}

var (
	showCmdOptionHelp = NewShowCmdOption(
		"help",
		showCmdOptionHelpDesc,
		BoolValue,
	)

	showCmdOptionRedact = HiddenOption(
		NewShowCmdOption(
			"redact",
			"",
			BoolValue,
		),
	)
)

// registeredGlobalOptions defines options that are automatically available to all show commands
var registeredGlobalOptions = []ShowCmdOption{
	showCmdOptionHelp,
	showCmdOptionRedact,
}

const (
	StringValue      ValueType = 0
	StringSliceValue ValueType = 1
	BoolValue        ValueType = 2
	IntValue         ValueType = 3
	EnumValue        ValueType = 4

	Required      OptionType = 0
	Optional      OptionType = 1
	Unimplemented OptionType = -1

	showCmdOptionHelpDesc = "[help=true]Show this message"
)

func (args CmdArgs) At(index int) string {
	val := ""
	if len(args) > index {
		val = args[index]
	}
	return val
}

func (ov OptionValue) String() (string, bool) {
	s, ok := ov.value.(string)
	return s, ok
}

func (ov OptionValue) Strings() ([]string, bool) {
	ss, ok := ov.value.([]string)
	return ss, ok
}

func (ov OptionValue) Bool() (bool, bool) {
	b, ok := ov.value.(bool)
	return b, ok
}

func (ov OptionValue) Int() (int, bool) {
	i, ok := ov.value.(int)
	return i, ok
}

func NewShowCmdOption(name string, desc string, valType ValueType, enumVals ...string) ShowCmdOption {
	return ShowCmdOption{
		optName:     name,
		optType:     Optional,
		description: desc,
		valueType:   valType,
		enumValues:  enumVals,
	}
}

func RequiredOption(option ShowCmdOption) ShowCmdOption {
	option.optType = Required
	return option
}

func UnimplementedOption(option ShowCmdOption) ShowCmdOption {
	option.optType = Unimplemented
	return option
}

func HiddenOption(option ShowCmdOption) ShowCmdOption {
	option.hidden = true
	option.description = "(hidden)"
	return option
}
