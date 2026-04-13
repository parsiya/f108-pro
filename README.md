# f108-pro
A command-line tool for configuring the Aula F108 Pro mechanical keyboard on
Linux, WSL2, and Windows. Supports backlight mode control, per-key RGB, key
remapping, LCD clock sync, and LCD image upload.

Please read the accompanying blog for details and discussion:  
[AI Borked my Keyboard - Reversing the Aula F108 Pro Software][blog].

[blog]: https://parsiya.net/blog/ai-borked-keyboard/

## Features

* Backlight control: Set any of the 20 built-in lighting effects with custom
  color, brightness, speed, and direction.
* Per-key RGB: Set individual key colors via CLI or YAML layout files.
* Key remapping: Remap keys on the normal and FN layers (key swap,
  multimedia, mouse, key combos).
* LCD clock sync: Set the keyboard's LCD screen clock to system time.
* LCD image upload: **Use with caution**. Upload a custom image to the
  keyboard's 240x135 LCD screen.

## Setup

### Windows
The tool uses the native HID API (`hid.dll`) and does not need any additional
drivers or libraries.

```powershell
go build -o f108-pro.exe .
```

Cross-compile from Linux/WSL2:

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o f108-pro.exe .
```

### WSL2
WSL2 does not see USB devices by default. Forward the keyboard from Windows
using `usbipd-win`. **This means you need a second keyboard because after you
attach the F108 Pro to WSL2, it will stop working in Windows**.

On Windows (PowerShell as Administrator):

```powershell
winget install usbipd
usbipd list
usbipd bind --busid <BUSID> 
usbipd attach --wsl --busid <BUSID>
```

In WSL2, install dependencies and set permissions:

```bash
sudo apt-get install -y libusb-1.0-0-dev pkg-config usbutils
lsusb | grep 0c45:800a
sudo chmod 666 /dev/bus/usb/<BUS>/<DEV>
```

When done, detach the keyboard back to Windows:

```powershell
usbipd detach --busid <BUSID>
```

Build is the same as native Linux: `go build .`

### Linux
**Linux instructions are untested and might be LLM hallucinations.**
Requires libusb development headers.

```bash
sudo apt-get install -y libusb-1.0-0-dev pkg-config
```

Set up USB permissions (so you don't need root):

```bash
echo 'SUBSYSTEM=="usb", ATTR{idVendor}=="0c45", ATTR{idProduct}=="800a", MODE="0666"' \
  | sudo tee /etc/udev/rules.d/99-aula.rules
sudo udevadm control --reload-rules
sudo udevadm trigger
```

Build:

```bash
go build .
```

### Ghidra MCP
If you want to follow along and let (A)I reverse the binary.

1. Download the F108 Pro driver. Current locations are:
  1. `v1.0.0.1`: https://aulagear.com/blogs/software/aula-f108-pro-driver
  2. `v1.0.0.3`: https://aulakeyboard.com/download/f108-pro-drive/
2. Extract the contents or install the package.
  1. We will only need `DeviceDriver.exe` but it's good for AI to have access to the rest of the files.
3. Download and extract Ghidra to a path.
4. Follow the instructions in [LaurieWired/GhidraMCP][ghidra-mcp] to:
  1. Download the Python bridge and extension.
  2. Add the extension to Ghidra.
5. Update `.vscode/mcp.json` with **absolute paths** to the Ghidra MCP bridge
   location for VS Code with GitHub Copilot Chat. Otherwise use your own client.
  1. Note I've created a venv for ease of use, but it's not necessary.
6. Run Ghidra and open the `DeviceDriver.exe` file from the installation directory.
7. Let AI loose on the binary.

[ghidra-mcp]: https://github.com/lauriewired/ghidramcp

## Usage

### Backlight Mode

```bash
f108-pro light <mode> [brightness] [speed] [r g b | colorful] [-d direction]
```

* `mode`: 0-19 or name (run `f108-pro modes` to list).
* `brightness`: 0-5 (default 5).
* `speed`: 0-5 (default 3).
* `r g b`: 0-255 each.
* `colorful`: Use rainbow colors.
* `-d`: Direction 0-3 (for Scrolling, Rolling, Rotating, Flowing, Tilt).

```bash
# Static red, max brightness:
f108-pro light static 5 0 255 0 0

