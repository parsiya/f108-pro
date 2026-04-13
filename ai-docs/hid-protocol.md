# HID Communication Protocol

## Overview
The software communicates with the keyboard using the Windows HID API. It does
NOT use `ReadFile`/`WriteFile` for normal commands. Instead, it uses:

* `HidD_SetFeature` - to send commands (feature reports) to the keyboard
* `DeviceIoControl` with IOCTL `0xb0192` (`IOCTL_HID_GET_FEATURE`) - to read
  responses back

The HID functions are loaded dynamically from `hid.dll` at runtime (see
`FUN_004509e0` at `0x004509e0`).

**Scope**: This document focuses on the USB wired protocol. The 2.4G wireless
mode uses a simplified single-packet protocol (command `05 10`) instead of the
multi-step flow documented here. Bluetooth configuration is not supported by
the software.

## Light Zones
The F108 Pro has three independent light zones:

| Zone           | Manual Name       | Software Control | FN Key Control |
| -------------- | ----------------- | ---------------- | -------------- |
| Main backlight | Backlight Effects | **Yes** (works)  | Yes            |
| Light bar      | Lightbar          | **No**           | Yes            |
| Side strips    | Sidelight Effects | **No**           | Yes            |

### Main Backlight
The per-key RGB LEDs under each keycap. Fully controllable via HID commands (20
modes). This is what we've implemented.

### Light Bar
The small LED strip on top of the arrow key cluster. Controlled only by the
keyboard's FN key combos:

| FN Combo        | Action                     |
| --------------- | -------------------------- |
| FN + Left Shift | Toggle lightbar effects    |
| FN + Left Ctrl  | Toggle lightbar speed      |
| FN + Left Alt   | Toggle lightbar brightness |
| FN + Z          | Toggle lightbar colors     |

### Sidelight Effects
The LED strips on the underside/sides of the keyboard case. Controlled only by
FN key combos:

| FN Combo         | Action                      |
| ---------------- | --------------------------- |
| FN + Right Shift | Toggle sidelight effects    |
| FN + Right Ctrl  | Toggle sidelight speed      |
| FN + Right Alt   | Toggle sidelight brightness |
| FN + /?          | Toggle sidelight colors     |

The Aula software has sidelight code (modes `0x1f`-`0x28`, command `04 13`) but
it is **disabled** for the F108 Pro (`sidelight="0"` in `rgb-keyboard.xml`).
Sending those commands has no visible effect — the firmware accepts them but the
light bar and sidelights appear to be on a separate internal controller not
reachable via HID.

### Connection Mode
Switch wireless/wired connection:

| FN Combo | Action      |
| -------- | ----------- |
| FN + ~   | USB 2.4G    |
| FN + 1   | Bluetooth 1 |
| FN + 2   | Bluetooth 2 |
| FN + 3   | Bluetooth 3 |
| FN + 4   | USB-C       |

### Device Type
Switch the device profile for the connected OS:

| FN Combo | Action  |
| -------- | ------- |
| FN + Q   | Android |
| FN + W   | Windows |
| FN + E   | Mac     |
| FN + R   | iOS     |

These are useful when the LCD is broken or showing custom images since the LCD
normally displays connection/device status.

## HID Function Pointers (Global Variables)

| Address      | Function                     |
| ------------ | ---------------------------- |
| `0x005950a0` | `HidD_SetFeature`            |
| `0x0059509c` | `HidD_GetFeature`            |
| `0x005950b4` | `HidD_GetAttributes`         |
| `0x00595098` | `HidD_GetSerialNumberString` |
| `0x005950b0` | `HidD_GetManufacturerString` |
| `0x005950b8` | `HidD_GetProductString`      |
| `0x005950bc` | `HidD_GetIndexedString`      |
| `0x005950a8` | `HidD_GetPreparsedData`      |
| `0x005950a4` | `HidD_FreePreparsedData`     |
| `0x005950ac` | `HidP_GetCaps`               |

## Packet Format

### Feature Report (SetFeature)
The Windows software sends 65-byte feature reports (`0x41` bytes) with
byte 0 as the report ID (`0x00`). However, the keyboard's HID report
descriptor on **interface 3** does NOT declare a report ID, so the
actual feature report is **64 bytes** on the wire.

