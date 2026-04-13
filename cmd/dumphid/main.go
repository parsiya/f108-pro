package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/google/gousb"
)

func main() {
	ctx := gousb.NewContext()
	defer ctx.Close()

	dev, err := ctx.OpenDeviceWithVIDPID(0x0C45, 0x800A)
	if err != nil || dev == nil {
		fmt.Fprintf(os.Stderr, "Cannot open device: %v\n", err)
		os.Exit(1)
	}
	defer dev.Close()
	dev.SetAutoDetach(true)

	// Dump HID report descriptor for each interface.
	// HID descriptor class request: GET_DESCRIPTOR with type=0x22 (Report).
	// bmRequestType: 0x81 (Device-to-host, Standard, Interface).
	for ifNum := 0; ifNum < 4; ifNum++ {
		fmt.Printf("\n=== Interface %d HID Report Descriptor ===\n", ifNum)

		buf := make([]byte, 4096)
		n, err := dev.Control(
			0x81,            // Device-to-host, Standard, Interface.
			0x06,            // GET_DESCRIPTOR.
			0x2200,          // Report descriptor type (0x22) in high byte.
			uint16(ifNum),   // Interface number.
			buf,
		)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}
		fmt.Printf("  Length: %d bytes\n", n)
		// Print in rows of 16.
		for i := 0; i < n; i += 16 {
			end := i + 16
			if end > n {
				end = n
			}
			fmt.Printf("  %04x: %s\n", i, hex.EncodeToString(buf[i:end]))
		}

		// Quick parse: look for Report ID items (0x85 = Report ID).
		fmt.Printf("  Report IDs found: ")
		ids := []byte{}
		for i := 0; i < n; i++ {
			if buf[i] == 0x85 && i+1 < n {
				ids = append(ids, buf[i+1])
			}
		}
		if len(ids) == 0 {
			fmt.Println("none (uses default ID 0)")
		} else {
			for _, id := range ids {
				fmt.Printf("0x%02X ", id)
			}
			fmt.Println()
		}

		// Look for Feature items (0xB1 = Feature).
		fmt.Printf("  Feature reports: ")
		featureCount := 0
		for i := 0; i < n; i++ {
			if buf[i] == 0xB1 {
				featureCount++
			}
		}
		fmt.Printf("%d found\n", featureCount)

		// Look for Usage Page (0x05 or 0x06 for extended).
		fmt.Printf("  Usage pages: ")
		for i := 0; i < n; i++ {
			if buf[i] == 0x05 && i+1 < n {
				fmt.Printf("0x%02X ", buf[i+1])
			} else if buf[i] == 0x06 && i+2 < n {
				fmt.Printf("0x%02X%02X ", buf[i+2], buf[i+1])
			}
		}
		fmt.Println()
	}
}
