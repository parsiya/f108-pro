# Macro Protocol
Parsia's Note: We documented the macro protocol but never implemented or tried
it. After the LCD screen fiasco I didn't want to do anything complex with the
keyboard.

## Overview
The Aula F108 Pro supports recording keyboard and mouse macros via the Windows
software. Macros are stored in a local SQLite database and pushed to the
keyboard over HID. A key can be remapped to trigger a macro using remap type
`0x06` (see [key-remap-protocol.md](key-remap-protocol.md)).

The macro system has three parts:

* Recording: Windows keyboard/mouse hooks capture events with timestamps.
* Storage: macros and their events are saved to SQLite tables.
* Transfer: the full macro table is serialized into a binary buffer and
  sent to the keyboard via HID feature reports.

## Source Functions (Ghidra)

| Function     | Address      | Purpose                            |
| ------------ | ------------ | ---------------------------------- |
| FUN_0043c3b0 | `0x0043c3b0` | Macro UI initialization            |
| FUN_00430430 | `0x00430430` | Save macro records from UI to DB   |
| FUN_0042cfc0 | `0x0042cfc0` | Save macro metadata + queue task 8 |
| FUN_0042d630 | `0x0042d630` | USB macro sender (task 8)          |
| FUN_0042dbb0 | `0x0042dbb0` | 2.4G macro sender (task 8)         |
| FUN_0040a9a0 | `0x0040a9a0` | Load all macros from t_macro_data  |
| FUN_0040aff0 | `0x0040aff0` | Load events from t_macrorecord     |
| FUN_0040a610 | `0x0040a610` | Update macro metadata in DB        |
| FUN_0040abe0 | `0x0040abe0` | Insert single macro record to DB   |
| FUN_0040adb0 | `0x0040adb0` | Batch insert macro records to DB   |
| FUN_0040b320 | `0x0040b320` | Count total macro records in DB    |
| FUN_00451460 | `0x00451460` | Windows VK code -> HID scancode    |

## Database Schema

### t_macro_data
Macro definitions. Each macro has a unique ID, a name, and playback settings.

```sql
CREATE TABLE IF NOT EXISTS t_macro_data(
  macro_id INTEGER PRIMARY KEY AUTOINCREMENT,
  play_times INTEGER,
  delay_type INTEGER,
  delay_time INTEGER,
  link_count INTEGER,
  name TEXT
)
```

| Column     | Description                                   |
| ---------- | --------------------------------------------- |
| macro_id   | Auto-increment primary key                    |
| play_times | Repeat count (1-99)                           |
| delay_type | 1 = recorded delay, 2 = no delay, 3 = custom  |
| delay_time | Custom delay in ms (used when delay_type = 3) |
| link_count | Number of recorded events                     |
| name       | Macro name (default: "New Macro")             |

### t_macrorecord
Individual events within a macro. Each row is one key press, key release, mouse
button press, mouse button release, or delay.

```sql
CREATE TABLE IF NOT EXISTS t_macrorecord(
  record_id INTEGER PRIMARY KEY AUTOINCREMENT,
  macro_id INTEGER,
  type INTEGER,
  record_index INTEGER,
  desc TEXT,
  value INTEGER,
  delay_time INTEGER
)
```

| Column       | Description                                     |
| ------------ | ----------------------------------------------- |
| record_id    | Auto-increment primary key                      |
| macro_id     | Foreign key to t_macro_data                     |
| type         | Event type (see below)                          |
| record_index | Order within the macro (0-based)                |
| desc         | Display text (e.g., key name)                   |
| value        | Windows VK code (keys) or button number (mouse) |
| delay_time   | Delay before this event in ms                   |

Event types:

| Type | Meaning           |
| ---- | ----------------- |
| 2    | Key down          |
| 3    | Key up            |
| 4    | Mouse button down |
| 5    | Mouse button up   |

Mouse button values: 1 = left, 2 = right, 3 = middle.

### t_key_macro_data
Binds keys to macros (used in the remap table).

