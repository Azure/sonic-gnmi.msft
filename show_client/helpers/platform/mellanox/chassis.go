/*
	Mellanox 
	Package mellanox implements Mellanox-specific platform APIs for SONiC.

	Module contains an implementation of SONiC Platform Base API and
	provides the Chassis information which are available in the platform

	Ported from:
	sonic-buildimage/platform/mellanox/mlnx-platform-api/sonic_platform/led.py
*/
package mellanox

// It manages the system status LED and system UID LED via lazy initialization.
type Chassis struct {
	led    *Led
	ledUid *Led
}

func (c *Chassis) initializeSystemLed() {
	/* initializeSystemLed performs lazy initialization of the system LED objects.*/
	if c.led == nil {
		c.led = SystemLed()
		c.ledUid = SystemUidLed()
	}
}

func (c *Chassis) SetStatusLed(color string) bool {
	/* SetStatusLed sets the state of the system status LED.
		color is a string such as "green", "red", "off", "green_blink", etc.
		Returns true if the LED state was set successfully, false otherwise.*/
	c.initializeSystemLed()
	if c.led == nil {
		return false
	}
	return c.led.SetStatus(color)
}

func (c *Chassis) GetStatusLed() string {
	/* GetStatusLed gets the current state of the system status LED.
		Returns a color string (e.g., "green", "red", "off").*/
	c.initializeSystemLed()
	if c.led == nil {
		return StatusLedColorOff
	}
	return c.led.GetStatus()
}

func (c *Chassis) SetUidLed(color string) bool {
	/* SetUidLed sets the state of the system UID (unit identification) LED.
		Returns true if the LED state was set successfully, false otherwise.*/
	c.initializeSystemLed()
	if c.ledUid == nil {
		return false
	}
	return c.ledUid.SetStatus(color)
}

func (c *Chassis) GetUidLed() string {
	/* GetUidLed gets the current state of the system UID LED.
		Returns a color string (e.g., "green", "off").*/
	c.initializeSystemLed()
	if c.ledUid == nil {
		return StatusLedColorOff
	}
	return c.ledUid.GetStatus()
}