#### Windows Behavior
`HidD_SetFeature(handle, buf, 0x41)` where `buf[0]=0x00` (report ID) and
`buf[1..64]` is the payload. The Windows HID stack strips the report ID byte
before sending over USB.

#### Linux/libusb Behavior
Send 64 bytes directly via USB control transfer. The report ID is part of the
`wValue` field, not the data buffer.

#### Payload Offset Convention
All payload offsets in this document are relative to the 64-byte payload that
goes on the wire. No report ID byte is included in the data buffer when using
libusb.

### Read-back (GetFeature)
After sending certain commands (those marked "readback=Yes"), the software reads
back a feature report from the keyboard.

**Windows**: Uses `DeviceIoControl` with IOCTL `0xb0192`
(`IOCTL_HID_GET_FEATURE`) to read back into the same buffer.

**Linux/libusb**: Uses GET_REPORT control transfer (64 bytes).

**The readback IS required.** Without it, the keyboard ignores subsequent
commands in the sequence. The keyboard's response has byte[3] set to `0x01` as
an acknowledgment. Example responses observed:

```
Sent:     04 18 00 00 ...
Response: 04 18 00 01 00 00 ...   (byte[3]=0x01 = ACK)

Sent:     04 02 00 00 ...
Response: 04 02 00 01 07 02 ...   (byte[4..5] = current mode/state)
```

### No State Query Commands (Write-Only Protocol)
The keyboard does not support reading back its current configuration (lighting
mode, brightness, speed, color, key remaps, etc.). The original Windows software
reads all settings from a local SQLite database on startup and pushes them to
the keyboard. It never queries the device for its current state.

The GET_REPORT readback after each command is purely an ACK mechanism (byte[3] =
0x01). The `04 02` apply response includes bytes 4-5 which may indicate
mode/state, but no dedicated "read current settings" command exists in the
protocol.

This means any tool that changes settings must track the desired state
locally. The database is the source of truth; the keyboard is a write-only
sink.

### USB Interface Details (Verified)

The keyboard exposes 4 HID interfaces:

| Interface | Protocol        | Usage Page | Feature Reports | Purpose                       |
| --------- | --------------- | ---------- | --------------- | ----------------------------- |
| 0         | Keyboard        | 0x01       | No              | Standard key input            |
| 1         | Mouse           | 0x0C/0x01  | No              | Media/mouse input             |
| 2         | Vendor (0xFF68) | 0xFF68     | No              | **LCD data** (4096B Output)   |
| 3         | Vendor (0xFF13) | 0xFF13     | **Yes (64 B)**  | **Configuration** (64B Feat.) |

**Interface 3 is the configuration endpoint.** The Windows software uses
`MI_00` in its HID interface filter string, but on the USB level, the
feature report descriptor is on interface 3.

### Volume Knob

The physical volume knob is **firmware-controlled** and cannot be remapped
through the software's remap protocol. It sends Consumer Control HID reports
directly on Interface 1 (Report ID 0x03) with a 16-bit usage code.

Interface 1 contains five report collections:

| Report ID | Usage Page      | Usage            | Description                    |
| --------- | --------------- | ---------------- | ------------------------------ |
| 0x03      | Consumer (0x0C) | Consumer Control | 16-bit consumer code (knob)    |
| 0x02      | Generic (0x01)  | System Control   | Power down/sleep/wake (3 bits) |
| 0x01      | Keyboard (0x07) | Keyboard         | 120-key NKRO bitmap            |
| 0x06      | Generic (0x01)  | Mouse            | 3 buttons + X/Y/wheel          |
| 0x05      | Vendor (0xFFFF) | Vendor           | 3 bytes vendor-specific        |

The knob generates standard consumer control codes:

* Rotate clockwise: Volume Up (0xE9)
* Rotate counter-clockwise: Volume Down (0xEA)
* Press: Mute (0xE2) or Play/Pause (0xCD)

FN + knob press toggles the knob between two modes:

