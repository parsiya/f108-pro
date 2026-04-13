package aula

import "fmt"

// Remap buffer size: 576 bytes = 144 slots × 4 bytes, last 2 bytes are 0x55AA trailer.
const remapBufSize = 0x240

// RemapAction is the action type byte in a key remap slot.
type RemapAction byte

const (
	RemapNone     RemapAction = 0x00 // No remap (passthrough).
	RemapSpecial  RemapAction = 0x01 // Special function (lock keys, etc.).
	RemapKey      RemapAction = 0x02 // Key combination (modifier + key).
	RemapConsumer RemapAction = 0x03 // Consumer control (multimedia).
	RemapProfile  RemapAction = 0x05 // Profile/lock function.
	RemapMacro    RemapAction = 0x06 // Macro execution.
	RemapMouse    RemapAction = 0x07 // Mouse function.
)

// KeyRemap defines a single key remapping.
// SourceIndex is the key_index of the physical key to remap.
// The 4-byte slot format is [Action, Param1, Param2, Param3].
type KeyRemap struct {
	SourceIndex uint8 // key_index of the physical key.
	Action      RemapAction
	Param1      byte
	Param2      byte
	Param3      byte
}

// KeyNameToHID maps human-readable key names to USB HID usage codes (Usage Page 0x07).
var KeyNameToHID = map[string]byte{
	"esc": 0x29, "f1": 0x3A, "f2": 0x3B, "f3": 0x3C,
	"f4": 0x3D, "f5": 0x3E, "f6": 0x3F, "f7": 0x40,
	"f8": 0x41, "f9": 0x42, "f10": 0x43, "f11": 0x44,
	"f12": 0x45, "printscreen": 0x46, "scrolllock": 0x47,
	"pause": 0x48,

	"grave": 0x35, "1": 0x1E, "2": 0x1F, "3": 0x20,
	"4": 0x21, "5": 0x22, "6": 0x23, "7": 0x24,
	"8": 0x25, "9": 0x26, "0": 0x27, "minus": 0x2D,
	"equal": 0x2E, "backspace": 0x2A,
	"insert": 0x49, "home": 0x4A, "pageup": 0x4B,
	"numlock": 0x53, "numslash": 0x54,
	"numstar": 0x55, "numminus": 0x56,

	"tab": 0x2B, "q": 0x14, "w": 0x1A, "e": 0x08,
	"r": 0x15, "t": 0x17, "y": 0x1C, "u": 0x18,
	"i": 0x0C, "o": 0x12, "p": 0x13,
	"lbracket": 0x2F, "rbracket": 0x30,
	"backslash": 0x31,
	"delete": 0x4C, "end": 0x4D, "pagedown": 0x4E,
	"num7": 0x5F, "num8": 0x60, "num9": 0x61, "numplus": 0x57,

	"capslock": 0x39, "a": 0x04, "s": 0x16, "d": 0x07,
	"f": 0x09, "g": 0x0A, "h": 0x0B, "j": 0x0D,
	"k": 0x0E, "l": 0x0F, "semicolon": 0x33, "quote": 0x34,
	"enter": 0x28,
	"num4": 0x5C, "num5": 0x5D, "num6": 0x5E,

	"lshift": 0xE1, "z": 0x1D, "x": 0x1B, "c": 0x06,
	"v": 0x19, "b": 0x05, "n": 0x11, "m": 0x10,
	"comma": 0x36, "dot": 0x37, "slash": 0x38,
	"rshift": 0xE5, "up": 0x52,
	"num1": 0x59, "num2": 0x5A, "num3": 0x5B, "numenter": 0x58,

	"lctrl": 0xE0, "lwin": 0xE3, "lalt": 0xE2,
	"space": 0x2C, "ralt": 0xE6, "fn": 0xAF,
	"menu": 0x65, "rctrl": 0xE4,
	"left": 0x50, "down": 0x51, "right": 0x4F,
	"num0": 0x62, "numdot": 0x63,
}

