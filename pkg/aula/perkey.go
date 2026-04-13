package aula

import "fmt"

// Per-key RGB buffer sizes.
const (
	perKeyRGBSize  = 0x240 // 576 bytes: 144 slots × 4 bytes.
	perKeyMonoSize = 0xC0  // 192 bytes: 192 slots × 1 byte.
)

// Default brightness for the LED strip setup preamble.
const defaultBrightness = 5

// KeyColor assigns an RGB color to a single key identified by its light index.
type KeyColor struct {
	LightIndex uint8
	R, G, B    uint8
}

// Key light indices from rgb-keyboard.xml.
// On the F108 Pro, light_index == key_index.
const (
	KeyEsc         = 1
	KeyF1          = 2
	KeyF2          = 3
	KeyF3          = 4
	KeyF4          = 5
	KeyF5          = 6
	KeyF6          = 7
	KeyF7          = 8
	KeyF8          = 9
	KeyF9          = 10
	KeyF10         = 11
	KeyF11         = 12
	KeyF12         = 13
	KeyPrintScreen = 112
	KeyScrollLock  = 113
	KeyPause       = 115

	KeyGrave     = 19
	Key1         = 20
	Key2         = 21
	Key3         = 22
	Key4         = 23
	Key5         = 24
	Key6         = 25
	Key7         = 26
	Key8         = 27
	Key9         = 28
	Key0         = 29
	KeyMinus     = 30
	KeyEqual     = 31
	KeyBackspace = 103
	KeyInsert    = 116
	KeyHome      = 117
	KeyPageUp    = 118
	KeyNumLock   = 32
	KeyNumSlash  = 33
	KeyNumStar   = 34
	KeyNumMinus  = 122

	KeyTab       = 37
	KeyQ         = 38
	KeyW         = 39
	KeyE         = 40
	KeyR         = 41
	KeyT         = 42
	KeyY         = 43
	KeyU         = 44
	KeyI         = 45
	KeyO         = 46
	KeyP         = 47
	KeyLBracket  = 48
	KeyRBracket  = 49
	KeyBackslash = 67
	KeyDelete    = 119
	KeyEnd       = 120
	KeyPageDown  = 121
	KeyNum7      = 50
	KeyNum8      = 51
	KeyNum9      = 52
	KeyNumPlus   = 123

	KeyCapsLock = 55
	KeyA        = 56
	KeyS        = 57
	KeyD        = 58
	KeyF        = 59
	KeyG        = 60
	KeyH        = 61
	KeyJ        = 62
	KeyK        = 63
	KeyL        = 64
	KeySemicolon = 65
	KeyQuote    = 66
	KeyEnter    = 85
	KeyNum4     = 68
	KeyNum5     = 69
	KeyNum6     = 70

	KeyLShift   = 73
	KeyZ        = 74
	KeyX        = 75
	KeyC        = 76
	KeyV        = 77
	KeyB        = 78
	KeyN        = 79
	KeyM        = 80
	KeyComma    = 81
	KeyDot      = 82
	KeySlash    = 83
	KeyRShift   = 84
	KeyUp       = 101
	KeyNum1     = 86
	KeyNum2     = 87
	KeyNum3     = 88
	KeyNumEnter = 106

	KeyLCtrl  = 91
	KeyLWin   = 92
	KeyLAlt   = 93
	KeySpace  = 94
	KeyRAlt   = 95
	KeyFn     = 96
	KeyMenu   = 97
	KeyRCtrl  = 98
	KeyLeft   = 99
	KeyDown   = 100
	KeyRight  = 102
	KeyNum0   = 104
	KeyNumDot = 105
)

// KeyNameToIndex maps human-readable key names to light indices.
var KeyNameToIndex = map[string]uint8{
	"esc": KeyEsc, "f1": KeyF1, "f2": KeyF2, "f3": KeyF3,
	"f4": KeyF4, "f5": KeyF5, "f6": KeyF6, "f7": KeyF7,
	"f8": KeyF8, "f9": KeyF9, "f10": KeyF10, "f11": KeyF11,
	"f12": KeyF12, "printscreen": KeyPrintScreen, "scrolllock": KeyScrollLock,
	"pause": KeyPause,

	"grave": KeyGrave, "1": Key1, "2": Key2, "3": Key3,
	"4": Key4, "5": Key5, "6": Key6, "7": Key7,
	"8": Key8, "9": Key9, "0": Key0, "minus": KeyMinus,
	"equal": KeyEqual, "backspace": KeyBackspace,
	"insert": KeyInsert, "home": KeyHome, "pageup": KeyPageUp,
	"numlock": KeyNumLock, "numslash": KeyNumSlash,
	"numstar": KeyNumStar, "numminus": KeyNumMinus,

	"tab": KeyTab, "q": KeyQ, "w": KeyW, "e": KeyE,
	"r": KeyR, "t": KeyT, "y": KeyY, "u": KeyU,
	"i": KeyI, "o": KeyO, "p": KeyP,
	"lbracket": KeyLBracket, "rbracket": KeyRBracket,
	"backslash": KeyBackslash,
	"delete": KeyDelete, "end": KeyEnd, "pagedown": KeyPageDown,
	"num7": KeyNum7, "num8": KeyNum8, "num9": KeyNum9, "numplus": KeyNumPlus,

	"capslock": KeyCapsLock, "a": KeyA, "s": KeyS, "d": KeyD,
	"f": KeyF, "g": KeyG, "h": KeyH, "j": KeyJ,
	"k": KeyK, "l": KeyL, "semicolon": KeySemicolon, "quote": KeyQuote,
	"enter": KeyEnter,
	"num4": KeyNum4, "num5": KeyNum5, "num6": KeyNum6,

	"lshift": KeyLShift, "z": KeyZ, "x": KeyX, "c": KeyC,
	"v": KeyV, "b": KeyB, "n": KeyN, "m": KeyM,
	"comma": KeyComma, "dot": KeyDot, "slash": KeySlash,
	"rshift": KeyRShift, "up": KeyUp,
	"num1": KeyNum1, "num2": KeyNum2, "num3": KeyNum3, "numenter": KeyNumEnter,

	"lctrl": KeyLCtrl, "lwin": KeyLWin, "lalt": KeyLAlt,
	"space": KeySpace, "ralt": KeyRAlt, "fn": KeyFn,
	"menu": KeyMenu, "rctrl": KeyRCtrl,
	"left": KeyLeft, "down": KeyDown, "right": KeyRight,
	"num0": KeyNum0, "numdot": KeyNumDot,
}

