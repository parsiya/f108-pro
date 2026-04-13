package aula

import "fmt"

// LightingMode represents a keyboard backlight effect.
type LightingMode int

const (
	ModeOff        LightingMode = 0
	ModeStatic     LightingMode = 1
	ModeSingleOn   LightingMode = 2
	ModeSingleOff  LightingMode = 3
	ModeGlittering LightingMode = 4
	ModeFalling    LightingMode = 5
	ModeColourful  LightingMode = 6
	ModeBreath     LightingMode = 7
	ModeSpectrum   LightingMode = 8
	ModeOutward    LightingMode = 9
	ModeScrolling  LightingMode = 10
	ModeRolling    LightingMode = 11
	ModeRotating   LightingMode = 12
	ModeExplode    LightingMode = 13
	ModeLaunch     LightingMode = 14
	ModeRipples    LightingMode = 15
	ModeFlowing    LightingMode = 16
	ModePulsating  LightingMode = 17
	ModeTilt       LightingMode = 18
	ModeShuttle    LightingMode = 19
)

// ModeNames maps mode IDs to their display names.
var ModeNames = map[LightingMode]string{
	ModeOff:        "Off",
	ModeStatic:     "Static",
	ModeSingleOn:   "SingleOn",
	ModeSingleOff:  "SingleOff",
	ModeGlittering: "Glittering",
	ModeFalling:    "Falling",
	ModeColourful:  "Colourful",
	ModeBreath:     "Breath",
	ModeSpectrum:   "Spectrum",
	ModeOutward:    "Outward",
	ModeScrolling:  "Scrolling",
	ModeRolling:    "Rolling",
	ModeRotating:   "Rotating",
	ModeExplode:    "Explode",
	ModeLaunch:     "Launch",
	ModeRipples:    "Ripples",
	ModeFlowing:    "Flowing",
	ModePulsating:  "Pulsating",
	ModeTilt:       "Tilt",
	ModeShuttle:    "Shuttle",
}

// LightingConfig holds the parameters for a lighting mode change.
type LightingConfig struct {
	Mode       LightingMode
	R, G, B    uint8
	Brightness uint8 // 0-5.
	Speed      uint8 // 0-5.
	Direction  uint8 // 0 or 1.
	Colorful   bool  // true = rainbow, false = single color.
}

// SetLighting changes the keyboard backlight mode and parameters.
// Sequence: begin -> lighting init -> data -> apply -> finalize.
//
// dev.SetLighting(aula.LightingConfig{Mode: aula.ModeBreath, R: 255, Brightness: 5, Speed: 3})
func (d *Device) SetLighting(cfg LightingConfig) error {
	if cfg.Mode < 0 || cfg.Mode > 19 {
		return fmt.Errorf("invalid mode %d (valid range: 0-19)", cfg.Mode)
	}
	if cfg.Brightness > 5 {
		return fmt.Errorf("invalid brightness %d (valid range: 0-5)", cfg.Brightness)
	}
	if cfg.Speed > 5 {
		return fmt.Errorf("invalid speed %d (valid range: 0-5)", cfg.Speed)
	}

	// Step 1: Begin transaction.
	if err := d.beginTransaction(); err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	// Step 2: Lighting init (04 13, byte[8]=01).
	if err := d.lightingInit(); err != nil {
		return fmt.Errorf("lighting init: %w", err)
	}

	// Step 3: Data packet.
	payload := make([]byte, 64)
	payload[0] = byte(cfg.Mode)

	if cfg.Mode != 0 {
		payload[1] = cfg.R
		payload[2] = cfg.G
		payload[3] = cfg.B

		colorful := uint8(0)
		if cfg.Colorful {
			colorful = 1
		}
		payload[8] = colorful
		payload[9] = cfg.Brightness
		payload[10] = cfg.Speed
		payload[11] = cfg.Direction
	}

	// Trailer 0x55AA at payload offsets 14-15.
	payload[14] = 0x55
	payload[15] = 0xAA

	if err := d.sendCommand(payload, false); err != nil {
		return fmt.Errorf("data: %w", err)
	}

	// Step 4: Apply.
	if err := d.applyTransaction(); err != nil {
		return fmt.Errorf("apply: %w", err)
	}

	// Step 5: Finalize.
	if err := d.finalizeTransaction(); err != nil {
		return fmt.Errorf("finalize: %w", err)
	}

	return nil
}
