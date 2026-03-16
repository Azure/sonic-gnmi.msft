/*
	Package mellanox implements Mellanox-specific platform APIs for SONiC.
	It provides the concrete LED control logic that satisfies the
	platform.ChassisBase interface.

	Ported from:
	sonic-buildimage/platform/mellanox/mlnx-platform-api/sonic_platform/led.py
*/
package mellanox

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	log "github.com/golang/glog"
	"github.com/sonic-net/sonic-gnmi/show_client/common"
)

// LED sysfs base path on Mellanox platforms.
const LedPath = "/var/run/hw-management/led/"

// LED brightness values written to sysfs files.
const (
	LedOn    = "255"
	LedOff   = "0"
	LedBlink = "50"
)

// LED color constants (extended beyond device_base to include blink variants).
const (
	StatusLedColorGreen       = "green"
	StatusLedColorRed         = "red"
	StatusLedColorOrange      = "orange"
	StatusLedColorBlue        = "blue"
	StatusLedColorOff         = "off"
	StatusLedColorGreenBlink  = "green_blink"
	StatusLedColorRedBlink    = "red_blink"
	StatusLedColorOrangeBlink = "orange_blink"
	StatusLedColorBlueBlink   = "blue_blink"
)

// SimilarColors maps a color to fallback colors when the primary is not
// supported by the hardware.
var SimilarColors = map[string][]string{
	"red":    {"amber", "orange"},
	"amber":  {"red", "orange"},
	"orange": {"red", "amber"},
}

// PrimaryColors maps actual hardware colors back to canonical "primary"
// colors for backward compatibility.
var PrimaryColors = map[string]string{
	"red":    "red",
	"amber":  "red",
	"orange": "red",
	"green":  "green",
	"blue":   "blue",
}

// Led represents a single hardware LED on a Mellanox platform.
// It reads capabilities from sysfs and sets brightness/blink values.
type Led struct {
	ledID           string
	supportedColors map[string]struct{}
	supportedBlinks map[string]struct{}
}

func NewLed(ledID string) *Led {
	/* NewLed creates a new Led with the given ID.
		The ledID determines the sysfs file names (e.g., "status" → led_status_*).*/
	return &Led{
		ledID:           ledID,
		supportedColors: make(map[string]struct{}),
		supportedBlinks: make(map[string]struct{}),
	}
}

func SystemLed() *Led {
	/* SystemLed creates a Led for the system status LED (ledID = "status"). */
	return &Led{
		ledID:           "status",
		supportedColors: make(map[string]struct{}),
		supportedBlinks: make(map[string]struct{}),
	}
}

func SystemUidLed() *Led {
	/* SystemUidLed creates a Led for the system UID LED (ledID = "uid"). */
	return &Led{
		ledID:           "uid",
		supportedColors: make(map[string]struct{}),
		supportedBlinks: make(map[string]struct{}),
	}
}

// --- Path helpers ---

func (led *Led) getLedPath(color string) string {
	/* getLedPath returns the sysfs path for the given color brightness file.
		e.g., /var/run/hw-management/led/led_status_green*/
	return filepath.Join(LedPath, fmt.Sprintf("led_%s_%s", led.ledID, color))
}

func (led *Led) getLedTrigger(color string) string {
	return filepath.Join(LedPath, fmt.Sprintf("led_%s_%s_trigger", led.ledID, color))
}

func (led *Led) getLedCapPath() string {
	return filepath.Join(LedPath, fmt.Sprintf("led_%s_capability", led.ledID))
}

func (led *Led) getLedDelayOnPath(color string) string {
	return filepath.Join(LedPath, fmt.Sprintf("led_%s_%s_delay_on", led.ledID, color))
}

func (led *Led) getLedDelayOffPath(color string) string {
	return filepath.Join(LedPath, fmt.Sprintf("led_%s_%s_delay_off", led.ledID, color))
}

// --- Capability parsing ---

func (led *Led) GetCapability() {
	/* GetCapability reads the LED capability file and populates supportedColors
		and supportedBlinks.*/
	caps := common.ReadStringFromFile(led.getLedCapPath(), "")
	for _, capability := range strings.Fields(caps) {
		if capability == "none" {
			continue
		}
		pos := strings.Index(capability, "_blink")
		if pos != -1 {
			led.supportedBlinks[capability[:pos]] = struct{}{}
		} else {
			led.supportedColors[capability] = struct{}{}
		}
	}
}

// --- Color resolution ---

func (led *Led) getActualColor(color string) string {
	/* getActualColor returns the given color if supported, otherwise falls back
		to a similar color. Returns empty string if no suitable color is found.*/
	if _, ok := led.supportedColors[color]; ok {
		return color
	}
	return led.getSimilarColor(color)
}

func (led *Led) getSimilarColor(color string) string {
	/* getSimilarColor tries to find a supported color similar to the given one,
		using the SimilarColors fallback map.*/
	similar, ok := SimilarColors[color]
	if ok {
		for _, candidate := range similar {
			if _, supported := led.supportedColors[candidate]; supported {
				return candidate
			}
		}
	}
	return ""
}

func (led *Led) getPrimaryColor(color string) string {
	/* getPrimaryColor maps an actual hardware color back to its canonical
		"primary" color for backward compatibility.*/
	if primary, ok := PrimaryColors[color]; ok {
		return primary
	}
	return color
}

// --- Blink color resolution ---