// hidToModifierBit converts HID modifier key codes (0xE0-0xE7) to bit flags.
// Non-modifier codes are returned unchanged.
//
// hidToModifierBit(0xE0) // 0x01 (Left Ctrl).
// hidToModifierBit(0xE1) // 0x02 (Left Shift).
// hidToModifierBit(0x04) // 0x04 (not a modifier, returned as-is).
func hidToModifierBit(code byte) byte {
	switch code {
	case 0xE0:
		return 0x01 // Left Ctrl.
	case 0xE1:
		return 0x02 // Left Shift.
	case 0xE2:
		return 0x04 // Left Alt.
	case 0xE3:
		return 0x08 // Left GUI/Win.
	case 0xE4:
		return 0x10 // Right Ctrl.
	case 0xE5:
		return 0x20 // Right Shift.
	case 0xE6:
		return 0x40 // Right Alt.
	case 0xE7:
		return 0x80 // Right GUI/Win.
	default:
		return code
	}
}

// isModifierKey returns true if the HID code is a modifier key (0xE0-0xE7).
//
// isModifierKey(0xE0) // true.
// isModifierKey(0x04) // false.
func isModifierKey(code byte) bool {
	return code >= 0xE0 && code <= 0xE7
}

// ConsumerNameToCode maps human-readable multimedia key names to USB HID
// Consumer Page (0x0C) usage codes.
var ConsumerNameToCode = map[string]byte{
	"play":    0xCD, // Play/Pause.
	"stop":    0xB7, // Stop.
	"prev":    0xB6, // Scan Previous Track.
	"next":    0xB5, // Scan Next Track.
	"volup":   0xE9, // Volume Up.
	"voldown": 0xEA, // Volume Down.
	"mute":    0xE2, // Mute.
}

// MouseNameToParams maps human-readable mouse action names to param bytes.
// Each entry is [param1, param2, param3].
var MouseNameToParams = map[string][3]byte{
	"lclick":   {0x01, 0x00, 0x00}, // Left click.
	"rclick":   {0x02, 0x00, 0x00}, // Right click.
	"mclick":   {0x04, 0x00, 0x00}, // Middle click.
	"scrollup": {0x00, 0x01, 0x00}, // Scroll up.
	"scrolldn": {0x00, 0xFF, 0x00}, // Scroll down.
}

// NewKeySwap creates a KeyRemap that swaps a physical key to produce a
// different key. Use key_index for source and HID code for target.
//
// NewKeySwap(KeyEsc, 0x04) // Remap Esc to produce 'A'.
func NewKeySwap(sourceIndex uint8, targetHID byte) KeyRemap {
	kr := KeyRemap{
		SourceIndex: sourceIndex,
		Action:      RemapKey,
	}
	if isModifierKey(targetHID) {
		// Modifier keys: put modifier bit in Param1, zero in Param2.
		kr.Param1 = hidToModifierBit(targetHID)
		kr.Param2 = 0
	} else {
		// Normal keys: no modifier in Param1, HID code in Param2.
		kr.Param1 = 0
		kr.Param2 = targetHID
	}
	return kr
}

// NewKeyCombo creates a KeyRemap that produces a key with modifiers held.
// modifierBitmask uses standard USB HID modifier bits (0x01=LCtrl, 0x02=LShift,
// 0x04=LAlt, 0x08=LWin, 0x10=RCtrl, 0x20=RShift, 0x40=RAlt, 0x80=RWin).
//
// NewKeyCombo(55, 0x01, 0x06) // CapsLock -> Ctrl+C.
// NewKeyCombo(55, 0x08, 0x07) // CapsLock -> Win+D.
func NewKeyCombo(sourceIndex uint8, modifierBitmask byte, targetHID byte) KeyRemap {
	return KeyRemap{
		SourceIndex: sourceIndex,
		Action:      RemapKey,
		Param1:      modifierBitmask,
		Param2:      targetHID,
	}
}

// NewConsumerRemap creates a KeyRemap that produces a multimedia/consumer
// control action (USB HID Consumer Page 0x0C).
//
// NewConsumerRemap(55, 0xCD) // CapsLock -> Play/Pause.
// NewConsumerRemap(2, 0xE9)  // F1 -> Volume Up.
func NewConsumerRemap(sourceIndex uint8, consumerCode byte) KeyRemap {
	return KeyRemap{
		SourceIndex: sourceIndex,
		Action:      RemapConsumer,
		Param1:      consumerCode,
	}
}

