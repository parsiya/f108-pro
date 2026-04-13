# Key Map

## Source
From `original-files/layouts/rgb-keyboard.xml`. Key codes are USB HID usage
codes (Usage Page 0x07 - Keyboard/Keypad).

## Key Attributes
Each key has:

* `code` - HID usage code (hex)
* `name` - Display name
* `key_index` - Internal key index used by the software
* `light_index` - LED index (same as key_index on this board)
* `row_col` - Grid position as `row#col`
* `fnlayer_disable` - Whether FN-layer remap is disabled for this key

## Full Key Map

### Row 0 (Function Row)

| Code | Name         | Key Index | Row#Col |
| ---- | ------------ | --------- | ------- |
| 0x29 | Esc          | 1         | 0#0     |
| 0x3A | F1           | 2         | 0#2     |
| 0x3B | F2           | 3         | 0#3     |
| 0x3C | F3           | 4         | 0#4     |
| 0x3D | F4           | 5         | 0#5     |
| 0x3E | F5           | 6         | 0#6     |
| 0x3F | F6           | 7         | 0#7     |
| 0x40 | F7           | 8         | 0#8     |
| 0x41 | F8           | 9         | 0#9     |
| 0x42 | F9           | 10        | 0#10    |
| 0x43 | F10          | 11        | 0#11    |
| 0x44 | F11          | 12        | 0#12    |
| 0x45 | F12          | 13        | 0#13    |
| 0x46 | Print Screen | 112       | 0#14    |
| 0x47 | Scroll Lock  | 113       | 0#15    |
| 0x48 | Pause        | 115       | 0#16    |

### Row 1 (Number Row)