func (led *Led) getActualBlinkColor(blinkColor string) string {
	/* getActualBlinkColor returns the given blink color if supported, otherwise
		falls back to a similar blink color.*/
	if _, ok := led.supportedBlinks[blinkColor]; ok {
		return blinkColor
	}
	return led.getSimilarBlinkColor(blinkColor)
}

func (led *Led) getSimilarBlinkColor(color string) string {
	/* getSimilarBlinkColor tries to find a supported blink color similar to the given one.*/
	similar, ok := SimilarColors[color]
	if ok {
		for _, candidate := range similar {
			if _, supported := led.supportedBlinks[candidate]; supported {
				return candidate
			}
		}
	}
	return ""
}

// --- Blink control ---

func (led *Led) triggerBlink(blinkTriggerFile string) {
	/* triggerBlink activates the blink timer on the given trigger file.*/
	common.WriteFile(blinkTriggerFile, "timer")
}

func (led *Led) untriggerBlink(blinkTriggerFile string) {
	/* untriggerBlink deactivates blinking on the given trigger file.*/
	common.WriteFile(blinkTriggerFile, "none")
}

func (led *Led) stopBlink() {
	/* stopBlink stops blinking for all supported colors.*/
	for color := range led.supportedColors {
		led.untriggerBlink(led.getLedTrigger(color))
	}
}

func (led *Led) setStatusBlink(color string) bool {
	/* setStatusBlink sets the LED to a blinking state for the given base color.
		Returns true on success.*/
	actualColor := led.getActualBlinkColor(color)
	if actualColor == "" {
		log.Errorf("Set LED to color %s_blink is not supported", color)
		return false
	}

	led.triggerBlink(led.getLedTrigger(actualColor))
	return led.setLedBlinkStatus(actualColor)
}

func (led *Led) setLedBlinkStatus(actualColor string) bool {
	/* setLedBlinkStatus writes the blink delay values after the trigger file
		creates the delay_on/delay_off sysfs entries.*/
	delayOnFile := led.getLedDelayOnPath(actualColor)
	delayOffFile := led.getLedDelayOffPath(actualColor)

	if !led.waitFilesReady(delayOnFile, delayOffFile) {
		return false
	}

	common.WriteFile(delayOnFile, LedBlink)
	common.WriteFile(delayOffFile, LedBlink)
	return true
}

func (led *Led) waitFilesReady(files ...string) bool {
	/* waitFilesReady waits up to 5 seconds for all given files to exist,
		using exponential backoff starting at 10ms.*/
	waitTime := 5.0
	sleepDuration := 0.01

	for waitTime > 0 {
		allExist := true
		for _, f := range files {
			if !common.FileExists(f) {
				allExist = false
				break
			}
		}
		if allExist {
			return true
		}
		time.Sleep(time.Duration(sleepDuration * float64(time.Second)))
		waitTime -= sleepDuration
		sleepDuration *= 2
	}
	return false
}

// --- Blink status reading ---

func (led *Led) getBlinkStatus() string {
	/* getBlinkStatus checks if any supported color is currently blinking.
		Returns the blink color string (e.g., "green_blink") or empty string.*/
	for color := range led.supportedColors {
		if led.isLedBlinking(color) {
			return color + "_blink"
		}
	}
	return ""
}

func (led *Led) isLedBlinking(color string) bool {
	/* isLedBlinking returns true if the given color's delay_on and delay_off
		values are both non-zero (i.e., blinking is active).*/
	delayOn := common.ReadStringFromFile(led.getLedDelayOnPath(color), LedOff)
	delayOff := common.ReadStringFromFile(led.getLedDelayOffPath(color), LedOff)
	return delayOn != LedOff && delayOff != LedOff
}

// --- Core set/get ---

func (led *Led) SetStatus(color string) bool {
	/* SetStatus sets the LED to the given color.
		Handles solid colors, blink colors (e.g., "green_blink"), and "off".
		Returns true on success, false on failure.*/
	led.GetCapability()

	if len(led.supportedColors) == 0 {
		if !common.IsSimxPlatform() {
			log.Errorf("Failed to get LED capability for %s LED", led.ledID)
		}
		return false
	}

	status := false

	led.stopBlink()

	// Check if this is a blink request
	blinkPos := strings.Index(color, "_blink")
	if blinkPos != -1 {
		return led.setStatusBlink(color[:blinkPos])
	}

	if color != StatusLedColorOff {
		actualColor := led.getActualColor(color)
		if actualColor == "" {
			log.Errorf("Set LED to color %s is not supported", color)
			return false
		}
		if common.WriteFile(led.getLedPath(actualColor), LedOn) {
			status = true
		}
	} else {
		// Turn off: set all supported colors to LED_OFF
		for c := range led.supportedColors {
			common.WriteFile(led.getLedPath(c), LedOff)
		}
		status = true
	}

	return status
}

func (led *Led) GetStatus() string {
	/* GetStatus reads the current LED state from sysfs.
		Returns the current color string (e.g., "green", "red_blink", "off").*/
	led.GetCapability()

	if len(led.supportedColors) == 0 {
		if !common.IsSimxPlatform() {
			log.Errorf("Failed to get LED capability for %s LED", led.ledID)
		}
		return StatusLedColorOff
	}

	// Check for blink status first
	blinkStatus := led.getBlinkStatus()
	if blinkStatus != "" {
		return blinkStatus
	}

	// Check which color is currently on
	for color := range led.supportedColors {
		val := common.ReadStringFromFile(led.getLedPath(color), LedOff)
		if val != LedOff {
			return led.getPrimaryColor(color)
		}
	}

	return StatusLedColorOff
}