* Volume mode: rotate = volume up/down, press = mute.
* Media mode: rotate = volume up/down, press = play/pause.

The knob is not listed in `rgb-keyboard.xml` (no `key_index`), there are no
knob-related strings or functions in the binary, and the `config.xml` has no
knob configuration entries. The firmware decides what codes the knob sends
and there is no HID command to change this behavior.

### Linux/libusb Implementation (Verified Working)

Using `github.com/google/gousb` (Go wrapper for libusb):

```go
// Open device.
dev, _ := ctx.OpenDeviceWithVIDPID(0x0C45, 0x800A)
dev.SetAutoDetach(true)
cfg, _ := dev.Config(1)
intf, _ := cfg.Interface(3, 0)  // Interface 3!

// SET_REPORT (HID Feature).
var buf [64]byte
copy(buf[:], payload)
dev.Control(0x21, 0x09, 0x0300, 3, buf[:])

// GET_REPORT (HID Feature) - for readback.
var rbuf [64]byte
dev.Control(0xA1, 0x01, 0x0300, 3, rbuf[:])
```

USB control transfer parameters:

| Parameter     | SET_REPORT | GET_REPORT |
| ------------- | ---------- | ---------- |
| bmRequestType | `0x21`     | `0xA1`     |
| bRequest      | `0x09`     | `0x01`     |
| wValue        | `0x0300`   | `0x0300`   |
| wIndex        | `3`        | `3`        |
| data length   | 64         | 64         |

The `wValue` is `0x0300` = feature report type (`0x03`) << 8 | report ID
(`0x00`).

**Permissions**: Requires write access to the USB device. Either run as root,
or set a udev rule:

```
SUBSYSTEM=="usb", ATTR{idVendor}=="0c45", ATTR{idProduct}=="800a", MODE="0666"
```

**Delay**: 35ms between commands (from `config.xml` `cmd_delaytime` value).

### Command Structure (within the 64-byte payload)

Based on analysis of multiple command-building functions, a common pattern is:

```
Byte 0: Command byte 1 (e.g., 0x04, 0x00)
Byte 1: Command byte 2 (e.g., 0x18, 0x01, 0x17, 0x11, 0x27, etc.)
Bytes 2+: Command-specific parameters
```

### Trailer
Many packets include a two-byte trailer near the end of the payload as a
validation marker. The Ghidra decompilation shows `uint16 0x55AA`, which is
stored little-endian in memory: the bytes on the wire are `0xAA 0x55`.

This was confirmed by USB traffic capture — the Windows software sends bytes
`AA 55`, not `55 AA`. Getting this wrong causes the keyboard to silently
ignore the command (ACKs are returned normally, but the data is not applied).

## Command IDs (Observed)

### Preamble/Postamble Commands
These wrap the actual data commands:

| Bytes 0-1 | Description                 | Notes                                     |
| --------- | --------------------------- | ----------------------------------------- |
| `04 18`   | Begin transaction           | Sent before data commands, with read-back |
| `04 17`   | Begin transaction (variant) | Byte 2 = `0x01`, byte 6 = `0x01`          |
| `04 02`   | End transaction / Apply     | Sent after data, with read-back           |
| `04 F0`   | Final command               | Sent last in some sequences               |

### Data Commands

| Bytes 0-1 | Description               | Notes                                                      |
| --------- | ------------------------- | ---------------------------------------------------------- |
| `00 01`   | Function settings         | FN switch, sleep time, key response time, trailer `0x55AA` |
| `00 80`   | Screen/sidelight config   | Byte 1 data from offset `0x814`                            |
| `04 11`   | Key remap data (normal)   | Byte 2 = `0x09`                                            |
| `04 27`   | Key remap data (FN layer) | Byte 2 = some value                                        |
| `04 13`   | LED/sidelight config      | Byte 2 = `0x01`                                            |

### Key Remap
See [key-remap-protocol.md](key-remap-protocol.md) for the full key remapping
protocol (normal layer `04 11`, FN layer `04 27`, 576-byte remap tables, and
all action types).

## Communication Flow
A typical command sequence:

