package aula

// Transport abstracts the OS-level USB/HID communication with the keyboard.
// Two implementations exist:
//   - transport_linux.go: uses gousb (libusb) for Linux
//   - transport_windows.go: uses native Windows HID API (hid.dll)
type Transport interface {
	// SetFeatureReport sends a 64-byte HID feature report on interface 3.
	SetFeatureReport(data [reportSize]byte) error

	// GetFeatureReport reads a 64-byte HID feature report from interface 3.
	GetFeatureReport() ([reportSize]byte, error)

	// InitLCD prepares the LCD data interface (interface 2) for transfers.
	// This is called lazily before the first LCD data page is sent.
	InitLCD() error

	// WriteLCDPage sends a 4096-byte data page via the LCD data interface.
	WriteLCDPage(data []byte) error

	// ReadLCDAck reads a 64-byte acknowledgment from the LCD data interface
	// with a timeout. A timeout error is not fatal.
	ReadLCDAck() ([]byte, error)

	// Close releases all USB/HID resources.
	Close()
}
