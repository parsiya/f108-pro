//go:build darwin

package aula

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sstallion/go-hid"
)

const (
	featureUsagePage = 0xFF13
	lcdUsagePage     = 0xFF68
	lcdAckTimeout    = 300 * time.Millisecond
	hidRetryAttempts = 3
)

var (
	hidLifecycleMu sync.Mutex
	hidUsers       int
)

// darwinHIDTransport implements Transport using HIDAPI's native IOKit
// backend. Opening in non-exclusive mode is important for keyboards: macOS
// keeps delivering normal key events while the vendor interfaces are open.
type darwinHIDTransport struct {
	feature *hid.Device
	lcd     *hid.Device
	closed  bool
}

func openTransport() (Transport, error) {
	if err := acquireHID(); err != nil {
		return nil, fmt.Errorf("initializing macOS HID: %w", err)
	}

	path, err := findDarwinHIDPath(featureUsagePage, InterfaceID)
	if err != nil {
		releaseHID()
		return nil, fmt.Errorf("finding keyboard feature interface: %w", err)
	}

	dev, err := hid.OpenPath(path)
	if err != nil {
		releaseHID()
		return nil, darwinOpenError("opening keyboard feature interface", err)
	}

	return &darwinHIDTransport{feature: dev}, nil
}

func acquireHID() error {
	hidLifecycleMu.Lock()
	defer hidLifecycleMu.Unlock()

	if hidUsers == 0 {
		hid.SetOpenExclusive(false)
		if err := hid.Init(); err != nil {
			return err
		}
	}
	hidUsers++
	return nil
}

func releaseHID() {
	hidLifecycleMu.Lock()
	defer hidLifecycleMu.Unlock()

	if hidUsers == 0 {
		return
	}
	hidUsers--
	if hidUsers == 0 {
		_ = hid.Exit()
	}
}

func findDarwinHIDPath(usagePage uint16, interfaceNbr int) (string, error) {
	var path string
	err := hid.Enumerate(VendorID, ProductID, func(info *hid.DeviceInfo) error {
		if path == "" && info.BusType == hid.BusUSB &&
			info.UsagePage == usagePage && info.InterfaceNbr == interfaceNbr {
			path = info.Path
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf(
			"keyboard not found (VID=%04x PID=%04x interface=%d usage_page=%04x)",
			VendorID, ProductID, interfaceNbr, usagePage,
		)
	}
	return path, nil
}

func darwinOpenError(operation string, err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "not permitted") {
		return fmt.Errorf(
			"%s: %w; grant the terminal or application Input Monitoring access in System Settings > Privacy & Security",
			operation, err,
		)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// retryDarwinHIDOperation retries transient IOKit feature-report failures.
// The keyboard occasionally rejects an otherwise valid report while it is
// still processing the previous command.
func retryDarwinHIDOperation(operation func() (int, error)) (int, error) {
	var (
		n   int
		err error
	)
	for attempt := 0; attempt < hidRetryAttempts; attempt++ {
		n, err = operation()
		if err == nil {
			return n, nil
		}
		if attempt+1 < hidRetryAttempts {
			time.Sleep(cmdDelay)
		}
	}
	return n, err
}

func (t *darwinHIDTransport) SetFeatureReport(data [reportSize]byte) error {
	buf := make([]byte, reportSize+1)
	copy(buf[1:], data[:]) // Byte 0 is report ID 0 for an unnumbered report.

	n, err := retryDarwinHIDOperation(func() (int, error) {
		return t.feature.SendFeatureReport(buf)
	})
	if err != nil {
		return fmt.Errorf("SET_REPORT: %w", err)
	}
	if n != len(buf) {
		return fmt.Errorf("SET_REPORT: short write: got %d bytes, want %d", n, len(buf))
	}
	return nil
}

func (t *darwinHIDTransport) GetFeatureReport() ([reportSize]byte, error) {
	buf := make([]byte, reportSize+1)

	n, err := retryDarwinHIDOperation(func() (int, error) {
		clear(buf)
		buf[0] = 0 // Requested report ID.
		return t.feature.GetFeatureReport(buf)
	})
	if err != nil {
		return [reportSize]byte{}, fmt.Errorf("GET_REPORT: %w", err)
	}
	if n != len(buf) {
		return [reportSize]byte{}, fmt.Errorf("GET_REPORT: short read: got %d bytes, want %d", n, len(buf))
	}

	var result [reportSize]byte
	copy(result[:], buf[1:])
	return result, nil
}

func (t *darwinHIDTransport) InitLCD() error {
	if t.lcd != nil {
		return nil
	}

	path, err := findDarwinHIDPath(lcdUsagePage, lcdInterfaceID)
	if err != nil {
		return fmt.Errorf("finding LCD interface: %w", err)
	}

	dev, err := hid.OpenPath(path)
	if err != nil {
		return darwinOpenError("opening LCD interface", err)
	}
	t.lcd = dev
	return nil
}

func (t *darwinHIDTransport) WriteLCDPage(data []byte) error {
	if t.lcd == nil {
		return errors.New("LCD interface is not initialized")
	}
	if len(data) != lcdPageSize {
		return fmt.Errorf("LCD page size is %d bytes, want %d", len(data), lcdPageSize)
	}

	buf := make([]byte, lcdPageSize+1)
	copy(buf[1:], data) // Byte 0 is report ID 0.
	n, err := t.lcd.Write(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return fmt.Errorf("short LCD write: got %d bytes, want %d", n, len(buf))
	}
	return nil
}

func (t *darwinHIDTransport) ReadLCDAck() ([]byte, error) {
	if t.lcd == nil {
		return nil, errors.New("LCD interface is not initialized")
	}

	// HIDAPI omits the report ID for an unnumbered input report, but tolerate
	// a leading zero so this stays correct if the descriptor changes.
	buf := make([]byte, reportSize+1)
	n, err := t.lcd.ReadWithTimeout(buf, lcdAckTimeout)
	if err != nil {
		return nil, err
	}
	switch {
	case n == reportSize:
		return append([]byte(nil), buf[:reportSize]...), nil
	case n == reportSize+1 && buf[0] == 0:
		return append([]byte(nil), buf[1:]...), nil
	default:
		return nil, fmt.Errorf("unexpected LCD acknowledgment size: %d", n)
	}
}

func (t *darwinHIDTransport) Close() {
	if t.closed {
		return
	}
	t.closed = true

	if t.lcd != nil {
		_ = t.lcd.Close()
		t.lcd = nil
	}
	if t.feature != nil {
		_ = t.feature.Close()
		t.feature = nil
	}
	releaseHID()
}