// NewMouseRemap creates a KeyRemap that produces a mouse action.
//
// NewMouseRemap(55, 0x01, 0x00, 0x00) // CapsLock -> Left click.
// NewMouseRemap(55, 0x00, 0x01, 0x00) // CapsLock -> Scroll up.
func NewMouseRemap(sourceIndex uint8, p1, p2, p3 byte) KeyRemap {
	return KeyRemap{
		SourceIndex: sourceIndex,
		Action:      RemapMouse,
		Param1:      p1,
		Param2:      p2,
		Param3:      p3,
	}
}

// SetKeyRemap sends key remappings to the keyboard's normal layer.
// Keys not in the remaps list keep their default behavior (passthrough).
// The entire remap table is sent each time (576 bytes, 9 packets).
//
// Sequence: begin -> 04 11 init -> remap data (576 bytes) -> apply -> finalize.
//
// dev.SetKeyRemap([]KeyRemap{NewKeySwap(KeyEsc, 0x39)})
func (d *Device) SetKeyRemap(remaps []KeyRemap) error {
	return d.sendRemapTable(remaps, false)
}

// SetFnKeyRemap sends key remappings to the keyboard's FN layer.
// Same format as SetKeyRemap but targets the FN layer (command 04 27).
//
// dev.SetFnKeyRemap([]KeyRemap{NewKeySwap(KeyF1, 0xCD)})
func (d *Device) SetFnKeyRemap(remaps []KeyRemap) error {
	return d.sendRemapTable(remaps, true)
}

// ResetKeyRemap clears all key remappings on the normal layer.
//
// dev.ResetKeyRemap()
func (d *Device) ResetKeyRemap() error {
	return d.sendRemapTable(nil, false)
}

// ResetFnKeyRemap clears all key remappings on the FN layer.
//
// dev.ResetFnKeyRemap()
func (d *Device) ResetFnKeyRemap() error {
	return d.sendRemapTable(nil, true)
}

// sendRemapTable sends the remap table for either the normal or FN layer.
func (d *Device) sendRemapTable(remaps []KeyRemap, fnLayer bool) error {
	// Step 1: Begin transaction.
	if err := d.beginTransaction(); err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	// Step 2: Remap init command.
	// Normal layer: 04 11, byte[8]=0x09.
	// FN layer:     04 27, byte[8]=0x09.
	initCmd := make([]byte, 64)
	initCmd[0] = 0x04
	if fnLayer {
		initCmd[1] = 0x27
	} else {
		initCmd[1] = 0x11
	}
	initCmd[8] = 0x09
	if err := d.sendCommand(initCmd, true); err != nil {
		return fmt.Errorf("remap init: %w", err)
	}

	// Step 3: Build and send remap data buffer (576 bytes).
	buf := make([]byte, remapBufSize)
	for _, r := range remaps {
		idx := int(r.SourceIndex)
		if idx <= 0 || idx*4+3 >= remapBufSize-2 {
			continue // Skip invalid indices (0 is unused, avoid trailer area).
		}
		off := idx * 4
		buf[off] = byte(r.Action)
		buf[off+1] = r.Param1
		buf[off+2] = r.Param2
		buf[off+3] = r.Param3
	}
	// Trailer 0x55AA (little-endian: bytes AA 55 on the wire).
	buf[remapBufSize-2] = 0xAA
	buf[remapBufSize-1] = 0x55

	if err := d.sendMultiPacket(buf, false); err != nil {
		return fmt.Errorf("remap data: %w", err)
	}

	// Step 4: Apply.
	if err := d.applyTransaction(); err != nil {
		return fmt.Errorf("apply: %w", err)
	}

	// Step 5: Finalize.
	if err := d.sendCommand([]byte{0x04, 0xF0}, true); err != nil {
		return fmt.Errorf("finalize: %w", err)
	}

	return nil
}
