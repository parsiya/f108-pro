//go:build windows

package aula

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	modHID    = syscall.NewLazyDLL("hid.dll")
	modSetupAPI = syscall.NewLazyDLL("setupapi.dll")
	modKernel32 = syscall.NewLazyDLL("kernel32.dll")

	procHidD_GetHidGuid               = modHID.NewProc("HidD_GetHidGuid")
	procHidD_GetAttributes            = modHID.NewProc("HidD_GetAttributes")
	procHidD_SetFeature               = modHID.NewProc("HidD_SetFeature")
	procHidD_GetFeature               = modHID.NewProc("HidD_GetFeature")
	procHidD_GetPreparsedData         = modHID.NewProc("HidD_GetPreparsedData")
	procHidD_FreePreparsedData        = modHID.NewProc("HidD_FreePreparsedData")
	procHidP_GetCaps                  = modHID.NewProc("HidP_GetCaps")
	procSetupDiGetClassDevsW          = modSetupAPI.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInterfaces   = modSetupAPI.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetailW = modSetupAPI.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procSetupDiDestroyDeviceInfoList  = modSetupAPI.NewProc("SetupDiDestroyDeviceInfoList")
	procCreateFileW                   = modKernel32.NewProc("CreateFileW")
	procCloseHandle                   = modKernel32.NewProc("CloseHandle")
	procWriteFile                     = modKernel32.NewProc("WriteFile")
	procReadFile                      = modKernel32.NewProc("ReadFile")
)

const (
	digcfPresent         = 0x02
	digcfDeviceInterface = 0x10
	invalidHandleValue   = ^syscall.Handle(0)
	genericRead          = 0x80000000
	genericWrite         = 0x40000000
	fileShareRead        = 0x01
	fileShareWrite       = 0x02
	openExisting         = 3
	fileFlagOverlapped   = 0x40000000
)

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type spDeviceInterfaceData struct {
	CbSize    uint32
	ClassGUID guid
	Flags     uint32
	Reserved  uintptr
}

type hidAttributes struct {
	Size      uint32
	VendorID  uint16
	ProductID uint16
	VersionNumber uint16
}

type hidpCaps struct {
	Usage                     uint16
	UsagePage                 uint16
	InputReportByteLength     uint16
	OutputReportByteLength    uint16
	FeatureReportByteLength   uint16
	Reserved                  [17]uint16
	NumberLinkCollectionNodes uint16
	NumberInputButtonCaps     uint16
	NumberInputValueCaps      uint16
	NumberInputDataIndices    uint16
	NumberOutputButtonCaps    uint16
	NumberOutputValueCaps     uint16
	NumberOutputDataIndices   uint16
	NumberFeatureButtonCaps   uint16
	NumberFeatureValueCaps    uint16
	NumberFeatureDataIndices  uint16
}

// winHIDTransport implements Transport using the Windows HID API.
type winHIDTransport struct {
	featureHandle syscall.Handle // Interface 3: 64-byte feature reports.
	lcdHandle     syscall.Handle // Interface 2: 4096-byte output reports (LCD data).
	lcdInitDone   bool
}

func openTransport() (Transport, error) {
	path, err := findHIDDevice(VendorID, ProductID, 0xFF13)
	if err != nil {
		return nil, fmt.Errorf("finding keyboard: %w", err)
	}

	handle, err := openHIDHandle(path)
	if err != nil {
		return nil, fmt.Errorf("opening keyboard: %w", err)
	}

	return &winHIDTransport{
		featureHandle: handle,
		lcdHandle:     invalidHandleValue,
	}, nil
}

