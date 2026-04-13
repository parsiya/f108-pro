# LCD Image Upload Investigation
This document records everything we tried (and failed) while figuring out how to
upload images to the Aula F108 Pro's 240x135 LCD screen from Linux.

## Background
The F108 Pro has a 240x135 LCD screen that displays a GIF animation and a clock.
The Windows Aula software can upload custom GIFs. We wanted to replicate this
from Linux using Go and libusb.

The keyboard exposes 4 USB HID interfaces. For normal configuration (lighting,
clock sync, key remapping), we use **interface 3** (usage page `0xFF13`) with
64-byte Feature Reports sent via USB control transfers. This works perfectly.

LCD image upload is different: the *control commands* (begin, header, apply)
still go on interface 3, but the actual *pixel data pages* go on
**interface 2** (usage page `0xFF68`).

## Phase 1: Ghidra Analysis of the Windows Software

### Identifying the Upload Flow

From the task dispatcher (`FUN_00434230`), three task types relate to the LCD:

| Task | Function       | Purpose             |
| ---- | -------------- | ------------------- |
| 15   | `FUN_00434e80` | GIF save            |
| 16   | `FUN_00422b50` | LCD screen transfer |
| 17   | `FUN_004349a0` | LCD GIF animation   |

**Task type 16** (`FUN_00422b50`) is the uploader. It:

1. Gets the current LCD view selection and frame data
2. Converts images to RGB565 (240 × 135 × 2 = 64,800 bytes/frame)
3. Builds a buffer: 256-byte header + concatenated RGB565 frames
4. Calculates page count: `ceil(total_size / 4096)`
5. Sends `04 18` (begin) as a 64-byte Feature Report on interface 3
6. Sends `04 72` (image header) with page count as a Feature Report on
   interface 3
7. Sends each 4096-byte page via `FUN_0044f1c0` on interface 2
8. Sends `04 02` (apply) as a Feature Report on interface 3

**Task type 17** (`FUN_004349a0`) prepares the GIF data. It calls:

- `MUI::LCDViewList::LoadGIFImageFile` — loads the GIF from disk
- `MUI::LCDViewList::GetDelayTimes` — extracts per-frame delay values
- `MUI::LCDViewList::GetImageRGB565Data` — converts each frame to RGB565

### GIF Files Are Standard GIFs
The files stored on disk (`original-files/gif/AULA F108Pro .../0.gif`) are **standard
GIF89a files**. They can be opened in any image viewer. The keyboard does NOT
understand the GIF format — the Windows software decodes the GIF, converts
each frame's pixels to RGB565 (16-bit, little-endian), and uploads the raw
pixel buffer. Our tool (`cmd/mkimage -gif`) replicates this conversion using
Go's `image/gif` package.

### The Data Page Sender: FUN_0044f1c0

This is where the important detail is. Decompiled:

```c
int __thiscall FUN_0044f1c0(void *this, void *param_1)
{
    // ...
    memset(&local_100c, 0, 0x1001);       // 4097 bytes
    memcpy(local_100b, param_1, 0x1000);  // copy 4096 bytes at offset 1
    // local_100c = 0x00 (report ID), local_100b = 4096 bytes of data

    DVar2 = FUN_00451110(puVar1, &local_100c, 0x1001);  // WriteFile
    if (0 < (int)DVar2) {
        DVar2 = FUN_004511f0(puVar1, &local_100c, 0x1001, 300);  // ReadFile, 300ms timeout
    }
    return CONCAT31((int3)(DVar2 >> 8), 1);
}
```

Key observations:
- Sends **4097 bytes**: 1 byte report ID (`0x00`) + 4096 bytes of data
- Uses `WriteFile` (-> `FUN_00451110`) and `ReadFile` (-> `FUN_004511f0`)
- ReadFile has a **300ms timeout**

### FUN_00451110 (WriteFile wrapper)

```c
DWORD __fastcall FUN_00451110(undefined4 *param_1, void *param_2, DWORD param_3)
{
    // If param_3 < the device's output report size, malloc a larger buffer
    // and pad with zeros
    BVar1 = WriteFile((HANDLE)*param_1, _Dst, param_3, NULL, &local_20);
    // Uses overlapped I/O, waits for completion
}
```

### FUN_004511f0 (ReadFile wrapper)