```sql
CREATE TABLE IF NOT EXISTS t_key_macro_data(
  key_id INTEGER PRIMARY KEY AUTOINCREMENT,
  profile INTEGER, fn_layer INTEGER, key_code INTEGER,
  layout_value INTEGER, layout_desc TEXT,
  macro_type INTEGER, macro_value INTEGER,
  macro_value2 INTEGER, macro_value3 INTEGER,
  macro_desc TEXT
)
```

## Recording Workflow
The software uses `SetWindowsHookExW` (loaded dynamically from user32.dll) to
install low-level keyboard and mouse hooks during recording. The UI provides
three event filter modes via radio buttons (`MACRO_EVENT` group):

* Keyboard only (param_1 + 0x568)
* Mouse only (param_1 + 0x56c)
* Keyboard + Mouse (param_1 + 0x570)

The recording flow:

1. User selects a macro from the list (MListBox at param_1 + 0x55c).
2. User clicks Record -> hooks are installed.
3. Key/mouse events are captured and added to the CRecordList UI control.
4. User clicks Stop -> hooks are removed.
5. User configures playback settings:
   * Play times (1-99, edit control at param_1 + 0x5a4).
   * Delay mode (`DELAY_TIME` radio group):
     * Record actual delays (delay_type = 1).
     * No delay (delay_type = 2).
     * Custom delay in ms (delay_type = 3, edit control at param_1 + 0x5b4,
       default 10ms).
6. User clicks Apply:
   * FUN_00430430 reads events from CRecordList::GetRecordItems() and
     batch-inserts them into t_macrorecord via FUN_0040adb0.
   * FUN_0042cfc0 updates t_macro_data (play_times, delay_type, delay_time)
     via FUN_0040a610 and queues task 8 (macro send).

## HID Protocol

### USB Protocol (FUN_0042d630)

#### Size Limit
Before sending, the function checks:
`(macro_count + total_event_count) * 8 <= 2700 (0xA8C)`

If the total exceeds this limit, the macro data is not sent.

#### Protocol Sequence

| Step | Packet                | Readback | Purpose            |
| ---- | --------------------- | -------- | ------------------ |
| 1    | `04 19`               | Yes      | Macro init         |
| 2    | `04 15` (byte[8] = N) | Yes      | Data header        |
| 3    | Data buffer (N × 64B) | Yes      | Macro data payload |
| 4    | `04 02`               | Yes      | Apply              |

30ms sleep between steps 2-3 and 3-4.

The `04 15` header packet has the data packet count at byte 8. The data buffer
is padded to a multiple of 64 bytes with a `0x55 0xAA` trailer at the last 2
bytes.

### 2.4G Wireless Protocol (FUN_0042dbb0)

#### Size Limit
`(macro_count + total_event_count) * 8 <= 3600 (0xE10)`

#### Protocol Sequence
The 2.4G variant does not use `04 19`/`04 15`. Instead, data is split into
28-byte (0x1C) chunks and sent via the 2.4G command interface (FUN_0044f4e0):

```
For each chunk:
  Byte 0: 0x00
  Byte 1: 0x09 (subcommand)
  Byte 2: chunk size (0x1C, or remaining bytes for the last chunk)
  Byte 3: chunk index (0-based)
  Bytes 4-31: 28 bytes of macro data
  Bytes 32-64: 0x00 (padding)
```

2ms sleep between chunks. The `0x55 0xAA` trailer is appended to the data before
chunking.

## Data Buffer Format
Both USB and 2.4G use the same binary buffer layout (max 3584 bytes / 0xE00).

### Header Area (bytes 0-399)
100 macro slots, 4 bytes each. Supports up to 100 macros.

```
For each macro index i (0-based):
  Byte i*4+0: data_offset low byte (16-bit LE)
  Byte i*4+1: data_offset high byte
  Byte i*4+2: 0x00
  Byte i*4+3: 0x00

Empty slot: 0xFF 0xFF 0xFF 0xFF (no recorded events)
```

The `data_offset` is the byte offset from the buffer start to the macro's data
block.

