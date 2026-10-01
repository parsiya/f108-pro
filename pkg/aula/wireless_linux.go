//go:build linux

package aula

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type WirelessDevice struct{ f *os.File }

// OpenWireless busca la interfaz vendor (usage page 0xFF60) del dongle 05ac:024f.
func OpenWireless() (*WirelessDevice, error) {
	nodes, _ := filepath.Glob("/sys/class/hidraw/hidraw*")
	for _, n := range nodes {
		ue, err := os.ReadFile(filepath.Join(n, "device/uevent"))
		if err != nil || !strings.Contains(string(ue), "HID_ID=0003:000005AC:0000024F") {
			continue
		}
		desc, err := os.ReadFile(filepath.Join(n, "device/report_descriptor"))
		if err != nil || !bytes.HasPrefix(desc, []byte{0x06, 0x60, 0xFF}) {
			continue
		}
		f, err := os.OpenFile("/dev/"+filepath.Base(n), os.O_RDWR, 0)
		if err != nil {
			return nil, err
		}
		return &WirelessDevice{f: f}, nil
	}
	return nil, errors.New("dongle 2.4G no encontrado")
}

func (d *WirelessDevice) Close() error { return d.f.Close() }

func (d *WirelessDevice) send(payload []byte) error {
	var pkt [33]byte // [0]=report ID, [1..31]=payload, [32]=checksum
	copy(pkt[1:32], payload)
	var sum byte
	for _, b := range pkt[:32] {
		sum += b
	}
	pkt[32] = sum

	buf := make([]byte, 64)
	for { // vaciar reportes pendientes
		d.f.SetReadDeadline(time.Now().Add(time.Millisecond))
		if _, err := d.f.Read(buf); err != nil {
			break
		}
	}
	var err error
	for try := 0; try < 2; try++ { // el driver original reintenta una vez
		if _, err = d.f.Write(pkt[:]); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		return err
	}
	for end := time.Now().Add(300 * time.Millisecond); time.Now().Before(end); {
		d.f.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
		if n, err := d.f.Read(buf); err == nil && n >= 3 && bytes.Equal(buf[:3], pkt[1:4]) {
			return nil
		}
	}
	return errors.New("sin ACK del teclado")
}

// SetKeyRemap envía un paquete por tecla. A diferencia del USB, las teclas
// que no están en la lista NO se tocan.
func (d *WirelessDevice) SetKeyRemap(remaps []KeyRemap, fn bool) error {
	cmd := byte(0x03)
	if fn {
		cmd = 0x0D
	}
	for _, r := range remaps {
		p := []byte{cmd, 0x04, r.SourceIndex, byte(r.Action), r.Param1, r.Param2, r.Param3}
		if err := d.send(p); err != nil {
			return fmt.Errorf("tecla %d: %w", r.SourceIndex, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// defaultNormalSlot es el slot "sin remap" de la capa normal: la tecla a sí misma.
func defaultNormalSlot(hid byte) [4]byte {
	if hid >= 0xE0 && hid <= 0xE7 { // modificadores: bit en vez de usage
		return [4]byte{0x02, 1 << (hid - 0xE0), 0x00, 0x00}
	}
	return [4]byte{0x02, 0x00, hid, 0x00}
}

// ResetKeys restaura teclas a su valor por defecto. Con indices vacío restaura
// todas las teclas conocidas, salvo key_index 86, que el driver trata aparte.
// Capa normal: la tecla a sí misma. Capa FN: slot en blanco (sin confirmar).
func (d *WirelessDevice) ResetKeys(indices []uint8, fn bool) error {
	hidByIndex := map[uint8]byte{}
	for name, idx := range KeyNameToIndex {
		if hid, ok := KeyNameToHID[name]; ok {
			hidByIndex[idx] = hid
		}
	}
	if len(indices) == 0 {
		for idx := range hidByIndex {
			if idx != 86 {
				indices = append(indices, idx)
			}
		}
		sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })
	}
	cmd := byte(0x03)
	if fn {
		cmd = 0x0D
	}
	for _, idx := range indices {
		var slot [4]byte
		if !fn {
			hid, ok := hidByIndex[idx]
			if !ok {
				return fmt.Errorf("la tecla %d no tiene código HID conocido", idx)
			}
			slot = defaultNormalSlot(hid)
		}
		p := []byte{cmd, 0x04, idx, slot[0], slot[1], slot[2], slot[3]}
		if err := d.send(p); err != nil {
			return fmt.Errorf("tecla %d: %w", idx, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}
