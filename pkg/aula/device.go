package aula

import (
	"fmt"
	"time"
)

const (
	VendorID    = 0x0C45
	ProductID   = 0x800A
	InterfaceID = 3 // Feature reports are on interface 3.

	// HID feature report size (no report ID prefix, 64 bytes).
	reportSize = 64

	// USB control transfer constants for HID Set/Get Feature.
	reqTypeOut        = 0x21   // Host-to-device, Class, Interface.
	reqTypeIn         = 0xA1   // Device-to-host, Class, Interface.
	reqSetReport      = 0x09   // SET_REPORT.
	reqGetReport      = 0x01   // GET_REPORT.
	featureReportType = 0x0300 // wValue high byte: 0x03 = Feature report.

	// Delay between commands from config.xml cmd_delaytime.
	cmdDelay = 35 * time.Millisecond
)

// Device represents an open connection to an Aula F108 Pro keyboard.
type Device struct {
	transport Transport
}

// Open finds and opens the Aula F108 Pro keyboard.
func Open() (*Device, error) {
	t, err := openTransport()
	if err != nil {
		return nil, err
	}
	return &Device{transport: t}, nil
}

// Close releases the device and all resources.
func (d *Device) Close() {
	if d.transport != nil {
		d.transport.Close()
	}
}

// setFeatureReport sends a 64-byte HID feature report to the keyboard.
func (d *Device) setFeatureReport(data [reportSize]byte) error {
	return d.transport.SetFeatureReport(data)
}

// getFeatureReport reads a 64-byte HID feature report from the keyboard.
func (d *Device) getFeatureReport() ([reportSize]byte, error) {
	return d.transport.GetFeatureReport()
}

// sendCommand builds a 64-byte report from a payload and sends it.
// If readback is true, it also reads back the response from the keyboard.
func (d *Device) sendCommand(payload []byte, readback bool) error {
	var report [reportSize]byte
	copy(report[:], payload)

	if err := d.setFeatureReport(report); err != nil {
		return err
	}

	time.Sleep(cmdDelay)

	if readback {
		if _, err := d.getFeatureReport(); err != nil {
			return fmt.Errorf("readback: %w", err)
		}
		time.Sleep(cmdDelay)
	}

	return nil
}

// beginTransaction sends the 04 18 begin command with readback.
func (d *Device) beginTransaction() error {
	return d.sendCommand([]byte{0x04, 0x18}, true)
}

// applyTransaction sends the 04 02 apply command with readback.
func (d *Device) applyTransaction() error {
	return d.sendCommand([]byte{0x04, 0x02}, true)
}

// finalizeTransaction sends the 04 F0 finalize command without readback.
func (d *Device) finalizeTransaction() error {
	return d.sendCommand([]byte{0x04, 0xF0}, false)
}

// sendMultiPacket splits data into 64-byte feature reports and sends them.
// If readback is true, a GET_REPORT is done after the last packet.
//
// d.sendMultiPacket(bigPayload, true)
func (d *Device) sendMultiPacket(data []byte, readback bool) error {
	nPackets := len(data) / reportSize
	if len(data)%reportSize != 0 {
		nPackets++
	}
	for i := 0; i < nPackets; i++ {
		var report [reportSize]byte
		start := i * reportSize
		end := start + reportSize
		if end > len(data) {
			end = len(data)
		}
		copy(report[:], data[start:end])
		if err := d.setFeatureReport(report); err != nil {
			return fmt.Errorf("packet %d/%d: %w", i+1, nPackets, err)
		}
		time.Sleep(cmdDelay)
	}
	if readback {
		if _, err := d.getFeatureReport(); err != nil {
			return fmt.Errorf("readback: %w", err)
		}
		time.Sleep(cmdDelay)
	}
	return nil
}

// lightingInit sends the 04 13 lighting command init with byte[8]=01.
func (d *Device) lightingInit() error {
	payload := make([]byte, 64)
	payload[0] = 0x04
	payload[1] = 0x13
	payload[8] = 0x01
	return d.sendCommand(payload, true)
}
