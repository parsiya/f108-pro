package aula

import (
	"fmt"
	"os"
)

const (
	lcdInterfaceID = 2  // Interface 2: LCD data (usage page 0xFF68).
	lcdOutEP       = 3  // EP 3 OUT: interrupt OUT for data pages.
	lcdInEP        = 4  // EP 4 IN: interrupt IN for acknowledgments.
	lcdPageSize    = 4096
)

// UploadLCDImage uploads a raw LCD image buffer to the keyboard.
//
// The buffer must be pre-built in the keyboard's native format:
//   - 256-byte header (byte[0]=frame count, byte[1..N]=per-frame delay, rest 0xFF)
//   - RGB565 pixel data (240x135x2 = 64800 bytes per frame)
//   - Padded to a multiple of 4096 bytes
//
// Use cmd/mkimage to generate test images, or build the buffer programmatically.
//
// The imageNumber selects the image slot on the keyboard (1-based).
//
// If progress is non-nil, it is called after each page with (pagesSent, totalPages).
func (d *Device) UploadLCDImage(buf []byte, imageNumber uint8, progress func(sent, total int)) error {
	if len(buf) == 0 || len(buf)%lcdPageSize != 0 {
		return fmt.Errorf("buffer size %d is not a positive multiple of %d", len(buf), lcdPageSize)
	}

	if err := d.transport.InitLCD(); err != nil {
		return fmt.Errorf("LCD init: %w", err)
	}

	pageCount := len(buf) / lcdPageSize

	// Step 1: Begin (Feature Report on interface 3).
	if err := d.beginTransaction(); err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	// Step 2: Image header 04 72 (Feature Report on interface 3).
	hdr := make([]byte, 64)
	hdr[0] = 0x04
	hdr[1] = 0x72
	hdr[2] = imageNumber
	hdr[8] = byte(pageCount & 0xFF)
	hdr[9] = byte((pageCount >> 8) & 0xFF)
	if err := d.sendCommand(hdr, true); err != nil {
		return fmt.Errorf("image header: %w", err)
	}

	// Step 3: Send data pages via the LCD data interface.
	for i := 0; i < pageCount; i++ {
		page := buf[i*lcdPageSize : (i+1)*lcdPageSize]

		if err := d.transport.WriteLCDPage(page); err != nil {
			return fmt.Errorf("page %d/%d write: %w", i+1, pageCount, err)
		}

		if _, err := d.transport.ReadLCDAck(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: page %d/%d ack timeout: %v\n", i+1, pageCount, err)
		}

		if progress != nil {
			progress(i+1, pageCount)
		}
	}

	// Step 4: Apply (Feature Report on interface 3).
	if err := d.applyTransaction(); err != nil {
		return fmt.Errorf("apply: %w", err)
	}

	return nil
}