```c
uint __fastcall FUN_004511f0(undefined4 *param_1, void *param_2, uint param_3, DWORD param_4)
{
    // param_4 = timeout in ms (300)
    // Reads into internal buffer, then copies to param_2
    // Strips report ID byte when it's 0x00:
    _Src = (char *)param_1[8];
    if (*_Src == '\0') {           // report ID = 0
        uVar3 = local_c - 1;      // subtract 1 for report ID
        memcpy(param_2, _Src + 1, param_3);  // skip report ID
        return uVar3;
    }
}
```

### What This Means
On Windows, `WriteFile`/`ReadFile` on a HID device go through the **interrupt
endpoints**, not the control pipe. The Windows HID class driver translates
`WriteFile` -> interrupt OUT transfer and `ReadFile` -> interrupt IN transfer.
This is fundamentally different from a SET_REPORT control transfer.

## Phase 2: First Attempt (Control Transfers — FAILED)

### What We Tried
In `aula-go/cmd/giftest/`, we built the raw image buffer on the fly in the same
Go program that uploaded it. The program generated a solid-color 240x135
single-frame image in memory (256-byte header + 64,800 bytes of RGB565 pixel
data = 16 pages), then immediately sent it to the keyboard. We never wrote the
buffer to disk or verified its contents independently, so when the upload failed
we had two unknowns: was the image data correct, and was the USB transfer
correct?

The data pages were sent as USB control transfers (SET_REPORT) to interface 2:

```go
// sendDataPage sends 4096-byte Output Report on interface 2.
// wValue=0x0200 = Output report type (0x02) << 8 | report ID (0x00)
_, err := dev.Control(0x21, 0x09, 0x0200, 2, data)
// Then read back acknowledgment:
_, err = dev.Control(0xA1, 0x01, 0x0100, 2, rbuf)
```

### What Happened
**The keyboard firmware crashed.** All lights went off, the LCD went blank, and
the keyboard became completely unresponsive. Required unplugging and replugging
(power cycle) to recover. The keyboard was not bricked — it came back fine after
the power cycle.

### Why It Failed
SET_REPORT is a **control transfer** on endpoint 0 (the default control pipe).
The keyboard firmware was not expecting a 4096-byte payload on the control pipe
— it was expecting it on the **interrupt OUT endpoint**. Sending an unexpected
large control transfer likely caused a buffer overflow or undefined behavior in
the firmware, causing the crash.

## Phase 3: USB Descriptor Analysis

### lsusb Output for Interface 2

```
Interface Descriptor:
  bInterfaceNumber        2
  bNumEndpoints           2
  bInterfaceClass         3 Human Interface Device
  Endpoint Descriptor:
    bEndpointAddress     0x84  EP 4 IN
    bmAttributes            3
      Transfer Type            Interrupt
    wMaxPacketSize     0x0040  1x 64 bytes
    bInterval               1
  Endpoint Descriptor:
    bEndpointAddress     0x03  EP 3 OUT
    bmAttributes            3
      Transfer Type            Interrupt
    wMaxPacketSize     0x0040  1x 64 bytes
    bInterval               1
```

Interface 2 has two **interrupt endpoints**:

- **EP 3 OUT** (address `0x03`): for sending data pages
- **EP 4 IN** (address `0x84`): for receiving acknowledgments

Both have a max packet size of **64 bytes**.

### HID Report Descriptor for Interface 2

```
06 68 ff    Usage Page (0xFF68)      ← vendor-specific
09 61       Usage (0x61)
a1 01       Collection (Application)
09 62         Usage (0x62)
15 00         Logical Minimum (0)
26 ff 00      Logical Maximum (255)
95 40         Report Count (64)      ← 64 bytes
75 08         Report Size (8)
81 02         Input (Data, Var, Abs)  ← 64-byte Input report
09 63         Usage (0x63)
15 00         Logical Minimum (0)
26 ff 00      Logical Maximum (255)
96 00 10      Report Count (4096)    ← 4096 bytes
75 08         Report Size (8)
91 02         Output (Data, Var, Abs) ← 4096-byte Output report
c0          End Collection
```

This confirms:
- **No report IDs** (none declared -> default ID 0)
- Output report: **4096 bytes** (this is the data page)
- Input report: **64 bytes** (this is the acknowledgment)

### The 4096/64 Mismatch
The Output report is 4096 bytes, but the endpoint max packet size is only 64.
This means a single 4096-byte HID Output report is sent as **64 USB interrupt
packets** of 64 bytes each. The USB host controller handles the fragmentation
automatically — libusb does this transparently when you write to an interrupt
endpoint.