# Breathing blue, speed 3:
f108-pro light breath 5 3 0 0 255

# Rainbow spectrum cycle:
f108-pro light spectrum

# Rolling rainbow, reversed direction:
f108-pro light rolling 5 3 colorful -d 1

# Turn off backlight:
f108-pro off
```

### Brightness

```bash
# Set brightness to max (defaults to static white):
f108-pro brightness 5

# Set brightness with a specific mode:
f108-pro brightness 3 -m breath
```

### LCD Clock Sync

```bash
f108-pro clock
```

### Per-Key RGB

```bash
# Set all keys to cyan:
f108-pro perkey --all 0,255,255

# Set all keys dim blue, WASD green:
f108-pro perkey --all 0,0,50 w 0 255 0 a 0 255 0 s 0 255 0 d 0 255 0

# Load from YAML:
f108-pro perkey layout.yaml

# List all key names:
f108-pro keys
```

YAML format:

```yaml
all: [0, 0, 50]
brightness: 3
keys:
  w: [0, 255, 0]
  a: [0, 255, 0]
  s: [0, 255, 0]
  d: [0, 255, 0]
  esc: [255, 0, 0]
```

See [examples/wasd-green.yaml](examples/wasd-green.yaml).

### LCD Image Upload - Use with Caution
Upload a raw image to the keyboard's 240x135 LCD screen. Use `cmd/mkimage` to
generate the image file from a GIF or solid color. A confirmation prompt warns
that uploading can permanently corrupt the keyboard's built-in menu graphics.
Images exceeding 141 frames are blocked.

```bash
f108-pro lcd image.bin
```

**Warning**: uploading more than 141 frames corrupted the screen menus on my
keyboard. See [ai-docs/lcd-upload-investigation](ai-docs/lcd-upload-investigation.md)

### Key Remapping

```bash
# Swap CapsLock and Left Ctrl:
f108-pro remap capslock lctrl

# Media keys:
f108-pro remap f1 media:play f2 media:volup f3 media:voldown

# Mouse actions:
f108-pro remap pause mouse:lclick

# Key combos:
f108-pro remap capslock combo:ctrl+c grave combo:win+d

# FN layer:
f108-pro remap --fn f1 media:play

# Load from YAML:
f108-pro remap remap.yaml

# Clear all remaps:
f108-pro remap --reset
```

Target types:

* Key name: any name from `f108-pro keys` (simple key swap).
* `media:<action>`: play, stop, prev, next, volup, voldown, mute.
* `mouse:<action>`: lclick, rclick, mclick, scrollup, scrolldn.
* `combo:<mods+key>`: modifiers are ctrl, shift, alt, win, rctrl, rshift,
  ralt, rwin.

YAML format:

```yaml
layer: normal
keys:
  capslock: lctrl
  f1: media:play
  pause: mouse:lclick
  f4: combo:ctrl+c