1. Send `04 18` (begin) -> read back response
2. Send command-specific "mode" byte -> read back response
3. Send data payload (may be multi-packet)
4. Send `04 02` (apply) -> read back response
5. Optionally send `04 F0` (finalize)

## Send Functions
The primary send function (`FUN_0044edc0`) handles both single-packet and
multi-packet modes. For payloads < 66 bytes, it sends a single feature report.
For larger payloads, it splits into 64-byte packets. A `param_3` flag controls
whether a GET_REPORT readback follows. The inter-command delay is configurable
(offset `0x30` in the device context).

See [ghidra-functions.md](ghidra-functions.md) for the full function index
including send variants, the SetFeature wrapper, and the IOCTL readback
function.

## Delay Configuration
From `config.xml`: `<cmd_delaytime value="35" />` - 35 ms delay between
commands.

## Lighting Protocol (Detailed)

### Lighting Mode / Effect (USB Wired)

**Sender function**: `FUN_0042b040` at `0x0042b040`

**Sequence**:

| Step | Packet     | Readback | Purpose               |
| ---- | ---------- | -------- | --------------------- |
| 1    | `04 18`    | Yes      | Begin transaction     |
| 2    | `04 13 01` | Yes      | Lighting command init |
| 3    | Data       | No       | Mode + parameters     |
| 4    | `04 02`    | Yes      | Apply                 |
| 5    | `04 F0`    | No       | Finalize              |

**Step 2 detail**: Byte[0]=`04`, Byte[1]=`13`, Byte[8]=`01`.

**Step 3 data packet layout** (payload offsets, prepend `0x00` report ID):

```
Offset  Field         Source                          Values
0       mode          t_light_data.mode               0=off, 1-N=effect IDs
1       R             color_value & 0xFF              0-255
2       G             (color_value >> 8) & 0xFF       0-255
3       B             (color_value >> 16) & 0xFF      0-255
4-7     (reserved)    0x00
8       colorful      t_light_data.colorful           0=single color, 1=rainbow
9       brightness    t_light_data.brightness         0-5
10      speed         t_light_data.speed              0-5
11      direction     t_light_data.direction          0-1
12-13   (reserved)    0x00
14-15   trailer       0x55AA (uint16 LE)
16-63   (padding)     0x00
```

**Note**: When mode=0 (off), the R/G/B, colorful, brightness, speed, and
direction fields are NOT populated (left as zero).

**DB table**: `t_light_data` with columns: `mode`, `brightness`, `speed`,
`direction`, `colorful`, `colorindex`, `color_value`, `config_func`,
`reserved`, `status`.

### Lighting Mode IDs
Source: `FUN_0042b2f0` (mode init) + `1033.lan` (English language file). Default
mode from XML: `default_mode=11` (Rolling).

| Mode ID | Name       | config_func | Brightness | Speed | Direction | Color | Colorful |
| ------- | ---------- | ----------- | ---------- | ----- | --------- | ----- | -------- |
| 0       | LED Off    | `0x00`      | -          | -     | -         | -     | -        |
| 1       | Static     | `0x31`      | Yes        | -     | -         | -     | Yes      |
| 2       | SingleOn   | `0x33`      | Yes        | Yes   | -         | -     | Yes      |
| 3       | SingleOff  | `0x33`      | Yes        | Yes   | -         | -     | Yes      |
| 4       | Glittering | `0x33`      | Yes        | Yes   | -         | -     | Yes      |
| 5       | Falling    | `0x33`      | Yes        | Yes   | -         | -     | Yes      |
| 6       | Colourful  | `0x03`      | Yes        | Yes   | -         | -     | -        |
| 7       | Breath     | `0x33`      | Yes        | Yes   | -         | -     | Yes      |
| 8       | Spectrum   | `0x03`      | Yes        | Yes   | -         | -     | -        |
| 9       | Outward    | `0x33`      | Yes        | Yes   | -         | -     | Yes      |
| 10      | Scrolling  | `0x3b`      | Yes        | Yes   | Dir(LR)   | -     | Yes      |
| 11      | Rolling    | `0x37`      | Yes        | Yes   | Dir(UD)   | -     | Yes      |
| 12      | Rotating   | `0x37`      | Yes        | Yes   | Dir(UD)   | -     | Yes      |
| 13      | Explode    | `0x33`      | Yes        | Yes   | -         | -     | Yes      |
| 14      | Launch     | `0x33`      | Yes        | Yes   | -         | -     | Yes      |
| 15      | Ripples    | `0x33`      | Yes        | Yes   | -         | -     | Yes      |
| 16      | Flowing    | `0x37`      | Yes        | Yes   | Dir(UD)   | -     | Yes      |
| 17      | Pulsating  | `0x33`      | Yes        | Yes   | -         | -     | Yes      |
| 18      | Tilt       | `0x37`      | Yes        | Yes   | Dir(UD)   | -     | Yes      |
| 19      | Shuttle    | `0x33`      | Yes        | Yes   | -         | -     | Yes      |

