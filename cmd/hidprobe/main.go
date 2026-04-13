//go:build windows

package main

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

var (
	modHID      = syscall.NewLazyDLL("hid.dll")
	modSetupAPI = syscall.NewLazyDLL("setupapi.dll")
	modKernel32 = syscall.NewLazyDLL("kernel32.dll")

	procHidD_GetHidGuid                  = modHID.NewProc("HidD_GetHidGuid")
	procHidD_GetAttributes               = modHID.NewProc("HidD_GetAttributes")
	procHidD_SetFeature                  = modHID.NewProc("HidD_SetFeature")
	procHidD_GetFeature                  = modHID.NewProc("HidD_GetFeature")
	procHidD_GetPreparsedData            = modHID.NewProc("HidD_GetPreparsedData")
	procHidD_FreePreparsedData           = modHID.NewProc("HidD_FreePreparsedData")
	procHidP_GetCaps                     = modHID.NewProc("HidP_GetCaps")
	procSetupDiGetClassDevsW             = modSetupAPI.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInterfaces      = modSetupAPI.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetailW = modSetupAPI.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procSetupDiDestroyDeviceInfoList     = modSetupAPI.NewProc("SetupDiDestroyDeviceInfoList")
	procCreateFileW                      = modKernel32.NewProc("CreateFileW")
	procCloseHandle                      = modKernel32.NewProc("CloseHandle")
)

type guid struct {
	D1     uint32
	D2, D3 uint16
	D4     [8]byte
}
type spDevIfData struct {
	CbSize   uint32
	GUID     guid
	Flags    uint32
	Reserved uintptr
}
type hidAttrs struct {
	Size          uint32
	VID, PID, Ver uint16
}
type hidpCaps struct {
	Usage, UsagePage                     uint16
	InLen, OutLen, FeatLen               uint16
	Reserved                             [17]uint16
	NLinkNodes                           uint16
	NInBtnCaps, NInValCaps, NInIdx       uint16
	NOutBtnCaps, NOutValCaps, NOutIdx    uint16
	NFeatBtnCaps, NFeatValCaps, NFeatIdx uint16
}

const (
	inv     = ^syscall.Handle(0)
	digcfPI = 0x12
)

func main() {
	var hGUID guid
	procHidD_GetHidGuid.Call(uintptr(unsafe.Pointer(&hGUID)))

	devInfo, _, _ := procSetupDiGetClassDevsW.Call(uintptr(unsafe.Pointer(&hGUID)), 0, 0, digcfPI)
	if syscall.Handle(devInfo) == inv {
		fmt.Println("SetupDi failed")
		return
	}
	defer procSetupDiDestroyDeviceInfoList.Call(devInfo)

	var ifd spDevIfData
	ifd.CbSize = uint32(unsafe.Sizeof(ifd))

	for i := uint32(0); ; i++ {
		r, _, _ := procSetupDiEnumDeviceInterfaces.Call(devInfo, 0, uintptr(unsafe.Pointer(&hGUID)), uintptr(i), uintptr(unsafe.Pointer(&ifd)))
		if r == 0 {
			break
		}

		path := getDetailPath(devInfo, &ifd)
		if path == "" {
			continue
		}
		lp := strings.ToLower(path)
		if !strings.Contains(lp, "vid_0c45") || !strings.Contains(lp, "pid_800a") {
			continue
		}

		h := openPath(path)
		if h == inv {
			continue
		}

		var attrs hidAttrs
		attrs.Size = uint32(unsafe.Sizeof(attrs))
		procHidD_GetAttributes.Call(uintptr(h), uintptr(unsafe.Pointer(&attrs)))

		var pp uintptr
		var caps hidpCaps
		procHidD_GetPreparsedData.Call(uintptr(h), uintptr(unsafe.Pointer(&pp)))
		if pp != 0 {
			procHidP_GetCaps.Call(pp, uintptr(unsafe.Pointer(&caps)))
			procHidD_FreePreparsedData.Call(pp)
		}

		// Extract mi_XX from path
		mi := "?"
		if idx := strings.Index(lp, "&mi_"); idx >= 0 {
			mi = lp[idx+1 : idx+6]
		}

		fmt.Printf("--- Device %d ---\n", i)
		fmt.Printf("  Path: ...%s\n", path[max(0, len(path)-60):])
		fmt.Printf("  MI: %s  UsagePage: 0x%04X  Usage: 0x%04X\n", mi, caps.UsagePage, caps.Usage)
		fmt.Printf("  FeatReportLen: %d  InLen: %d  OutLen: %d\n", caps.FeatLen, caps.InLen, caps.OutLen)

		// Try sending 04 18 (begin) + readback
		if caps.FeatLen > 0 || true { // Try even if descriptor says 0
			buf := make([]byte, 65)
			buf[0] = 0x00 // report ID
			buf[1] = 0x04
			buf[2] = 0x18
			r, _, err := procHidD_SetFeature.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), 65)
			if r != 0 {
				fmt.Printf("  SetFeature(04 18): OK\n")
				// Try readback
				rbuf := make([]byte, 65)
				rbuf[0] = 0x00
				r2, _, _ := procHidD_GetFeature.Call(uintptr(h), uintptr(unsafe.Pointer(&rbuf[0])), 65)
				if r2 != 0 {
					fmt.Printf("  GetFeature: ACK=%02x %02x %02x %02x\n", rbuf[1], rbuf[2], rbuf[3], rbuf[4])
				} else {
					fmt.Printf("  GetFeature: FAILED\n")
				}
			} else {
				fmt.Printf("  SetFeature(04 18): FAILED (%v)\n", err)
			}
		}

		procCloseHandle.Call(uintptr(h))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func getDetailPath(devInfo uintptr, ifd *spDevIfData) string {
	var sz uint32
	procSetupDiGetDeviceInterfaceDetailW.Call(devInfo, uintptr(unsafe.Pointer(ifd)), 0, 0, uintptr(unsafe.Pointer(&sz)), 0)
	if sz == 0 {
		return ""
	}
	buf := make([]byte, sz)
	cb := uint32(8)
	if unsafe.Sizeof(uintptr(0)) == 4 {
		cb = 6
	}
	*(*uint32)(unsafe.Pointer(&buf[0])) = cb
	r, _, _ := procSetupDiGetDeviceInterfaceDetailW.Call(devInfo, uintptr(unsafe.Pointer(ifd)), uintptr(unsafe.Pointer(&buf[0])), uintptr(sz), 0, 0)
	if r == 0 {
		return ""
	}
	p := buf[4:]
	return syscall.UTF16ToString((*[1024]uint16)(unsafe.Pointer(&p[0]))[:])
}

func openPath(path string) syscall.Handle {
	p, _ := syscall.UTF16PtrFromString(path)
	r, _, _ := procCreateFileW.Call(uintptr(unsafe.Pointer(p)), 0xC0000000, 3, 0, 3, 0, 0)
	return syscall.Handle(r)
}