On Windows, `WriteFile(handle, buf, 4097, ...)` sends:

- byte[0] = `0x00` (report ID, stripped by the HID driver)
- bytes[1..4096] = data -> sent as 64 × 64-byte interrupt OUT transfers

### Report ID Handling
Since interface 2 has no report IDs:

- **Windows**: `WriteFile` requires a leading `0x00` byte (the HID driver
  strips it before sending). Similarly `ReadFile` prepends `0x00` to the
  received data.
- **Linux/libusb**: When using interrupt endpoint transfers directly, we do
  **NOT** include the report ID byte. We write 4096 bytes directly.

## Phase 4: Corrected Approach (Interrupt Transfers)

### What Needs to Change

Instead of `dev.Control()` (control transfer on endpoint 0), we need to use
gousb's interrupt endpoint API:

```go
// Claim interface 2 and get endpoints
intf2, _ := cfg.Interface(2, 0)

// Find the OUT endpoint (EP 3, address 0x03)
outEP, _ := intf2.OutEndpoint(3)

// Find the IN endpoint (EP 4, address 0x84)
inEP, _ := intf2.InEndpoint(4)

// Send a 4096-byte data page (no report ID prefix)
outEP.Write(pageData)  // libusb fragments into 64-byte USB packets

// Read 64-byte acknowledgment
inEP.Read(ackBuf)       // with appropriate timeout via context
```

The control commands (`04 18`, `04 72`, `04 02`) remain as Feature Reports on
interface 3 via `dev.Control()` — no change there.

### Key Differences from the Failed Attempt

| Aspect         | Failed (Control Transfer)      | Corrected (Interrupt Transfer)       |
| -------------- | ------------------------------ | ------------------------------------ |
| Transfer type  | Control (SET_REPORT on EP 0)   | Interrupt (EP 3 OUT)                 |
| Sending method | `dev.Control(0x21, 0x09, ...)` | `outEndpoint.Write(data)`            |
| Reading method | `dev.Control(0xA1, 0x01, ...)` | `inEndpoint.Read(buf)`               |
| Report ID      | Part of wValue field           | Not included in data buffer          |
| Fragmentation  | None (single control transfer) | Automatic (64 × 64-byte USB packets) |
| Result         | **Firmware crash**             | **Success!** Solid red on LCD        |

### Separating Image Generation from Upload
In the first attempt, `giftest` both generated the image buffer and uploaded it
in the same program. When the upload crashed the firmware, we couldn't tell if
the image data itself was wrong or if the transfer method was the problem.

To fix this, we split the two concerns:

- **`cmd/mkimage/`**: Generates the raw LCD image buffer and writes it to a
  file. You can inspect the file with a hex editor or convert it to a viewable
  image to verify the pixel data is correct before touching the keyboard.
- **`cmd/giftest/`**: Reads a pre-built `.bin` file and uploads it to the
  keyboard via the interrupt endpoints.

This way we can verify the image is correct independently, and if the upload
crashes the keyboard again we know the problem is the USB transfer, not the
image data.

## Open Questions (Resolved)

1. **Does gousb handle the 4096-byte write fragmentation correctly?**
   **Yes.** libusb splits the write into 64 × 64-byte interrupt transfers
   automatically. Works perfectly.

2. **Acknowledgment timing.** 300ms context timeout works. In practice the
   keyboard responds much faster — no timeouts observed during 3,386 pages.

3. **Does the keyboard send an ack after each 4096-byte page, or after each
   64-byte USB packet?** **Per page.** One 64-byte ACK (`01 5a 02 00 ...`)
   after each complete 4096-byte page write.

4. **Endpoint numbering in gousb.** `OutEndpoint(3)` and `InEndpoint(4)` work
   correctly — gousb uses the endpoint number, not the full address.

## Phase 5: Success!
Using interrupt transfers worked. Uploaded a solid red 240x135 single-frame
image and it displayed correctly on the LCD.

Output from the successful run:

```
File: test.bin (65536 bytes)
Frames: 1, pages: 16
OUT endpoint: ep #3 OUT (address 0x03) interrupt - undefined usage [64 bytes]
IN endpoint: ep #4 IN (address 0x84) interrupt - undefined usage [64 bytes]
FEAT Begin     : 04180000000000000000000000000000
  RECV:       04180001000000000000000000000000
FEAT ImgHeader : 04720100000000001000000000000000
  RECV:       04720101000000001000000000000000
PAGE 1/16: writing 4096 bytes via interrupt OUT...
  wrote 4096 bytes
  ACK (64 bytes): 015a0200000000000000000000000000
...
PAGE 16/16: writing 4096 bytes via interrupt OUT...
  wrote 4096 bytes
  ACK (64 bytes): 015a0200000000000000000000000000
FEAT Apply     : 04020000000000000000000000000000
  RECV:       04020001000000000000000000000000
Done! Check the LCD screen.
```

Key observations from the successful transfer:

- Every page ACK is `01 5a 02 00 ...` — the `5a` matches the magic byte used
  in the clock sync protocol.
- The `04 72` header readback echoes byte[3]=`01` (ACK) and byte[2]=`01`
  (confirming image number).
- No delays needed between pages beyond what libusb/the interrupt transfer
  naturally provides.
- Total transfer time for 16 pages was roughly 1-2 seconds.

### Summary of What Works

| Transfer            | Method               | Interface | Endpoint    | Status               |
| ------------------- | -------------------- | --------- | ----------- | -------------------- |
| Begin/Apply/Header  | Control (Feature)    | 3         | EP 0 (ctrl) | Works                |
| Data pages (4096 B) | Interrupt OUT        | 2         | EP 3 OUT    | Works                |
| Page acknowledgment | Interrupt IN         | 2         | EP 4 IN     | Works                |
| Data pages (4096 B) | Control (SET_REPORT) | 2         | EP 0 (ctrl) | **Crashes firmware** |

## Phase 6: Multi-Frame GIF Animation Upload
After the solid-color test worked, we uploaded the original 214-frame GIF
animation that ships with the keyboard software 
(`original-files/gif/AULA F108Pro 三模机械键盘/0.gif`).

### GIF-to-Raw Conversion
The GIF file on disk is a standard GIF89a. The keyboard does not understand GIF
format — the Windows software decodes it and converts each frame to raw RGB565
before uploading. We added `-gif` support to `cmd/mkimage/` using Go's
`image/gif` package which handles:

- GIF frame compositing (disposal methods, partial frames, transparency)
- RGBA to RGB565 conversion (5-6-5 bit packing, little-endian)
- Header construction with per-frame delay values

### Size Comparison

| Stage               | Size                        | Notes                             |
| ------------------- | --------------------------- | --------------------------------- |
| Original GIF        | 3,009,866 bytes (~2.9 MB)   | GIF89a, LZW compressed            |
| Raw RGB565 buffer   | 13,869,056 bytes (~13.2 MB) | 256 header + 214 × 64,800 pixels  |
| Page count          | 3,386 pages                 | 3,386 × 4,096 = 13,869,056        |
| Padding waste       | 1,600 bytes                 | Last page partially filled (0xFF) |
| USB packets on wire | 216,704                     | 3,386 pages × 64 packets/page     |

The 4.6x size increase (2.9 MB -> 13.2 MB) is because the keyboard stores
uncompressed pixel data. The keyboard likely has an external SPI NOR flash chip
(16 MB or 32 MB are common for cheap keyboards with LCD screens) — the Sonix
SN32 MCU doesn't have enough internal flash for 13 MB.

### Upload Results
The upload completed successfully. All 3,386 pages were sent with ACKs from the
keyboard. After the upload, the keyboard showed a progress bar on the LCD while
it wrote the data to internal flash. Once it reached 100%, the animation started
playing — **confirmed working with all 214 frames**.

### Frame Count Limit
From the Ghidra decompilation of `FUN_00422b50`, the Windows software caps
the frame count:

```c
if (*(int *)(param_1 + 0x527c8) < local_1040) {
    local_1040 = *(int *)(param_1 + 0x527c8);
}
```

The value at offset `0x527c8` is the max frame count allowed by the device
configuration. The `config.xml` specifies `gif_maxframes="141"` for this
keyboard. The 214-frame GIF that shipped with the software **exceeded this limit
by 73 frames**, and the extra data overwrote the keyboard's built-in menu
graphics (see WARNING section below).

The max safe upload is **141 frames** (141 x 64,800 = 9,136,800 bytes raw
RGB565, plus 256-byte header = ~8.9 MB, or 2,231 pages).