#### config_func Bitmask
The `config_func` field controls which UI elements are visible for each mode:

| Bit | Hex    | Feature                        |
| --- | ------ | ------------------------------ |
| 0   | `0x01` | Brightness slider              |
| 1   | `0x02` | Speed slider                   |
| 2   | `0x04` | Direction Left/Right radio     |
| 3   | `0x08` | Direction Up/Down radio        |
| 4   | `0x10` | RGB color sliders              |
| 5   | `0x20` | Color picker + Colorful toggle |

**Sender function**: `FUN_0042b1e0` at `0x0042b1e0`

Wireless mode sends a single all-in-one packet via `FUN_0044f4e0` instead of the
multi-step USB protocol:

```
Offset  Field         Value
0       (unused)      0x00
1-2     command       0x05 0x10
3       (reserved)    0x00
4       mode          effect ID
5       R             color R
6       G             color G
7       B             color B
8-11    (reserved)    0x00
12      colorful      0=single, 1=rainbow
13      brightness    0-5
14      speed         0-5
15      direction     0-1
16-17   (reserved)    0x00
18-19   trailer       0x55AA
20-63   (padding)     0x00
```

### Per-Key RGB Lighting (User Light)

**Sender function**: `FUN_0044b910` at `0x0044b910`

**Sequence**:

| Step | Packet    | Readback | Purpose            |
| ---- | --------- | -------- | ------------------ |
| 1    | Sidelight | -        | Calls FUN_0044b800 |
| 2    | `04 18`   | Yes      | Begin transaction  |
| 3    | `04 23`   | Yes      | Per-key data init  |
| 4    | Data      | Yes      | Key RGB data       |
| 5    | `04 02`   | Yes      | Apply              |
| 6    | `04 F0`   | Yes      | Finalize           |

**Step 3 detail**: Byte[0]=`04`, Byte[1]=`23`, Byte[2]=`03` (monochrome) or
`09` (RGB). The mode depends on `this+0x7ac` flag.

**Step 4 data formats**:

*Monochrome mode* (0xC0 = 192 bytes, multi-packet): One byte per key
`light_index`. `0xFF` = on, `0x00` = off. Trailer `0x55AA` at end.

*RGB mode* (0x240 = 576 bytes, multi-packet): Four bytes per key:

```
Byte 0: light_index
Byte 1: R
Byte 2: G
Byte 3: B
```

Trailer `0x55AA` at end.

### Sidelight

**Sender function**: `FUN_0044b800` at `0x0044b800`

**Sequence**: `04 18` -> `04 13 01` -> `00 80 data` -> `04 02` -> `04 F0`

The data packet has byte[0]=`0x00`, byte[1]=`0x80`, byte[2]=default brightness
value from `this+0x814` (initialized from XML `default_brightness=5`). This
function is called before per-key RGB data as a preamble.

**Note**: The `sidelight` feature is **disabled** on the F108 Pro
(`sidelight="0"` in `rgb-keyboard.xml`), so the UI never shows these controls.

### LED Layer / Sidelight Mode

**Sender function**: `FUN_00433fd0` at `0x00433fd0`

**Sequence**: `04 18` -> `04 13 01` -> data -> `04 02` -> `04 F0`

