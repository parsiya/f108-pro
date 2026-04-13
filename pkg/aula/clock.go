package aula

import (
	"fmt"
	"time"
)

// SyncClock sets the keyboard's LCD clock to the given time.
// Sequence: begin -> clock init (04 28) -> data -> apply.
//
// dev.SyncClock(time.Now())
func (d *Device) SyncClock(t time.Time) error {
	// Step 1: Begin.
	if err := d.beginTransaction(); err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	// Step 2: Clock init (04 28, byte[8]=01).
	initPayload := make([]byte, 64)
	initPayload[0] = 0x04
	initPayload[1] = 0x28
	initPayload[8] = 0x01
	if err := d.sendCommand(initPayload, true); err != nil {
		return fmt.Errorf("clock init: %w", err)
	}

	// Step 3: Clock data.
	data := make([]byte, 64)
	data[0] = 0x00                    // Zero.
	data[1] = 0x01                    // Profile 1.
	data[2] = 0x5A                    // Magic marker.
	data[3] = byte(t.Year() % 2000)   // Year (e.g., 26 for 2026).
	data[4] = byte(t.Month())         // Month (1-12).
	data[5] = byte(t.Day())           // Day (1-31).
	data[6] = byte(t.Hour())          // Hour (0-23).
	data[7] = byte(t.Minute())        // Minute (0-59).
	data[8] = byte(t.Second())        // Second (0-59).
	data[10] = byte(t.Weekday())      // Day of week (0=Sun).
	data[62] = 0x55                   // Trailer.
	data[63] = 0xAA
	if err := d.sendCommand(data, true); err != nil {
		return fmt.Errorf("clock data: %w", err)
	}

	// Step 4: Apply (no finalize for clock sync).
	if err := d.applyTransaction(); err != nil {
		return fmt.Errorf("apply: %w", err)
	}

	return nil
}