```

See [examples/capslock-ctrl.yaml](examples/capslock-ctrl.yaml).

### List Modes

```bash
f108-pro modes
```

### Lighting Modes

| ID  | Name       | Color   | Speed | Direction |
| --- | ---------- | ------- | ----- | --------- |
| 0   | Off        | -       | -     | -         |
| 1   | Static     | Yes     | -     | -         |
| 2   | SingleOn   | Yes     | Yes   | -         |
| 3   | SingleOff  | Yes     | Yes   | -         |
| 4   | Glittering | Yes     | Yes   | -         |
| 5   | Falling    | Yes     | Yes   | -         |
| 6   | Colourful  | Rainbow | Yes   | -         |
| 7   | Breath     | Yes     | Yes   | -         |
| 8   | Spectrum   | Rainbow | Yes   | -         |
| 9   | Outward    | Yes     | Yes   | -         |
| 10  | Scrolling  | Yes     | Yes   | L/R       |
| 11  | Rolling    | Yes     | Yes   | U/D       |
| 12  | Rotating   | Yes     | Yes   | U/D       |
| 13  | Explode    | Yes     | Yes   | -         |
| 14  | Launch     | Yes     | Yes   | -         |
| 15  | Ripples    | Yes     | Yes   | -         |
| 16  | Flowing    | Yes     | Yes   | U/D       |
| 17  | Pulsating  | Yes     | Yes   | -         |
| 18  | Tilt       | Yes     | Yes   | U/D       |
| 19  | Shuttle    | Yes     | Yes   | -         |

## Additional Tools

### mkimage
Generates raw LCD image buffer files from solid colors or GIF files.

```bash
go build ./cmd/mkimage/

# Solid red:
mkimage -o output.bin -r 255 -g 0 -b 0

# From GIF:
mkimage -o output.bin -gif input.gif

# Hex color:
mkimage -o output.bin -hex FF00FF
```

### dumphid
Dumps HID report descriptors from all keyboard USB interfaces.

```bash
go build ./cmd/dumphid/
```

### hidprobe (Windows only)
Enumerates HID devices using the Windows native HID API.

```bash
go build ./cmd/hidprobe/
```

## Light Zones
The F108 Pro has three independent light zones:

| Zone           | Control              | Notes                          |
| -------------- | -------------------- | ------------------------------ |
| Main backlight | Software (this tool) | 20 modes, fully controllable   |
| Light bar      | FN key combos only   | Strip above arrow keys         |
| Sidelights     | FN key combos only   | Strips under the keyboard case |

The light bar and sidelights are controlled by the keyboard's firmware only
via FN key combos (e.g., FN + Left Shift to toggle light bar effects).

## Technical Details

* VID/PID: `0x0C45` / `0x800A` (Sonix/Microdia)
* USB Interface 3: Feature reports (64 bytes) for commands
* USB Interface 2: Interrupt endpoints for LCD data pages (4096 bytes)
* Report size: 64 bytes (no report ID prefix)
* Protocol: USB HID SET_REPORT / GET_REPORT control transfers (interface 3),
  interrupt transfers (interface 2 for LCD data)
* Readback required after begin/init/apply commands
* Command delay: 35ms between packets
* Linux backend: `gousb` (libusb wrapper, requires cgo)
* Windows backend: native HID API (`hid.dll` + `setupapi.dll`, pure Go, no
  cgo)

See [ai-ddocs/hid-protocol.md](ai-docs/hid-protocol.md) for full protocol
documentation.

## Hardware Tests
The `pkg/aula/` package includes hardware integration tests that exercise
lighting modes, per-key RGB, and clock sync on a connected keyboard. **Tests are
skipped automatically if the keyboard is not connected.**

```bash
sudo go test -v -count=1 ./pkg/aula/ -run TestHW
```

### Running Hardware Tests on Windows from WSL2
If the keyboard is connected to Windows, `go test` in WSL2 skips the hardware
tests. Cross-compile a test binary and run it on the Windows side via
PowerShell:

```bash
# Cross-compile the test binary:
CGO_ENABLED=0 GOOS=windows go test -c -o hardware_test.exe ./pkg/aula/

# Run from WSL2 via PowerShell (no copying needed):
powershell.exe -Command "& './hardware_test.exe' '-test.v' '-test.run=TestHW' '-test.count=1'"

# Clean up:
rm hardware_test.exe
```

PowerShell can run binaries directly from the WSL filesystem
(`\\wsl.localhost\...`). Use `=` syntax for flags (e.g., `-test.run=TestHW`)
because PowerShell splits space-separated flag arguments.