### Root Cause of the Corruption
The original Ghidra analysis correctly identified both the `gif_maxframes`
config value (141) and the cap check in `FUN_00422b50`. However, the analysis
concluded the cap was "at least 214" because the original gif embedded in the
app had 214 frames and our Go tool uploaded all 214 frames and the keyboard
accepted every page with ACKs. The animation played fine.

The error was assuming the **keyboard firmware enforces the frame limit**. It
does not. The firmware blindly writes whatever you send via `04 72` + data
pages to SPI flash with no bounds checking. The 141-frame limit exists
**only in the Windows software's upload function** (`FUN_00422b50`), which
caps the frame count before building the upload buffer.

Our Go tool bypassed the Windows software entirely and sent the raw 214-frame
buffer directly to the keyboard. The keyboard firmware accepted all 3,386 pages,
wrote them to SPI flash, and the 73 extra frames' worth of data (4.7 MB)
overflowed past the image slot boundary into the adjacent menu graphics region.

The `gif_maxframes="141"` config value was a **hard limit on the SPI flash
layout**, not a soft suggestion. The Windows software enforces it as a
software-side check because the firmware has no protection. Our tool should have
respected this limit from the start.

### Tools
The workflow for uploading a GIF animation:

```bash
# 1. Convert GIF to raw buffer
go run ./cmd/mkimage/ -gif animation.gif -o anim.bin

# 2. Upload to keyboard
go run ./cmd/aula/ lcd anim.bin
```

The GIF must be **240x135 pixels** to match the LCD screen dimensions.
Non-matching sizes will be cropped (larger) or black-padded (smaller).

## WARNING: LCD Upload Can Corrupt Keyboard Menu Graphics
Uploading the 214-frame original GIF (13.2 MB raw, 3386 pages) to the keyboard
**overwrote the built-in menu graphics**. After the upload, the knob menu showed
garbled/corrupted screens instead of the normal settings UI. Only the GIF
animation played — the clock, brightness, and other menu screens were gone.

The cause: the `config.xml` specifies `gif_maxframes="141"` which is the maximum
the keyboard's flash can hold in the image slot. The 214-frame GIF exceeded this
by 73 frames. The extra 73 frames (roughly 4.7 MB of raw RGB565 data) overflowed
past the image slot boundary and overwrote the adjacent flash region containing
the menu graphics.

* Max safe frames: **141** (141 x 64,800 = ~8.9 MB + 256-byte header = 2,231 pages)
* Uploaded frames: 214 (214 x 64,800 = ~13.2 MB + header = 3,386 pages)
* Overflow: 73 frames (~4.7 MB, 1,155 pages past the boundary)

**Status**: The `aula lcd` command has been re-added with a 141-frame limit
enforced. Use `--force` to bypass the limit for recovery experiments.

**Recovery**: No known full recovery method. The menu graphics data was
physically overwritten in SPI flash. Tested all of the following:

* Factory reset (`fn+esc`): only resets lighting colors/modes, not LCD graphics
* Firmware update (Sonix ISP flasher V1.07): only reflashes MCU internal
  flash, menu graphics live on a separate SPI NOR flash chip
* Uploading a small 1-frame image via `aula lcd test.bin`: the image slot
  updated correctly (solid red displayed), but the menu screens remained
  corrupted with garbled pixels
* Re-uploading the 214-frame GIF via the Windows software 1.0.0.3: the
  software caps at 141 frames (does not re-corrupt), no improvement
* Uploading a 214-frame "clear" image (`mkclearimage`): 5 red frames + 209
  black frames, same total size as the original overflow. This replaced the
  garbled data with black (zeros). **Partial recovery**: the clock screen now
  has a transparent/black background instead of artifacts, and the menu
  structure (knob navigation) still works — menus can be entered and
  interacted with blind. The menu option screens are blank (black) but
  functional. The GIF animation area shows the red frames as expected.

This confirms the menu graphics are stored as raw image data on SPI flash
immediately after the GIF slot. Overwriting them with zeros makes them blank
instead of garbled. Full recovery would require the original menu graphic data,
which is factory-burned and not available in the Windows software. Contacted
vendor support to request a recovery utility.

**TODO**:
* Enforce `gif_maxframes` (141) limit in mkimage and upload code before re-enabling
* Test uploading a GIF with <=141 frames to confirm it stays within bounds
* Investigate exact flash layout — image slot boundaries, menu graphics location