Data packet byte[0] = sidelight mode value from `this+0x808`. The value is the
mode's `config_value + 0x1f`, set by `FUN_004340d0` when the user selects a
sidelight mode from the combo box (task type 10 in the dispatcher).

### Sidelight Mode IDs
Source: `FUN_004340d0` + `1033.lan`. These modes exist in the code for other
Aula keyboard models that have software-controllable sidelights. On the F108
Pro, sending these commands has no visible effect (see Light Zones section
above). The light bar and sidelight strips are controlled only by FN key combos
on this model. See [sidelight-investigation.md](sidelight-investigation.md) for
the full investigation.

| Mode Index | Byte Value | Language ID | Name          |
| ---------- | ---------- | ----------- | ------------- |
| 0          | `0x1f`     | 661         | Flowing Light |
| 1          | `0x20`     | 662         | Red           |
| 2          | `0x21`     | 663         | Yellow        |
| 3          | `0x22`     | 664         | Green         |
| 4          | `0x23`     | 665         | Ice Blue      |
| 5          | `0x24`     | 666         | Blue          |
| 6          | `0x25`     | 667         | Pink          |
| 7          | `0x26`     | 668         | White         |
| 8          | `0x27`     | 669         | Neon          |
| 9          | `0x28`     | 670         | Off           |

### LCD Screen Image Upload (Verified Working)

**Sender function**: `FUN_00422b50` at `0x00422b50`

**Status**: **Working.** Both single-frame images and multi-frame GIF animations
upload successfully. The initial attempt using control transfers (SET_REPORT)
crashed the firmware. The fix was switching to **interrupt endpoint transfers**
on interface 2 (EP 3 OUT for data, EP 4 IN for acks). See
[lcd-upload-investigation.md](lcd-upload-investigation.md) for the full
investigation trail.

**Key discovery**: The image data uses a **different USB interface** than the
control commands:

| Interface | Report Type | Size   | Purpose                        |
| --------- | ----------- | ------ | ------------------------------ |
| 3         | Feature     | 64 B   | Control (begin, header, apply) |
| 2         | Output      | 4096 B | Image data pages               |
| 2         | Input       | 64 B   | Data page acknowledgment       |

Interface 2 HID descriptor (usage page `0xFF68`):

* Output Report: 4096 bytes (for data pages)
* Input Report: 64 bytes (for acknowledgment)

#### Buffer Format

* Header (256 bytes): byte[0] = frame count, byte[1..N] = per-frame delay
  (frame_duration_centiseconds / 2, min 1), rest 0xFF.
* Pixel data (64800 bytes per frame): 240x135 RGB565, little-endian,
  frames packed contiguously starting at byte 256.
* Total for 1 frame: 65056 bytes = 16 pages of 4096 bytes.

#### Original GIF from Software
`original-files/gif/AULA F108Pro 三模机械键盘/0.gif`:

| Property           | Value                                |
| ------------------ | ------------------------------------ |
| File size          | 3,009,866 bytes (~2.9 MB compressed) |
| Format             | GIF89a                               |
| Dimensions         | 240 x 135                            |
| Frame count        | 214                                  |
| Per-frame (RGB565) | 64,800 bytes (240 * 135 * 2)         |
| Total raw (RGB565) | ~13.2 MB uncompressed                |
| Pages (1 frame)    | 16 (256 header + 64800 data)         |
| Pages (214 frames) | 3,386                                |

This confirms the screen dimensions and pixel format. The GIF is a standard
GIF89a file that the software decodes, converts to RGB565, and uploads using the
protocol below. When looking for test GIFs, they must be exactly **240x135 pixels**.

**Protocol sequence** (USB wired):

1. `04 18` begin (Feature Report, interface 3, with readback)
2. `04 72` header (Feature Report, interface 3, with readback):
   byte[2]=image_number, byte[8-9]=page_count (uint16 LE)
3. Data pages (Output Report, interface 2, 4096 bytes each, with ReadFile
   acknowledgment at 300ms timeout)
4. `04 02` apply (Feature Report, interface 3, with readback)

