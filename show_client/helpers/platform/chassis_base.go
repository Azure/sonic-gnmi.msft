package platform

/*
    chassis_base.go

    Base interface for implementing a platform-specific code with which
    to interact with a chassis device in SONiC.

	Vendor implementations (e.g., mellanox.Chassis) satisfy this interface.
*/
type ChassisBase interface {
	// SetStatusLed sets the state of the system status LED.
	// color is one of the StatusLedColor* constants (or a vendor-specific string).
	// Returns true if the LED state was set successfully, false otherwise.
	SetStatusLed(color string) bool

	// GetStatusLed gets the current state of the system status LED.
	// Returns a color string, or StatusLedColorOff on failure.
	GetStatusLed() string

	// SetUidLed sets the state of the system UID (unit identification) LED.
	// Returns true if the LED state was set successfully, false otherwise.
	SetUidLed(color string) bool

	// GetUidLed gets the current state of the system UID LED.
	// Returns a color string, or StatusLedColorOff on failure.
	GetUidLed() string
}