| Code | Name      | Key Index | Row#Col |
| ---- | --------- | --------- | ------- |
| 0x35 | ~\`       | 19        | 1#0     |
| 0x1E | !1        | 20        | 1#1     |
| 0x1F | @2        | 21        | 1#2     |
| 0x20 | #3        | 22        | 1#3     |
| 0x21 | $4        | 23        | 1#4     |
| 0x22 | %5        | 24        | 1#5     |
| 0x23 | ^6        | 25        | 1#6     |
| 0x24 | &7        | 26        | 1#7     |
| 0x25 | *8        | 27        | 1#8     |
| 0x26 | (9        | 28        | 1#9     |
| 0x27 | )0        | 29        | 1#10    |
| 0x2D | _-        | 30        | 1#11    |
| 0x2E | +=        | 31        | 1#12    |
| 0x2A | Backspace | 103       | 1#13    |
| 0x49 | Insert    | 116       | 1#14    |
| 0x4A | Home      | 117       | 1#15    |
| 0x4B | Page Up   | 118       | 1#16    |
| 0x53 | Num Lock  | 32        | 1#17    |
| 0x54 | Numpad /  | 33        | 1#18    |
| 0x55 | Numpad *  | 34        | 1#19    |
| 0x56 | Numpad -  | 122       | 1#20    |

### Row 2 (QWERTY Row)

| Code | Name      | Key Index | Row#Col |
| ---- | --------- | --------- | ------- |
| 0x2B | Tab       | 37        | 2#0     |
| 0x14 | Q         | 38        | 2#1     |
| 0x1A | W         | 39        | 2#2     |
| 0x08 | E         | 40        | 2#3     |
| 0x15 | R         | 41        | 2#4     |
| 0x17 | T         | 42        | 2#5     |
| 0x1C | Y         | 43        | 2#6     |
| 0x18 | U         | 44        | 2#7     |
| 0x0C | I         | 45        | 2#8     |
| 0x12 | O         | 46        | 2#9     |
| 0x13 | P         | 47        | 2#10    |
| 0x2F | {[        | 48        | 2#11    |
| 0x30 | }}        | 49        | 2#12    |
| 0x31 | \|\       | 67        | 2#13    |
| 0x4C | Delete    | 119       | 2#14    |
| 0x4D | End       | 120       | 2#15    |
| 0x4E | Page Down | 121       | 2#16    |
| 0x5F | Numpad 7  | 50        | 2#17    |
| 0x60 | Numpad 8  | 51        | 2#18    |
| 0x61 | Numpad 9  | 52        | 2#19    |
| 0x57 | Numpad +  | 123       | 2#20    |

### Row 3 (Home Row)

| Code | Name     | Key Index | Row#Col |
| ---- | -------- | --------- | ------- |
| 0x39 | CapsLock | 55        | 3#0     |
| 0x04 | A        | 56        | 3#1     |
| 0x16 | S        | 57        | 3#2     |
| 0x07 | D        | 58        | 3#3     |
| 0x09 | F        | 59        | 3#4     |
| 0x0A | G        | 60        | 3#5     |
| 0x0B | H        | 61        | 3#6     |
| 0x0D | J        | 62        | 3#7     |
| 0x0E | K        | 63        | 3#8     |
| 0x0F | L        | 64        | 3#9     |
| 0x33 | :;       | 65        | 3#10    |
| 0x34 | "'       | 66        | 3#11    |
| 0x28 | Enter    | 85        | 3#13    |
| 0x5C | Numpad 4 | 68        | 3#17    |
| 0x5D | Numpad 5 | 69        | 3#18    |
| 0x5E | Numpad 6 | 70        | 3#19    |

### Row 4 (Shift Row)

| Code | Name         | Key Index | Row#Col |
| ---- | ------------ | --------- | ------- |
| 0xE1 | Shift_L      | 73        | 4#0     |
| 0x1D | Z            | 74        | 4#2     |
| 0x1B | X            | 75        | 4#3     |
| 0x06 | C            | 76        | 4#4     |
| 0x19 | V            | 77        | 4#5     |
| 0x05 | B            | 78        | 4#6     |
| 0x11 | N            | 79        | 4#7     |
| 0x10 | M            | 80        | 4#8     |
| 0x36 | ,<           | 81        | 4#9     |
| 0x37 | .>           | 82        | 4#10    |
| 0x38 | /?           | 83        | 4#11    |
| 0xE5 | Shift_R      | 84        | 4#13    |
| 0x52 | Up           | 101       | 4#15    |
| 0x59 | Numpad 1     | 86        | 4#17    |
| 0x5A | Numpad 2     | 87        | 4#18    |
| 0x5B | Numpad 3     | 88        | 4#19    |
| 0x58 | Numpad Enter | 106       | 4#20    |

### Row 5 (Bottom Row)

| Code | Name       | Key Index | Row#Col |
| ---- | ---------- | --------- | ------- |
| 0xE0 | Ctrl_L     | 91        | 5#0     |
| 0xE3 | Win_L      | 92        | 5#1     |
| 0xE2 | Alt_L      | 93        | 5#2     |
| 0x2C | Space      | 94        | 5#6     |
| 0xE6 | Alt_R      | 95        | 5#10    |
| 0xAF | Fn         | 96        | 5#11    |
| 0x65 | APP (Menu) | 97        | 5#12    |
| 0xE4 | Ctrl_R     | 98        | 5#13    |
| 0x50 | Left       | 99        | 5#14    |
| 0x51 | Down       | 100       | 5#15    |
| 0x4F | Right      | 102       | 5#16    |
| 0x62 | Numpad 0   | 104       | 5#18    |
| 0x63 | Numpad .   | 105       | 5#19    |

## Notes

* Key index is not sequential - there are gaps (e.g., no indices 14-18, 35-36,
  53-54, 71-72, 89-90, etc.)
* The Fn key uses code `0xAF` which is not a standard USB HID code - it's a
  vendor-specific code
* Total keys: 108 (full-size layout including numpad)
* Light index matches key index 1:1 on this keyboard

## Lighting Config
From `rgb-keyboard.xml`. See [hid-protocol.md](hid-protocol.md) for the full
lighting mode table and protocol.

* `default_mode="11"` - Default lighting mode ID
* `default_brightness="5"`, `brightness_max="5"` - 5 brightness levels
* `default_speed="3"`, `speed_max="5"` - 5 speed levels
* Grid: 21 columns x 6 rows

## Screen Config
From `rgb-keyboard.xml`. See [hid-protocol.md](hid-protocol.md) for the LCD
upload protocol.

* `gif_headlength="256"` - GIF header size
* `gif_maxframes="141"` - Max animation frames
* `gif_count="1"` - Number of GIF slots
