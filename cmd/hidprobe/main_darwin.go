//go:build darwin

package main

import (
	"fmt"
	"os"

	"github.com/sstallion/go-hid"
)

const (
	vendorID  = 0x0C45
	productID = 0x800A

	featureInterface = 3
	featureUsagePage = 0xFF13
	lcdInterface     = 2
	lcdUsagePage     = 0xFF68
)

func main() {
	hid.SetOpenExclusive(false)
	if err := hid.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "HID initialization failed: %v\n", err)
		os.Exit(1)
	}
	defer hid.Exit()

	found := 0
	openFailures := 0
	err := hid.Enumerate(vendorID, productID, func(info *hid.DeviceInfo) error {
		isFeature := info.InterfaceNbr == featureInterface && info.UsagePage == featureUsagePage
		isLCD := info.InterfaceNbr == lcdInterface && info.UsagePage == lcdUsagePage
		if info.BusType != hid.BusUSB || (!isFeature && !isLCD) {
			return nil
		}
		found++
		fmt.Printf(
			"interface=%d usage_page=0x%04X usage=0x%04X product=%q\n",
			info.InterfaceNbr, info.UsagePage, info.Usage, info.ProductStr,
		)

		dev, err := hid.OpenPath(info.Path)
		if err != nil {
			openFailures++
			fmt.Printf("  open: FAILED: %v\n", err)
			return nil
		}
		fmt.Println("  open: OK (non-exclusive)")
		if err := dev.Close(); err != nil {
			fmt.Printf("  close: FAILED: %v\n", err)
			openFailures++
		} else {
			fmt.Println("  close: OK")
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "HID enumeration failed: %v\n", err)
		os.Exit(1)
	}
	if found == 0 {
		fmt.Fprintf(os.Stderr, "AULA F108Pro USB HID interfaces were not found\n")
		os.Exit(1)
	}
	if openFailures != 0 {
		fmt.Fprintf(os.Stderr, "%d interface(s) could not be opened\n", openFailures)
		os.Exit(1)
	}
}
