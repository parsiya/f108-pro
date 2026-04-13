# Ghidra Functions Index

## Naming Convention
Functions not yet renamed in Ghidra use the auto-generated `FUN_XXXXXXXX`
format. This file maps them to their purpose.

## HID Layer

| Function       | Purpose                   | Notes                                                                                                                                        |
| -------------- | ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `FUN_004509e0` | HID DLL loader            | Loads hid.dll, resolves all HidD_*/HidP_* function pointers. Returns 0 on success, -1 on failure.                                            |
| `FUN_00450af0` | Device enumerator         | Enumerates all HID devices using SetupDi API, builds linked list of device info structs. Uses GUID `{4D1E55B2-F16F-11CF-88CB-001111000030}`. |
| `FUN_00451330` | HidD_SetFeature wrapper   | Sends 0x41-byte feature report. Returns 0x41 on success, -1 on error.                                                                        |
| `FUN_004513d0` | DeviceIoControl read-back | Uses IOCTL 0xb0192 (IOCTL_HID_GET_FEATURE) to read response from keyboard into same buffer.                                                  |
| `FUN_00450960` | Error handler             | Called on DeviceIoControl failure.                                                                                                           |

## Communication Layer

| Function       | Purpose               | Notes                                                                                                                            |
| -------------- | --------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `FUN_0044edc0` | Primary send function | Send with configurable delay. Handles single-packet (<0x42 bytes) and multi-packet modes. param_3 controls read-back.            |
| `FUN_0044efb0` | Timed send function   | Similar to primary but uses Sleep(5) and 100ms timeout for multi-packet.                                                         |
| `FUN_0044f290` | Another send variant  | Calls FUN_00451330. Needs further analysis.                                                                                      |
| `FUN_0044fe20` | Find device handle    | Searches device list for matching device by interface number. Checks 3 priority levels (0, 1, 2). Returns device handle pointer. |
| `FUN_0044e830` | Register device mode  | Called during config parsing to register VID/PID/name/type/mode combos into a global device registry.                            |

## Config & Initialization

| Function       | Purpose            | Notes                                                                                    |
| -------------- | ------------------ | ---------------------------------------------------------------------------------------- |
| `FUN_0041c070` | Config parser      | Parses config.xml: device.info, modes, language, software info. Heart of initialization. |
| `FUN_00456eb0` | XML load           | Part of XML parsing pipeline.                                                            |
| `FUN_00457ed0` | XML validate       | Validates parsed XML.                                                                    |
| `FUN_004572c0` | XML find element   | Finds XML element by name.                                                               |
| `FUN_0040bde0` | XML get attribute  | Gets attribute value from current XML element.                                           |
| `FUN_00457e30` | XML enter children | Moves XML cursor into child elements.                                                    |
| `FUN_00457e70` | XML exit children  | Moves XML cursor back to parent.                                                         |
| `FUN_0040bd00` | XML parser init    | Initializes XML parser state.                                                            |
| `FUN_00456ce0` | XML parser cleanup | Cleans up XML parser.                                                                    |

## Command Builders

| Function       | Purpose                     | Notes                                                                                                                    |
| -------------- | --------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `FUN_00414230` | Function settings sender    | Sends FN switch, sleep time, key response time. Commands: 04 18, 04 17, 00 01, 04 02.                                    |
| `FUN_004185e0` | Key remap sender            | Sends key remapping data. Builds 0x240-byte payload with 4 bytes per key. Commands: 04 18, 04 11/27, data, 04 02, 04 F0. |
| `FUN_0042b040` | Lighting mode sender (USB)  | Sends lighting effect mode, color, brightness, speed, direction. Commands: 04 18, 04 13, data, 04 02, 04 F0. ANALYZED.   |
| `FUN_0042b1e0` | Lighting mode sender (2.4G) | Wireless variant. Single packet with command 05 10. ANALYZED.                                                            |
| `FUN_0044b910` | Per-key RGB sender          | Sends per-key RGB data. Commands: 04 18, 04 23, data (0xC0 mono or 0x240 RGB), 04 02, 04 F0. ANALYZED.                   |
| `FUN_0044b800` | Sidelight sender            | Sends sidelight config. Commands: 04 18, 04 13, 00 80, 04 02, 04 F0. ANALYZED.                                           |
| `FUN_00433fd0` | LED layer sender            | Sends LED layer mode. Commands: 04 18, 04 13, data, 04 02, 04 F0. ANALYZED.                                              |
| `FUN_00422b50` | LCD image uploader          | Uploads RGB565 images to 240x135 LCD. Commands: 04 18, 04 72, pages, 04 02. ANALYZED.                                    |
| `FUN_00423b10` | LCD clock sync              | Sends system datetime to keyboard LCD. Commands: 04 18, 04 28, data, 04 02. ANALYZED.                                    |
| `FUN_0042ae20` | Key color readback          | Reads per-key colors via 04 F5, updates custom light UI. NOT macro-related.                                              |
| `FUN_0042d630` | USB macro sender            | Sends all macros to keyboard (task 8, USB). See [macro-protocol.md](macro-protocol.md).                                  |
| `FUN_0042dbb0` | 2.4G macro sender           | Sends all macros to keyboard (task 8, 2.4G). See [macro-protocol.md](macro-protocol.md).                                 |

## Task Dispatch

| Function       | Purpose                  | Notes                                                                             |
| -------------- | ------------------------ | --------------------------------------------------------------------------------- |
| `FUN_00434230` | Worker thread dispatcher | CThreadBasic worker. Reads task queue at offset 0x764, dispatches by type.        |
| `FUN_004159e0` | Task enqueue             | Inserts task into linked list at offset 0x764. Task type stored at node offset 8. |
| `FUN_00408520` | Light data DB reader     | Reads t_light_data row by profile+mode. Returns struct with all lighting fields.  |
| `FUN_004081d0` | Light data DB writer     | Updates t_light_data row. Called by UI handlers before queueing task.             |

## UI / MUI Framework

| Function       | Purpose             | Notes                                                    |
| -------------- | ------------------- | -------------------------------------------------------- |
| `FUN_00413440` | Get device context  | Returns the global device communication context pointer. |
| `FUN_00405aa0` | Get app instance    | Returns application singleton pointer.                   |
| `FUN_004201c0` | Get selected device | Returns currently selected device ID.                    |
| `FUN_0040f470` | Get data manager    | Returns data manager instance (for DB queries).          |

## Utility Functions

| Function       | Purpose             | Notes                                                               |
| -------------- | ------------------- | ------------------------------------------------------------------- |
| `FUN_00451b90` | Key code converter  | Converts between key code formats. Used in key remapping.           |
| `FUN_004194e0` | Media key lookup    | Looks up consumer control codes. Used for multimedia key remapping. |
| `FUN_00417060` | Device info cleanup | Frees device info structure.                                        |
| `FUN_0040c9e0` | String concat       | Concatenates wide strings.                                          |
| `FUN_00405620` | Format string       | Printf-like string formatting.                                      |
| `FUN_004050b0` | String assign       | Assigns wide string to a string object.                             |
| `FUN_00404cd0` | String copy         | Copies string object.                                               |
| `FUN_0040d070` | Memory free         | Frees allocated memory.                                             |
| `FUN_00405160` | Vector cleanup      | Frees vector contents.                                              |
| `FUN_004205c0` | Vector grow         | Grows vector capacity.                                              |
| `FUN_00420790` | Vector insert       | Inserts element into vector.                                        |
