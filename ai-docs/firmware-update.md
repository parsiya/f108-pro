# Firmware Update Protocol
Note from Parsia: I analyzed the firmware update with Ghidra MCP to see what it
does and if it has the data to flash the screen. But it looks like it didn't.
Here's the analysis.

The Windows Aula software does **not** flash firmware itself. It downloads and
launches a separate `FirmwareUpdateTool.exe`. To understand the actual USB
flashing protocol, you would need to reverse that tool or capture USB traffic
during a firmware update.

## Windows Software Versions (as of 2026-04-12)

1. `v1.0.0.1`: https://aulagear.com/blogs/software/aula-f108-pro-driver
2. `v1.0.0.3`: https://aulakeyboard.com/download/f108-pro-drive/

## How the Windows Software Handles Updates
There are two paths: remote (download from server) and local (bundled with the
installer).

### Remote Firmware Update — `FUN_00435540` (task type 21)
Triggered when `config.xml` has a non-empty `<upgrade url="..."/>`. This is not
the case for F108 Pro software versions `1.0.0.1` and `1.0.0.3`.

1. Builds path `%appdata%\temp\FirmwareUpdateTool.zip`
2. Deletes any existing copy via `DeleteFileW`
3. Downloads the zip via `FUN_0045a430` (WinINet HTTP downloader, user agent
   `IIC2.0/PC 5.5.0000`, reads in 10 KB chunks, posts progress to UI via
   `PostMessageW`)
4. Checks the file exists with `PathFileExistsW`
5. Extracts with `FUN_004637e0` (zip extraction)
6. Deletes the zip
7. Launches `FirmwareUpdateTool.exe` via `ShellExecuteW("open")`
8. Calls `exit(0)` — kills the main application

### Local Firmware Update — `FUN_00415200`
Checks for firmware files bundled in the `firmware\` directory next to the
application. Software version `1.0.0.3` has such a firmware. The firmware
directory for version `1.0.0.1` is empty.

1. Reads a version string from a local setting and compares it (via `_wtoi`)
   to the device firmware version (from object offset `0x7e4`)
2. If local version > device version: builds path
   `<appdir>\firmware\<filename>` using the filename from offset `0x7e0`
3. Checks `PathFileExistsW` — if the file exists:
    * Calls `FUN_0044e920` (likely closes HID handles to release the device)
    * Launches the firmware file via `CreateProcessW`
    * Stores the process ID at offset `0x5280c`
    * Hides the main window with `ShowWindow(hwnd, 0)`
4. If versions match: shows a "firmware is up to date" dialog

### Software Self-Update — `FUN_004352b0` (task type 20)
Same pattern as the remote firmware update but for the application itself:

1. Downloads `Keyboard-Software-Setup.zip` to `%appdata%\temp\`
2. Extracts it
3. Launches `Keyboard-Software-Setup.exe` via `ShellExecuteW`
4. Calls `exit(0)`

## Current State of the F108 Pro

### Software Version 1.0.0.1 (original)

* `<upgrade url=""/>` in `config.xml` is **empty**
* `<firmware version="120" file="" url="" />` in `rgb-keyboard.xml` — version
  120 (1.20) but no file or URL
* `firmware/` directory is empty
* No URLs of any kind exist anywhere in the original files

### Software Version 1.0.0.3 (newer)
Key differences from 1.0.0.1:

* `<firmware version="107" file="索艾SI-2688 1.14横屏三模RGB_HFD80CP100_V1.07_20250928_0xEC23(默认中文) .exe" url="" />`
* The `firmware/` directory contains the actual firmware update tool
* Software name changed from `AULA F108Pro 三模机械键盘 Driver` to `AULA F108Pro Driver`
* Copyright updated from 2024 to 2025

The version numbers are confusing: the old software lists firmware version
`120` and the new one ships firmware `107` (V1.07).

Note from Parsia: I am not sure the analysis below is correct. When I started
`1.0.0.3` it showed a prompt asking permission to update the firmware.

The local update check in `FUN_00415200` compares `local_version > device_version`,
so `DeviceDriver.exe` would not auto-trigger the update since 107 < 120. The
firmware tool runs independently though.

## Firmware Update Tool Analysis
The firmware tool (`索艾SI-2688...exe`) is a
**Sonix ISP (In-System Programming) flasher**. It uses the standard Sonix
bootloader protocol over USB HID.

### MCU Identification
VID `0C45` is Sonix/Microdia. The tool supports multiple Sonix chip families:

* SN32F22x, SN32F23x, SN32F24x, SN32F24xB, SN32F24xC
* SN32F26x, SN32F280, SN32F290
* SN8F series (SN8F5280, SN8F2267F, SN8F22E8xB, etc.)

The filename mentions `HFD80CP100` which is likely the PCB/board identifier.

Note from Parsia: Searching for `sonix HFD80CP100` returns a few other keyboards
such as `Ajazz AK820 Pro`.

### ISP Protocol Commands
The tool uses the standard Sonix ISP command set via `HidD_SetFeature` /
`HidD_GetFeature`:

| Command | Code (SN32) | Code (SN8) | Description                |
| ------- | ----------- | ---------- | -------------------------- |
| 0x20    | 0x20        | -          | Enter ISP Mode             |
| 0x21    | 0x21        | 0x01       | Get Firmware Version       |
| 0x22    | 0x22        | 0x02       | Compare ISP Password       |
| 0x23    | 0x23        | 0x03       | Set Encryption Algorithm   |
| 0x24    | 0x24        | 0x04       | Enable Erase               |
| 0x25    | 0x25        | 0x05       | Enable Program             |
| 0x26    | 0x26        | 0x06       | Get Check Sum              |
| 0x27    | 0x27        | 0x07       | Return User Mode           |
| 0x28    | 0x28        | 0x08       | Set Code Option            |
| 0x29    | 0x29        | 0x09       | Get Code Option            |
| 0x2A    | 0x2A        | 0x0B       | Enable Check Data / Verify |

### Flash Sequence
Based on the strings, the flash procedure is:

1. Find HID device by VID/PID
2. Enter ISP Mode (0x20)
3. Get Firmware Version (0x21)
4. Compare ISP Password (0x22)
5. Enable Erase (0x24) -> "Chip Erase OK!!"
6. Enable Program (0x25) -> send firmware data
7. Get Check Sum (0x26) -> verify checksum (from `.SN8` hex file)
8. Return User Mode (0x27)

The firmware file format is `.SN8` (a variant of Intel HEX).

### Two Separate Flash Chips
The firmware update only touches the MCU's internal flash. The LCD menu
graphics live on a separate SPI NOR flash chip that the firmware updater does
not access. This means firmware updates cannot restore corrupted LCD graphics.
See [lcd-upload-investigation.md](lcd-upload-investigation.md) for details.

The ISP protocol is the same one documented by the QMK community for Sonix
keyboards (SonixQMK).

Note from Parsia: Does it mean we might be able to make a QMK compatible version
of the firmware? :)