### Data Area (bytes 400+)
Each macro's data block starts at its `data_offset`:

```
offset+0: event_count low byte (16-bit LE)
offset+1: event_count high byte
offset+2 through offset+7: padding (zeros, 6 bytes)
offset+8: event 0 (4 bytes)
offset+12: event 1 (4 bytes)
...
```

Total block size: 8 + (event_count × 4) bytes.

### Event Format (4 bytes per event)

```
Byte 0: delay_low (delay events only, 0 for key/mouse)
Byte 1: delay_high (delay events only, 0 for key/mouse)
Byte 2: code (HID scancode or mouse button code)
Byte 3: action flag
```

#### Action Flags

| Flag   | Meaning           | Byte 2        |
| ------ | ----------------- | ------------- |
| `0xB0` | Key down          | HID scancode  |
| `0x30` | Key up            | HID scancode  |
| `0x90` | Mouse button down | Button code   |
| `0x10` | Mouse button up   | Button code   |
| `0x50` | Delay             | 0x00 (unused) |

Flag bit pattern:

* Bit 7 (`0x80`): press/down (set for down events).
* Bit 5 (`0x20`): keyboard event.
* Bit 4 (`0x10`): base event marker.
* Bit 6 (`0x40`): delay event.

#### Mouse Button Codes (Wire Format)
The mouse button codes on the wire differ from the database values:

| DB value | Wire code | Button |
| -------- | --------- | ------ |
| 1        | `0x01`    | Left   |
| 2        | `0x04`    | Right  |
| 3        | `0x02`    | Middle |

#### Key Code Conversion
The database stores Windows Virtual Key codes. FUN_00451460 at `0x00451460`
converts them to USB HID scancodes for transmission. Examples:

| VK Code | HID Scancode | Key        |
| ------- | ------------ | ---------- |
| 0x41    | 0x04         | A          |
| 0x42    | 0x05         | B          |
| 0x0D    | 0x28         | Enter      |
| 0x1B    | 0x29         | Escape     |
| 0x08    | 0x2A         | Backspace  |
| 0x09    | 0x2B         | Tab        |
| 0x20    | 0x2C         | Space      |
| 0x70    | 0x3A         | F1         |
| 0xA2    | 0xE0         | Left Ctrl  |
| 0xA0    | 0xE1         | Left Shift |

#### Delay Events
For delay events (action flag `0x50`):

```
Byte 0: delay_ms low byte
Byte 1: delay_ms high byte (16-bit LE, minimum 10ms)
Byte 2: 0x00
Byte 3: 0x50
```

Delays below 10ms are clamped to 10ms.

### Buffer Example
Two macros: macro 0 has 2 events (A down, A up), macro 1 is empty.

```
Header:
  00: 90 01 00 00    Macro 0: data at offset 0x0190 (400)
  04: FF FF FF FF    Macro 1: no data

Data (at offset 400):
  190: 02 00 00 00 00 00 00 00    event_count = 2, padding
  198: 00 00 04 B0                event 0: key down, HID 0x04 (A)
  19C: 00 00 04 30                event 1: key up, HID 0x04 (A)

Trailer (at end of padded buffer):
  ...: 55 AA
```

## Key Binding
To bind a key to a macro, set the key's remap slot to type `0x06`:

```
Byte 0: 0x06 (macro execution)
Byte 1: macro index (0-based position in the macro list)
Byte 2: loop count parameter
Byte 3: additional parameter
```

The macro index is the position in the ordered macro list, looked up via
FUN_0040a9a0 which queries `SELECT * FROM t_macro_data`. The source data's
`iVar14 + 0x1c` field is matched against macro IDs to find the position.

See [key-remap-protocol.md](key-remap-protocol.md) "Type 0x06: Macro Execution"
for details.

## Task Dispatch
Macro sending is task type 8 in the worker thread dispatcher (FUN_00434230). The
connection mode at `this+0x4d8` determines which sender is called:

* `0x4d8 == 0` (USB): FUN_0042d630
* `0x4d8 == 2` (2.4G): FUN_0042dbb0
