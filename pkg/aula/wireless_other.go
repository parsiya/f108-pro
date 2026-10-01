//go:build !linux

package aula

import "errors"

type WirelessDevice struct{}

var errWirelessUnsupported = errors.New("el modo 2.4G solo está soportado en Linux por ahora")

func OpenWireless() (*WirelessDevice, error)                     { return nil, errWirelessUnsupported }
func (d *WirelessDevice) Close() error                           { return nil }
func (d *WirelessDevice) SetKeyRemap(_ []KeyRemap, _ bool) error { return errWirelessUnsupported }

func (d *WirelessDevice) ResetKeys(_ []uint8, _ bool) error { return errWirelessUnsupported }
