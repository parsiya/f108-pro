# Key Remap Protocol

## Status
Verified working on hardware (Windows, via `aula.exe remap`). The initial
implementation failed silently because the trailer bytes `0x55AA` were written
in the wrong byte order (big-endian instead of little-endian).

## Overview
The Aula F108 Pro supports remapping individual keys via HID feature reports.
There are two independent remap layers:

* Normal layer: active during regular typing (command `04 11`).
* FN layer: active while the FN key is held (command `04 27`).

Each layer sends a complete 576-byte (0x240) remap table covering all key
slots. The table is split into nine 64-byte feature reports for transmission.

## Source Functions (Ghidra)

| Function     | Address      | Purpose                            |
| ------------ | ------------ | ---------------------------------- |
| FUN_004185e0 | `0x004185e0` | Remap sender (normal or FN layer)  |
| FUN_00434840 | `0x00434840` | Task 6: send both layers           |
| FUN_004348a0 | `0x004348a0` | Task 7: send single layer          |
| FUN_00451b90 | `0x00451b90` | HID modifier code -> bit converter |
| FUN_0041cd80 | `0x0041cd80` | Builds key_index map from XML      |

## Protocol Sequence

| Step | Packet                | Readback | Purpose           |
| ---- | --------------------- | -------- | ----------------- |
| 1    | `04 18`               | Yes      | Begin transaction |
| 2    | `04 11` or `04 27`    | Yes      | Remap init        |
| 3    | 576 bytes (9 packets) | Yes      | Remap data        |
| 4    | `04 02`               | Yes      | Apply             |
| 5    | `04 F0`               | Yes      | Finalize          |

### Step 2: Init Command

```
Byte 0: 0x04
Byte 1: 0x11 (normal layer) or 0x27 (FN layer)
Byte 8: 0x09
Bytes 2-7, 9-63: 0x00
```

The `param_1` argument to FUN_004185e0 selects the layer:

* `param_1 = 0` -> command byte `0x11` (normal layer).
* `param_1 = 1` -> command byte `0x27` (FN layer).

## Remap Data Buffer (576 bytes)

### Layout
The buffer is 576 bytes (0x240) = 144 four-byte slots + 2-byte trailer.

```
Offset 0x000: Slot 0 (unused, always zero)
Offset 0x004: Slot 1 (key_index 1 = Esc)
Offset 0x008: Slot 2 (key_index 2 = F1)
...
Offset 0x1EC: Slot 123 (key_index 123 = Num+)
...
Offset 0x23C: Padding
Offset 0x23E: Trailer 0xAA 0x55 (uint16 0x55AA, little-endian)
```

### Slot Index
Each slot corresponds to a `key_index` from `rgb-keyboard.xml`. The mapping is
built at runtime by FUN_0041cd80 which parses the `<key>` elements and creates
a std::map from HID usage code -> `key_index`.

Slot position = `key_index * 4` bytes into the buffer.

Slot 0 is unused. Valid key indices range from 1 to 123 (with gaps).
See [key-map.md](key-map.md) for the full key_index table.

### Slot Format (4 bytes)
```
Byte 0: Action type
Byte 1: Param1
Byte 2: Param2
Byte 3: Param3
```

A slot of `00 00 00 00` means no remap (key keeps its default behavior).

## Action Types

### Type 0x00: No Remap
All bytes zero. Key produces its default output.

### Type 0x01: Special Function
Used for lock/system keys.

| Param1 | Param2 | Meaning              |
| ------ | ------ | -------------------- |
| 0x01   | 0x01   | Num Lock             |
| 0x01   | 0x04   | Scroll Lock          |
| 0x01   | 0x02   | Caps Lock            |
| 0x01   | 0x03   | Insert toggle        |
| 0x03   | 0x01   | Calculator           |
| 0x03   | 0xFF   | Lock PC              |
| 0x01   | 0x08   | App switch (Alt+Tab) |
| 0x01   | 0x10   | System function 8    |

The original software's "special function" remap type (`iVar5 == 5`) maps
sub-IDs 1-8 to these values.

### Type 0x02: Key Combination
Remaps to a keyboard key, optionally with modifiers.

```
Byte 0: 0x02
Byte 1: Modifier bitmask (0 = none)
Byte 2: HID usage code of the target key
Byte 3: 0x00 (unused)
```

Modifier bitmask (standard USB HID):

| Bit | Hex    | Modifier    |
| --- | ------ | ----------- |
| 0   | `0x01` | Left Ctrl   |
| 1   | `0x02` | Left Shift  |
| 2   | `0x04` | Left Alt    |
| 3   | `0x08` | Left Win    |
| 4   | `0x10` | Right Ctrl  |
| 5   | `0x20` | Right Shift |
| 6   | `0x40` | Right Alt   |
| 7   | `0x80` | Right Win   |

To remap to a modifier key itself (e.g., remap CapsLock -> Left Ctrl):

```
Byte 0: 0x02
Byte 1: modifier bit (e.g., 0x01 for Left Ctrl)
Byte 2: 0x00
```

FUN_00451b90 converts HID modifier codes (0xE0-0xE7) to bitmask values:

* `0xE0` -> `0x01`, `0xE1` -> `0x02`, `0xE2` -> `0x04`, `0xE3` -> `0x08`
* `0xE4` -> `0x10`, `0xE5` -> `0x20`, `0xE6` -> `0x40`, `0xE7` -> `0x80`

Examples:

* CapsLock -> A: `02 00 04 00`
* CapsLock -> Left Ctrl: `02 01 00 00`
* Key -> Ctrl+C: `02 01 06 00`

The original software also has hardcoded combos for "shortcut" type
(`iVar5 == 7`):

| Sub-ID | Slot bytes | Meaning       |
| ------ | ---------- | ------------- |
| 1      | `02 08 07` | Win+D         |
| 2      | `02 08 08` | Win+E         |
| 3      | `02 08 0F` | Win+L         |
| 4      | `02 01 1A` | Ctrl+W        |
| 5      | `02 04 2B` | Alt+Tab       |
| 6      | `02 01 06` | Ctrl+C        |
| 7      | `02 01 19` | Ctrl+V        |
| 8      | `02 01 1B` | Ctrl+X        |
| 0x0B   | `02 01`    | Modifier only |
| 0x0A   | `02 02`    | Modifier only |

### Type 0x03: Consumer Control (Multimedia)
Remaps to a USB HID Consumer Page (0x0C) usage.

```
Byte 0: 0x03
Byte 1: Consumer usage code
Byte 2: 0x00
Byte 3: 0x00
```

Known consumer codes from the decompiled switch (sub-IDs 1-7):

| Sub-ID | Byte 1 | Consumer Usage | Meaning        |
| ------ | ------ | -------------- | -------------- |
| 1      | `0xCD` | Play/Pause     | Media play     |
| 2      | `0xB7` | Stop           | Media stop     |
| 3      | `0xB6` | Scan Previous  | Previous track |
| 4      | `0xB5` | Scan Next      | Next track     |
| 5      | `0xE9` | Volume Up      | Volume up      |
| 6      | `0xEA` | Volume Down    | Volume down    |
| 7      | `0xE2` | Mute           | Mute toggle    |

Sub-ID 0x0C uses a secondary lookup via FUN_004194e0.

### Type 0x05: Profile Switch
```
Byte 0: 0x05
Byte 1: 0x02
Byte 2: Sub-value from source data
Byte 3: 0x00
```

Used for sub-IDs 8-11 (profile switching, lock functions). The exact semantics
depend on the software's profile management.

### Type 0x06: Macro Execution
```
Byte 0: 0x06
Byte 1: Macro index (0-based position in macro list)
Byte 2: Loop count parameter
Byte 3: Additional parameter
```

The macro index is looked up from a list via FUN_0040a9a0. The source data's
`iVar14 + 0x1c` field is matched against macro IDs to find the position.

### Type 0x07: Mouse Function
```
Byte 0: 0x07
Byte 1: Mouse param 1 (from source +0x1c)
Byte 2: Mouse param 2 (from source +0x20)
Byte 3: Mouse param 3 (from source +0x24)
```

Used for mouse button/scroll remapping (sub-ID 0x0D in the original software).

## FN Layer Behavior
When `param_1 = 1` (FN layer), the function checks for keys with
`fnlayer_disable` set in `rgb-keyboard.xml`. Keys flagged as disabled in the FN
layer are skipped — their remap data is left as `00 00 00 00` regardless of what
the user configured. This prevents remapping keys that have firmware-controlled
FN functions (like FN+F1 for media).

The FN disable map is at `this+0x82c` in the KeyboardDlg object.

## Internal Data Structures

### Key-to-Slot Maps (KeyboardDlg offsets)

| Offset       | Type              | Key      | Value       |
| ------------ | ----------------- | -------- | ----------- |
| `this+0x81c` | std::map<int,int> | HID code | key_index   |
| `this+0x824` | std::map<int,int> | HID code | light_index |
| `this+0x82c` | std::map<int,int> | HID code | fn_disable  |

All three maps are populated by FUN_0041cd80 from `rgb-keyboard.xml`.

### Connection Mode Check
FUN_004348a0 checks `this+0x784` for connection mode:

* `0x784 == 0`: USB wired -> uses FUN_004185e0.
* `0x784 == 2`: 2.4G wireless -> uses FUN_004191c0 (different sender).

The `this+0x7fc` field selects which layer to send:

* `0x7fc == 0`: Normal layer only.
* `0x7fc == 1`: FN layer (also checks `this+0x7b0` for FN layer support).

## Wire Format Example
Remapping CapsLock (key_index=55) to Left Ctrl on the normal layer:

```
Step 1: 04 18 00 00 ... (begin, 64 bytes) + readback
Step 2: 04 11 00 00 00 00 00 00 09 00 ... (init, 64 bytes) + readback

Step 3: 576-byte remap buffer sent as 9 × 64-byte packets:
  Packet 1: bytes 0-63   (slots 0-15)
  Packet 2: bytes 64-127 (slots 16-31)
  ...
  At offset 220 (55*4): 02 01 00 00  (action=key, modifier=LCtrl, key=none)
  ...
  Last 2 bytes (574-575): AA 55 (trailer, uint16 0x55AA little-endian)
  + readback after last packet

Step 4: 04 02 00 00 ... (apply, 64 bytes) + readback
Step 5: 04 F0 00 00 ... (finalize, 64 bytes) + readback
```
