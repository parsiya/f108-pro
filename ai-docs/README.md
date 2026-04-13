# Aula F108 Pro Reverse Engineering
This page had the original prompt and was edited by (A)I as we moved
through the process.

**These are mostly AI generated and then reviewed/edited by Parsia.** I've
marked my notes in the documents.

## Project Goal
Reverse engineer the Aula/Epomakler F108 Pro keyboard configuration software to
create an open-source replacement in Go that can communicate with the keyboard
over USB HID.

### Working Features

* 20 built-in lighting modes with speed, brightness, color, direction controls
* Per-key RGB lighting
* LCD clock sync
* LCD image and GIF animation upload (with 141-frame safety limit)
* Key remapping (normal layer and FN layer)
* Hardware integration tests

### Not Yet Implemented

* Macro recording and sending (protocol fully documented but not implemented)
* Realtime streaming mode (`04 20`), see [Punkster81/AULA-F108-Driver].

[pu]: https://github.com/Punkster81/AULA-F108-Driver?tab=readme-ov-file#-layers

## Documentation Index

* [device-info.md](device-info.md) - USB VID/PID, device identifiers, config.xml analysis
* [hid-protocol.md](hid-protocol.md) - HID communication protocol: packet format, commands, IOCTL codes
* [key-map.md](key-map.md) - Keyboard layout, key codes, and key index mapping
* [key-remap-protocol.md](key-remap-protocol.md) - Key remapping protocol for normal and FN layers
* [macro-protocol.md](macro-protocol.md) - Macro recording, storage, and HID transfer protocol
* [ghidra-functions.md](ghidra-functions.md) - Index of analyzed Ghidra functions with addresses and descriptions
* [firmware-update.md](firmware-update.md) - Firmware update protocol (Sonix ISP flasher)
* [lcd-upload-investigation.md](lcd-upload-investigation.md) - LCD upload investigation and corruption incident
* [sidelight-investigation.md](sidelight-investigation.md) - Sidelight LED investigation

## Environment

* Keyboard: Aula F108 Pro (full-size, RGB, tri-mode mechanical keyboard)
* MCU: Sonix SN32F2xx family (VID `0x0C45`, PID `0x800A`)