func (t *winHIDTransport) SetFeatureReport(data [reportSize]byte) error {
	// Windows HidD_SetFeature expects report ID as the first byte.
	// Interface 3 has no report ID, so we prepend 0x00.
	buf := make([]byte, reportSize+1)
	buf[0] = 0x00
	copy(buf[1:], data[:])

	r, _, err := procHidD_SetFeature.Call(
		uintptr(t.featureHandle),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if r == 0 {
		return fmt.Errorf("HidD_SetFeature: %w", err)
	}
	return nil
}

func (t *winHIDTransport) GetFeatureReport() ([reportSize]byte, error) {
	// HidD_GetFeature: buf[0] = report ID (0x00), rest is filled.
	buf := make([]byte, reportSize+1)
	buf[0] = 0x00

	r, _, err := procHidD_GetFeature.Call(
		uintptr(t.featureHandle),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if r == 0 {
		return [reportSize]byte{}, fmt.Errorf("HidD_GetFeature: %w", err)
	}

	var result [reportSize]byte
	copy(result[:], buf[1:]) // Strip the report ID byte.
	return result, nil
}

func (t *winHIDTransport) InitLCD() error {
	if t.lcdInitDone {
		return nil
	}

	path, err := findHIDDevice(VendorID, ProductID, 0xFF68)
	if err != nil {
		return fmt.Errorf("finding LCD interface: %w", err)
	}

	handle, err := openHIDHandle(path)
	if err != nil {
		return fmt.Errorf("opening LCD interface: %w", err)
	}

	t.lcdHandle = handle
	t.lcdInitDone = true
	return nil
}

func (t *winHIDTransport) WriteLCDPage(data []byte) error {
	// WriteFile on HID output report: prepend report ID 0x00.
	buf := make([]byte, len(data)+1)
	buf[0] = 0x00
	copy(buf[1:], data)

	var written uint32
	// Use synchronous WriteFile. The HID driver handles interrupt transfer.
	r, _, err := procWriteFile.Call(
		uintptr(t.lcdHandle),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&written)),
		0, // No overlapped.
	)
	if r == 0 {
		return fmt.Errorf("WriteFile LCD: %w", err)
	}
	return nil
}

func (t *winHIDTransport) ReadLCDAck() ([]byte, error) {
	// ReadFile on HID input report: report ID 0x00 + 64 bytes data.
	buf := make([]byte, 65)
	buf[0] = 0x00

	var read uint32
	r, _, err := procReadFile.Call(
		uintptr(t.lcdHandle),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&read)),
		0,
	)
	if r == 0 {
		return nil, fmt.Errorf("ReadFile LCD ack: %w", err)
	}

	// Strip report ID, return 64 data bytes.
	result := make([]byte, 64)
	copy(result, buf[1:])
	return result, nil
}

func (t *winHIDTransport) Close() {
	if t.lcdHandle != invalidHandleValue {
		procCloseHandle.Call(uintptr(t.lcdHandle))
	}
	if t.featureHandle != invalidHandleValue {
		procCloseHandle.Call(uintptr(t.featureHandle))
	}
}

// findHIDDevice enumerates HID devices and returns the device path matching
// the given VID, PID, and HID usage page.
func findHIDDevice(vid, pid uint16, usagePage uint16) (string, error) {
	var hidGUID guid
	procHidD_GetHidGuid.Call(uintptr(unsafe.Pointer(&hidGUID)))

	devInfo, _, _ := procSetupDiGetClassDevsW.Call(
		uintptr(unsafe.Pointer(&hidGUID)),
		0,
		0,
		digcfPresent|digcfDeviceInterface,
	)
	if syscall.Handle(devInfo) == invalidHandleValue {
		return "", fmt.Errorf("SetupDiGetClassDevsW failed")
	}
	defer procSetupDiDestroyDeviceInfoList.Call(devInfo)

	var ifaceData spDeviceInterfaceData
	ifaceData.CbSize = uint32(unsafe.Sizeof(ifaceData))

	for i := uint32(0); ; i++ {
		r, _, _ := procSetupDiEnumDeviceInterfaces.Call(
			devInfo, 0,
			uintptr(unsafe.Pointer(&hidGUID)),
			uintptr(i),
			uintptr(unsafe.Pointer(&ifaceData)),
		)
		if r == 0 {
			break
		}

		path, err := getDeviceInterfaceDetail(devInfo, &ifaceData)
		if err != nil {
			continue
		}

		handle, err := openHIDHandle(path)
		if err != nil {
			continue
		}

		var attrs hidAttributes
		attrs.Size = uint32(unsafe.Sizeof(attrs))
		r, _, _ = procHidD_GetAttributes.Call(
			uintptr(handle),
			uintptr(unsafe.Pointer(&attrs)),
		)
		if r == 0 {
			procCloseHandle.Call(uintptr(handle))
			continue
		}

		if attrs.VendorID != vid || attrs.ProductID != pid {
			procCloseHandle.Call(uintptr(handle))
			continue
		}

		// Check usage page via preparsed data.
		var preparsed uintptr
		r, _, _ = procHidD_GetPreparsedData.Call(
			uintptr(handle),
			uintptr(unsafe.Pointer(&preparsed)),
		)
		if r == 0 {
			procCloseHandle.Call(uintptr(handle))
			continue
		}

		var caps hidpCaps
		procHidP_GetCaps.Call(preparsed, uintptr(unsafe.Pointer(&caps)))
		procHidD_FreePreparsedData.Call(preparsed)
		procCloseHandle.Call(uintptr(handle))

		if caps.UsagePage == usagePage {
			return path, nil
		}
	}

	return "", fmt.Errorf("device not found (VID=%04x PID=%04x UsagePage=%04x)", vid, pid, usagePage)
}

