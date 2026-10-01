# 2.4G Wireless Key Remap Protocol

## Status

Verified on hardware (one F108 Pro, Linux, dongle `05AC:024F`): single-key
remap on the normal and FN layers, and per-key restore. Normal layer: `02 00
<HID usage>`, or `02 <modifier bit> 00` for modifier keys. FN layer: blank slot
`00 00 00 00`.

Not verified: persistence after power cycle, behavior after switching to
USB-C, and restoring all keys at once.
## Source Functions

| Function     | Address      | Purpose                                              |
| ------------ | ------------ | ---------------------------------------------------- |
| FUN_004348a0 | `0x004348a0` | Dispatcher. `this+0x784 == 2` (2.4G) calls FUN_004191c0 with `this+0x7fc` (0 = normal, 1 = FN) |
| FUN_004191c0 | `0x004191c0` | Builds the per-key remap packet                      |
| FUN_00418230 | `0x00418230` | Encodes an action into a 4-byte slot (same as USB)   |
| FUN_0044f4e0 | `0x0044f4e0` | Checksum, send, wait for ACK                         |
| FUN_00451110 | `0x00451110` | `WriteFile` (overlapped)                             |
| FUN_004511f0 | `0x004511f0` | `ReadFile` (overlapped)                              |
| FUN_0044fe20 | `0x0044fe20` | Selects the device handle                            |

## Transport

The dongle (`05AC:024F`) exposes five HID interfaces. Remap uses the vendor
interface with usage page `0xFF60`: 32-byte input and output reports, no
report ID. Packets are sent as plain output reports (`write()` on
`/dev/hidrawN` on Linux), not feature reports like the wired protocol. A second
vendor interface (`0xFF59`, 64-byte reports) is not used for remap.

## Packet (33 bytes written, byte 0 is the report ID 0)

| Offset | Value                                                        |
| ------ | ------------------------------------------------------------ |
| 0      | `0x00` report ID (stripped by hidraw)                        |
| 1      | `0x03` normal layer, `0x0D` FN layer                         |
| 2      | `0x04` slot length                                           |
| 3      | `key_index` (same indices as the wired protocol)             |
| 4-7    | Slot: action, param1, param2, param3                         |
| 8-31   | `0x00`                                                       |
| 32     | Checksum: sum of bytes 0-31, modulo 256                      |

One packet remaps one key and takes effect immediately; no begin/apply/finalize
steps were observed. Keys that are not sent are not touched (unlike the wired
protocol, which sends the whole table).

Example, Insert (`key_index` 116) -> `A` on the normal layer:

00 03 04 74 02 00 04 00 00 ... 00 81


## ACK

The keyboard answers with a 32-byte input report. Its first three bytes echo
bytes 1-3 of the request; the last byte is the sum of the payload. The original
driver waits 5 ms, then reads up to 20 times, 10 ms apart, and retries the write
once if it fails.

## Slot format

Same 4-byte slots as the wired protocol, see
[key-remap-protocol.md](key-remap-protocol.md). With no action assigned, the
original driver sends `02 00 <HID usage>` (the key mapped to itself).
