# Sidelight Investigation

## Summary
The F108 Pro has three independent light zones: the main per-key backlight, a
light bar above the arrow keys, and LED strips on each side of the case. We
investigated whether the light bar and sidelights could be controlled via HID
commands like the main backlight. They cannot. The sidelight code in the Aula
software exists for other keyboard models and is disabled for the F108 Pro via
`sidelight="0"` in `layouts/rgb-keyboard.xml`.

## Why We Thought It Would Work
Three pieces of evidence pointed to software-controllable sidelights:

### 1. Ghidra Functions Exist
`FUN_0044b800` at `0x0044b800` is a fully implemented sidelight sender. It
follows the standard protocol sequence: `04 18` begin, `04 13` with the lighting
subcommand byte, `00 80` data payload, `04 02` apply, `04 F0` finalize. A second
function, `FUN_00433fd0` at `0x00433fd0`, handles sidelight mode switching with
ten defined modes. The task dispatcher has a dedicated task type 10 for
sidelight mode changes. Language string IDs 660-670 define mode names. This is
not dead code. It is a complete, referenced feature with UI strings, a task
type, and a combo box handler (`FUN_004340d0`).

### 2. The Keyboard ACKed Every Command
When we sent sidelight commands over HID, the keyboard returned proper
acknowledgments. Byte 3 of the readback was `0x01` (ACK) for every command in
the sequence. The begin, config, data, and apply commands all succeeded from the
USB protocol's perspective.

### 3. The Manual Was Ambiguous
The manual documents FN key combos for controlling the light bar and sidelights.
But it also documents FN key combos for per-key RGB and programmable keys, both
of which *are* software-controllable. The existence of keyboard shortcuts does
not imply the absence of software control. So the manual's silence about
software sidelight control proved nothing either way.

## Why It Does Not Work

### The Feature Flag
The `layouts/rgb-keyboard.xml` file contains per-keyboard feature flags:

```xml
<menu macro="1" light_mode="1" sidelight="0" user_light="1"
      custom_light="1" music="1" screen="1" />
```

`sidelight="0"` disables the sidelight UI for this keyboard model. The flag is
parsed by `FUN_0041cd80` during initialization and stored as bit `0x100` of the
device feature flags at `this+0x7a4`. When the bit is zero, the sidelight combo
box is never shown and task type 10 is never queued from the UI.

### Generic Software Platform
The Aula software is a generic platform shared across many Epomaker/Aula
keyboard models. Each model has its own `rgb-keyboard.xml` that enables or
disables features. The sidelight code exists because other models have
software-controllable sidelights. The F108 Pro is not one of them.

### Separate Controller
The light bar and side LED strips appear to be on a separate internal controller
that the main MCU does not proxy HID commands to. The firmware accepts the
sidelight HID commands and ACKs them, but the data never reaches the LEDs.
Whether the firmware discards sidelight commands entirely or forwards them to a
controller with no LEDs connected is unknown. The effect is the same: nothing
changes.

## Sidelight Modes and FN Key Combos
See [hid-protocol.md](hid-protocol.md) for the sidelight mode table (10 modes,
`0x1f`-`0x28`) and the FN key combos for controlling the light bar and side
strips.

## Parallels to the Trailer Bug
This investigation mirrors the key remap trailer bug. In both cases:

* The keyboard ACKed every command successfully.
* No error or NAK was returned.
* The protocol sequence was correct.
* The data was silently ignored.

The difference is that the trailer bug had a fix (swap two bytes), while the
sidelight limitation is a hardware constraint. ACKs on this keyboard are not
proof that a feature is working. The firmware acknowledges at the HID layer
regardless of whether the feature controller behind it acts on the data.

## Related Documentation

* [hid-protocol.md](hid-protocol.md) - Sidelight protocol details (Sidelight
  and LED Layer sections)
* [ghidra-functions.md](ghidra-functions.md) - `FUN_0044b800` and
  `FUN_00433fd0` entries