**Windows implementation**: Uses `WriteFile` + `ReadFile` (overlapped I/O) on a
separate HID handle opened for interface 2. The `FUN_0044f1c0` function sends
4097 bytes (1 byte report ID + 4096 data) via `WriteFile`, then reads back via
`ReadFile` with 300ms timeout.

**Linux/libusb implementation**: Data pages MUST be sent via **interrupt
endpoint transfers** on interface 2 (EP 3 OUT, address `0x03`), NOT control
transfers. Acknowledgments are read from EP 4 IN (address `0x84`). Using control
transfers (SET_REPORT) crashes the keyboard firmware. Each 4096-byte write is
automatically fragmented by libusb into 64 × 64-byte interrupt packets. The ACK
for each page is `01 5a 02 00 ...` (64 bytes).

```go
// Interrupt endpoints on interface 2.
outEP, _ := intf2.OutEndpoint(3)  // EP 3 OUT
inEP, _ := intf2.InEndpoint(4)    // EP 4 IN

// Send 4096 bytes (no report ID prefix).
outEP.Write(pageData)

// Read 64-byte ack with 300ms timeout.
ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
inEP.ReadContext(ctx, ackBuf)
```

### LCD Clock/DateTime Sync (Verified Working)

**Sender function**: `FUN_00423b10` at `0x00423b10`

Syncs system time to the keyboard's LCD clock. **Verified working** via
`aula-go/cmd/clocksync/`.

**Sequence** (all steps use readback, no finalize `04 F0`):

| Step | Packet  | Readback | Purpose           |
| ---- | ------- | -------- | ----------------- |
| 1    | `04 18` | Yes      | Begin transaction |
| 2    | `04 28` | Yes      | Clock init        |
| 3    | Data    | Yes      | Date/time data    |
| 4    | `04 02` | Yes      | Apply             |

**Step 2 detail**: byte[0]=`04`, byte[1]=`28`, byte[8]=`01`.

**Step 3 data packet layout** (payload offsets):

```
Offset  Field         Values
0       (zero)        0x00
1       profile       1 (profile number)
2       magic         0x5A
3       year          year % 2000 (e.g., 26 for 2026)
4       month         1-12
5       day           1-31
6       hour          0-23
7       minute        0-59
8       second        0-59
9       (unused)      0x00
10      day_of_week   0=Sun, 1=Mon, 2=Tue, 3=Wed, 4=Thu, 5=Fri, 6=Sat
11-61   (padding)     0x00
62-63   trailer       0x55AA
```

## Complete Command Reference

| Bytes 0-1 | Function    | Direction  | Description                    | Verified |
| --------- | ----------- | ---------- | ------------------------------ | -------- |
| `04 18`   | Transaction | Host -> KB | Begin transaction              | Yes      |
| `04 02`   | Transaction | Host -> KB | Apply / commit                 | Yes      |
| `04 F0`   | Transaction | Host -> KB | Finalize                       | Yes      |
| `04 17`   | Function    | Host -> KB | Function settings header       | No       |
| `00 01`   | Function    | Host -> KB | FN/sleep/respond settings data | No       |
| `04 11`   | Key remap   | Host -> KB | Standard layer remap data      | No       |
| `04 27`   | Key remap   | Host -> KB | FN layer remap data            | No       |
| `04 13`   | Lighting    | Host -> KB | Lighting command init          | Yes      |
| `04 23`   | Lighting    | Host -> KB | Per-key RGB data init          | No       |
| `04 20`   | Lighting    | Host -> KB | Custom light real-time preview | No       |
| `04 F5`   | Lighting    | KB -> Host | Key color readback             | No       |
| `00 80`   | Sidelight   | Host -> KB | Sidelight brightness data      | No       |
| `04 72`   | LCD         | Host -> KB | Image upload header            | Yes      |
| `04 28`   | LCD         | Host -> KB | Clock sync header              | Yes      |
| `04 19`   | Macro       | Host -> KB | Macro init                     | No       |
| `04 15`   | Macro       | Host -> KB | Macro data header + payload    | No       |
| `05 10`   | Wireless    | Host -> KB | Wireless lighting (all-in-one) | No       |
| `7F 03`   | Wireless    | Host -> KB | Wireless LCD upload header     | No       |