// SetPerKeyRGB sets individual key colors using the per-key RGB protocol.
// Keys not in the list will be dark (off). The full sequence is:
//
//  1. LED strip setup preamble (begin -> lighting init -> brightness data -> apply -> finalize).
//  2. Per-key RGB data (begin -> 04 23 init -> 576-byte buffer -> apply -> finalize).
//
// Brightness controls the overall LED brightness (0-5, default 5).
//
// dev.SetPerKeyRGB([]aula.KeyColor{{LightIndex: aula.KeyEsc, R: 255}}, 5)
func (d *Device) SetPerKeyRGB(keys []KeyColor, brightness uint8) error {
	if brightness > 5 {
		brightness = 5
	}

	// Step 1: LED strip setup preamble.
	if err := d.ledStripSetup(brightness); err != nil {
		return fmt.Errorf("LED strip setup: %w", err)
	}

	// Step 2: Begin transaction.
	if err := d.beginTransaction(); err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	// Step 3: Per-key RGB init (04 23, byte[8]=09 for RGB mode).
	initCmd := make([]byte, 64)
	initCmd[0] = 0x04
	initCmd[1] = 0x23
	initCmd[8] = 0x09
	if err := d.sendCommand(initCmd, true); err != nil {
		return fmt.Errorf("perkey init: %w", err)
	}

	// Step 4: Build and send RGB data buffer (576 bytes).
	buf := make([]byte, perKeyRGBSize)
	for _, kc := range keys {
		idx := int(kc.LightIndex)
		if idx <= 0 || idx*4+3 >= perKeyRGBSize-2 {
			continue // Skip invalid indices (0 is unused, avoid trailer area).
		}
		off := idx * 4
		buf[off] = kc.LightIndex
		buf[off+1] = kc.R
		buf[off+2] = kc.G
		buf[off+3] = kc.B
	}
	// Trailer 0x55AA at the last two bytes.
	buf[perKeyRGBSize-2] = 0x55
	buf[perKeyRGBSize-1] = 0xAA

	if err := d.sendMultiPacket(buf, true); err != nil {
		return fmt.Errorf("perkey data: %w", err)
	}

	// Step 5: Apply.
	if err := d.applyTransaction(); err != nil {
		return fmt.Errorf("apply: %w", err)
	}

	// Step 6: Finalize (with readback, per decompiled code).
	if err := d.sendCommand([]byte{0x04, 0xF0}, true); err != nil {
		return fmt.Errorf("finalize: %w", err)
	}

	return nil
}

// ledStripSetup sends the LED strip preamble required before per-key RGB data.
// Sequence: begin -> lighting init -> brightness data -> apply -> finalize.
//
// d.ledStripSetup(5)
func (d *Device) ledStripSetup(brightness uint8) error {
	// Begin.
	if err := d.beginTransaction(); err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	// Lighting init (04 13, byte[8]=01).
	if err := d.lightingInit(); err != nil {
		return fmt.Errorf("lighting init: %w", err)
	}

	// Data packet: byte[0]=0x80, byte[9]=brightness, byte[14-15]=0x55AA.
	data := make([]byte, 64)
	data[0] = 0x80
	data[9] = brightness
	data[14] = 0x55
	data[15] = 0xAA
	if err := d.sendCommand(data, false); err != nil {
		return fmt.Errorf("brightness data: %w", err)
	}

	// Apply.
	if err := d.applyTransaction(); err != nil {
		return fmt.Errorf("apply: %w", err)
	}

	// Finalize (no readback for this sub-sequence).
	if err := d.finalizeTransaction(); err != nil {
		return fmt.Errorf("finalize: %w", err)
	}

	return nil
}
