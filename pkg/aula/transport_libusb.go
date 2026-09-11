//go:build !windows

package aula

import (
	"context"
	"fmt"
	"time"

	"github.com/google/gousb"
)

// goUSBTransport implements Transport using gousb (libusb).
type goUSBTransport struct {
	ctx  *gousb.Context
	dev  *gousb.Device
	cfg  *gousb.Config
	intf *gousb.Interface

	// LCD interface 2 — claimed lazily.
	lcdIntf *gousb.Interface
	lcdOut  *gousb.OutEndpoint
	lcdIn   *gousb.InEndpoint
}

func openTransport() (Transport, error) {
	ctx := gousb.NewContext()

	dev, err := ctx.OpenDeviceWithVIDPID(VendorID, ProductID)
	if err != nil {
		ctx.Close()
		return nil, fmt.Errorf("opening device: %w", err)
	}
	if dev == nil {
		ctx.Close()
		return nil, fmt.Errorf("keyboard not found (VID=%04x PID=%04x)", VendorID, ProductID)
	}

	if err := dev.SetAutoDetach(true); err != nil {
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("setting auto-detach: %w", err)
	}

	cfg, err := dev.Config(1)
	if err != nil {
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("getting config: %w", err)
	}

	intf, err := cfg.Interface(InterfaceID, 0)
	if err != nil {
		cfg.Close()
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("claiming interface %d: %w", InterfaceID, err)
	}

	return &goUSBTransport{
		ctx:  ctx,
		dev:  dev,
		cfg:  cfg,
		intf: intf,
	}, nil
}

func (t *goUSBTransport) SetFeatureReport(data [reportSize]byte) error {
	_, err := t.dev.Control(
		reqTypeOut,
		reqSetReport,
		featureReportType,
		uint16(InterfaceID),
		data[:],
	)
	if err != nil {
		return fmt.Errorf("SET_REPORT: %w", err)
	}
	return nil
}

func (t *goUSBTransport) GetFeatureReport() ([reportSize]byte, error) {
	var buf [reportSize]byte
	_, err := t.dev.Control(
		reqTypeIn,
		reqGetReport,
		featureReportType,
		uint16(InterfaceID),
		buf[:],
	)
	if err != nil {
		return buf, fmt.Errorf("GET_REPORT: %w", err)
	}
	return buf, nil
}

func (t *goUSBTransport) InitLCD() error {
	if t.lcdOut != nil {
		return nil
	}

	intf, err := t.cfg.Interface(lcdInterfaceID, 0)
	if err != nil {
		return fmt.Errorf("claiming interface %d: %w", lcdInterfaceID, err)
	}

	outEP, err := intf.OutEndpoint(lcdOutEP)
	if err != nil {
		intf.Close()
		return fmt.Errorf("OUT endpoint %d: %w", lcdOutEP, err)
	}

	inEP, err := intf.InEndpoint(lcdInEP)
	if err != nil {
		intf.Close()
		return fmt.Errorf("IN endpoint %d: %w", lcdInEP, err)
	}

	t.lcdIntf = intf
	t.lcdOut = outEP
	t.lcdIn = inEP
	return nil
}

func (t *goUSBTransport) WriteLCDPage(data []byte) error {
	_, err := t.lcdOut.Write(data)
	return err
}

func (t *goUSBTransport) ReadLCDAck() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	buf := make([]byte, 64)
	_, err := t.lcdIn.ReadContext(ctx, buf)
	return buf, err
}

func (t *goUSBTransport) Close() {
	if t.lcdIntf != nil {
		t.lcdIntf.Close()
	}
	if t.intf != nil {
		t.intf.Close()
	}
	if t.cfg != nil {
		t.cfg.Close()
	}
	if t.dev != nil {
		t.dev.Close()
	}
	if t.ctx != nil {
		t.ctx.Close()
	}
}
