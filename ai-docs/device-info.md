# Device Information

## Keyboard
Aula F108 Pro (AULA F108Pro 三模机械键盘) - full-size 108-key RGB mechanical
keyboard with tri-mode connectivity (USB, 2.4G wireless, possibly Bluetooth).

Manufacturer: 东莞市索艾电子科技有限公司 (Dongguan Suoai Electronic Technology Co.)

## MCU Identification
VID `0C45` is Sonix/Microdia. The tool supports multiple Sonix chip families:

* SN32F22x, SN32F23x, SN32F24x, SN32F24xB, SN32F24xC
* SN32F26x, SN32F280, SN32F290
* SN8F series (SN8F5280, SN8F2267F, SN8F22E8xB, etc.)

The filename mentions `HFD80CP100` which is likely the PCB/board identifier.

Note from Parsia: Searching for `sonix HFD80CP100` returns a few other keyboards
such as `Ajazz AK820 Pro`.

## USB Identifiers

### USB Wired Mode
* VID: `0x0C45` (Microdia / Sonix Technology)
* PID: `0x800A`
* Product name: "AULA F108Pro"
* HID interface filter: `VID_0C45&PID_800A&MI_00`
* Uses interface 0 (`MI_00`)

### 2.4G Wireless Mode (TYPE-A dongle)
Apparently it's common for Chinese peripherals to spoof Apple hardware VID/PID.
`05AC:024F` is [Apple's Aluminium Keyboard - ANSI][vi].

[vi]: https://devicehunt.com/view/type/usb/vendor/05AC/device/024F

* VID: `0x05AC` (Apple Inc.)
* PID: `0x024F`
* Product name: "F108Pro Dongle"
* HID interface filter: `VID_05AC&PID_024F&MI_03`
* Uses interface 3 (`MI_03`)

## Config Source
All of the above comes from `original-files/config.xml`:

```xml
<device.info>
  <keyboard name="AULA F108Pro 三模机械键盘" device_type="101"
            device_info="rgb-keyboard"
            img_connected="home_keyboard_connected.png"
            img_disconnected="home_keyboard_disconnected.png">
    <mode value="0" desc="USB" vid="0C45" pid="800A"
          product_name="AULA F108Pro"
          hid_interface="VID_0C45&PID_800A&MI_00"/>
    <mode value="2" desc="2.4G TYPE-A" vid="05AC" pid="024F"
          product_name="F108Pro Dongle"
          hid_interface="VID_05AC&PID_024F&MI_03"/>
  </keyboard>
</device.info>
```

## Connectivity & Configuration Modes
The keyboard supports three connection modes:

| Mode          | Value | Protocol                                                  | Config Support |
| ------------- | ----- | --------------------------------------------------------- | -------------- |
| USB Wired     | 0     | HID Feature Reports (multi-step: 04 18/04 13/04 02/04 F0) | Full           |
| 2.4G Wireless | 2     | HID single-packet (command 05 10)                         | Partial        |
| Bluetooth     | -     | Not present in config.xml                                 | Not supported  |

The software explicitly warns in the language file (string #62):

> keyboard settings are not supported in Bluetooth mode for now. Please use the
> USB or 2.4 GHz connection for the keyboard setting.

The 2.4G wireless mode uses a simplified single-packet protocol (command
`05 10`) instead of the multi-step begin/data/apply flow used over USB. See
[hid-protocol.md](hid-protocol.md) for the full protocol documentation.

This project focuses on wired USB configuration as the primary target.

## Device Type
`device_type="101"` and `device_info="rgb-keyboard"` - the layout file is
`layouts/rgb-keyboard.xml`.

## MCU and Firmware
VID `0x0C45` is Sonix Technology (often labeled as Microdia). The firmware runs
on a Sonix SN32F2xx chip. The config references firmware version `120`. See
[firmware-update.md](firmware-update.md) for the update protocol and version
history.