// getDeviceInterfaceDetail returns the device path for a SetupDi interface.
func getDeviceInterfaceDetail(devInfo uintptr, ifaceData *spDeviceInterfaceData) (string, error) {
	// First call to get required buffer size.
	var reqSize uint32
	procSetupDiGetDeviceInterfaceDetailW.Call(
		devInfo,
		uintptr(unsafe.Pointer(ifaceData)),
		0, 0,
		uintptr(unsafe.Pointer(&reqSize)),
		0,
	)
	if reqSize == 0 {
		return "", fmt.Errorf("GetDeviceInterfaceDetailW size query failed")
	}

	// Allocate buffer. The struct has a 4-byte CbSize header followed by the
	// null-terminated UTF-16 device path.
	buf := make([]byte, reqSize)

	// CbSize of SP_DEVICE_INTERFACE_DETAIL_DATA_W.
	// On 64-bit Windows it's 8 (4 byte CbSize + 4 byte alignment before wchar).
	// On 32-bit Windows it's 6 (4 byte CbSize + 2 byte wchar start).
	cbSize := uint32(8)
	if unsafe.Sizeof(uintptr(0)) == 4 {
		cbSize = 6
	}
	*(*uint32)(unsafe.Pointer(&buf[0])) = cbSize

	r, _, err := procSetupDiGetDeviceInterfaceDetailW.Call(
		devInfo,
		uintptr(unsafe.Pointer(ifaceData)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(reqSize),
		0, 0,
	)
	if r == 0 {
		return "", fmt.Errorf("GetDeviceInterfaceDetailW: %w", err)
	}

	// Path starts after the 4-byte CbSize field, as UTF-16.
	pathBytes := buf[4:]
	path := syscall.UTF16ToString((*[1024]uint16)(unsafe.Pointer(&pathBytes[0]))[:])

	// Trim at first null.
	if idx := strings.IndexByte(path, 0); idx >= 0 {
		path = path[:idx]
	}

	return path, nil
}

// openHIDHandle opens a HID device by its device path.
func openHIDHandle(path string) (syscall.Handle, error) {
	pathUTF16, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return invalidHandleValue, err
	}

	// Give it a moment for the device to be ready.
	time.Sleep(1 * time.Millisecond)

	r, _, callErr := procCreateFileW.Call(
		uintptr(unsafe.Pointer(pathUTF16)),
		genericRead|genericWrite,
		fileShareRead|fileShareWrite,
		0,
		openExisting,
		0, // Synchronous I/O (no FILE_FLAG_OVERLAPPED).
		0,
	)
	handle := syscall.Handle(r)
	if handle == invalidHandleValue {
		return invalidHandleValue, fmt.Errorf("CreateFileW: %w", callErr)
	}
	return handle, nil
}
