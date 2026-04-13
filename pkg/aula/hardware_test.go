package aula_test

import (
	"testing"
	"time"

	"github.com/parsiya/f108-pro/pkg/aula"
)

// These are hardware integration tests. They require a connected Aula F108 Pro
// keyboard. Run with: go test -v -count=1 ./pkg/aula/ -run TestHW
//
// Each test resets the keyboard to a default state (Rolling mode) at the end.

func openOrSkip(t *testing.T) *aula.Device {
	t.Helper()
	dev, err := aula.Open()
	if err != nil {
		t.Skipf("keyboard not available: %v", err)
	}
	return dev
}

func resetToDefault(t *testing.T, dev *aula.Device) {
	t.Helper()
	// Set all keys to cyan at full brightness.
	var keys []aula.KeyColor
	for _, idx := range aula.KeyNameToIndex {
		keys = append(keys, aula.KeyColor{LightIndex: idx, R: 0, G: 255, B: 255})
	}
	if err := dev.SetPerKeyRGB(keys, 5); err != nil {
		t.Errorf("reset to default failed: %v", err)
	}
}

// --- Lighting mode tests ---

func TestHW_LightStaticRed(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	err := dev.SetLighting(aula.LightingConfig{
		Mode:       aula.ModeStatic,
		R:          255,
		Brightness: 5,
	})
	if err != nil {
		t.Fatalf("SetLighting static red: %v", err)
	}
	time.Sleep(2 * time.Second)
}

func TestHW_LightStaticGreen(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	err := dev.SetLighting(aula.LightingConfig{
		Mode:       aula.ModeStatic,
		G:          255,
		Brightness: 5,
	})
	if err != nil {
		t.Fatalf("SetLighting static green: %v", err)
	}
	time.Sleep(2 * time.Second)
}

func TestHW_LightStaticBlue(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	err := dev.SetLighting(aula.LightingConfig{
		Mode:       aula.ModeStatic,
		B:          255,
		Brightness: 5,
	})
	if err != nil {
		t.Fatalf("SetLighting static blue: %v", err)
	}
	time.Sleep(2 * time.Second)
}

func TestHW_LightStaticWhite(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	err := dev.SetLighting(aula.LightingConfig{
		Mode:       aula.ModeStatic,
		R:          255,
		G:          255,
		B:          255,
		Brightness: 5,
	})
	if err != nil {
		t.Fatalf("SetLighting static white: %v", err)
	}
	time.Sleep(2 * time.Second)
}

func TestHW_LightBreathPurple(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	err := dev.SetLighting(aula.LightingConfig{
		Mode:       aula.ModeBreath,
		R:          128,
		B:          255,
		Brightness: 5,
		Speed:      3,
	})
	if err != nil {
		t.Fatalf("SetLighting breath purple: %v", err)
	}
	time.Sleep(3 * time.Second)
}

func TestHW_LightSpectrum(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	err := dev.SetLighting(aula.LightingConfig{
		Mode:       aula.ModeSpectrum,
		Brightness: 5,
		Speed:      4,
	})
	if err != nil {
		t.Fatalf("SetLighting spectrum: %v", err)
	}
	time.Sleep(3 * time.Second)
}

func TestHW_LightRollingColorful(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	err := dev.SetLighting(aula.LightingConfig{
		Mode:       aula.ModeRolling,
		Brightness: 5,
		Speed:      4,
		Colorful:   true,
	})
	if err != nil {
		t.Fatalf("SetLighting rolling colorful: %v", err)
	}
	time.Sleep(3 * time.Second)
}

func TestHW_LightScrollingWithDirection(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	err := dev.SetLighting(aula.LightingConfig{
		Mode:       aula.ModeScrolling,
		Brightness: 5,
		Speed:      3,
		Direction:  1,
		Colorful:   true,
	})
	if err != nil {
		t.Fatalf("SetLighting scrolling direction=1: %v", err)
	}
	time.Sleep(3 * time.Second)
}

func TestHW_LightDirectionModes(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	// Modes that support direction.
	modes := []struct {
		mode aula.LightingMode
		name string
	}{
		{aula.ModeScrolling, "Scrolling"},
		{aula.ModeRolling, "Rolling"},
		{aula.ModeRotating, "Rotating"},
		{aula.ModeFlowing, "Flowing"},
		{aula.ModeTilt, "Tilt"},
	}

	for _, m := range modes {
		for dir := uint8(0); dir <= 1; dir++ {
			t.Logf("%s direction=%d", m.name, dir)
			err := dev.SetLighting(aula.LightingConfig{
				Mode:       m.mode,
				Brightness: 5,
				Speed:      3,
				Direction:  dir,
				Colorful:   true,
			})
			if err != nil {
				t.Fatalf("SetLighting %s direction=%d: %v", m.name, dir, err)
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func TestHW_LightOff(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	err := dev.SetLighting(aula.LightingConfig{Mode: aula.ModeOff})
	if err != nil {
		t.Fatalf("SetLighting off: %v", err)
	}
	time.Sleep(2 * time.Second)
}

func TestHW_LightBrightnessLevels(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	for b := uint8(0); b <= 5; b++ {
		err := dev.SetLighting(aula.LightingConfig{
			Mode:       aula.ModeStatic,
			R:          255,
			G:          100,
			Brightness: b,
		})
		if err != nil {
			t.Fatalf("SetLighting brightness=%d: %v", b, err)
		}
		time.Sleep(1 * time.Second)
	}
}

func TestHW_LightSpeedLevels(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	for s := uint8(0); s <= 5; s++ {
		err := dev.SetLighting(aula.LightingConfig{
			Mode:       aula.ModeBreath,
			R:          0,
			G:          255,
			B:          0,
			Brightness: 5,
			Speed:      s,
		})
		if err != nil {
			t.Fatalf("SetLighting speed=%d: %v", s, err)
		}
		time.Sleep(2 * time.Second)
	}
}

func TestHW_LightAllModes(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	for mode := aula.LightingMode(0); mode <= 19; mode++ {
		name := aula.ModeNames[mode]
		t.Logf("mode %d: %s", mode, name)
		err := dev.SetLighting(aula.LightingConfig{
			Mode:       mode,
			R:          0,
			G:          200,
			B:          255,
			Brightness: 5,
			Speed:      3,
			Colorful:   true,
		})
		if err != nil {
			t.Fatalf("SetLighting mode=%d (%s): %v", mode, name, err)
		}
		time.Sleep(2 * time.Second)
	}
}

// --- Per-key RGB tests ---

func TestHW_PerKeyWASDGreen(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	err := dev.SetPerKeyRGB([]aula.KeyColor{
		{LightIndex: aula.KeyW, R: 0, G: 255, B: 0},
		{LightIndex: aula.KeyA, R: 0, G: 255, B: 0},
		{LightIndex: aula.KeyS, R: 0, G: 255, B: 0},
		{LightIndex: aula.KeyD, R: 0, G: 255, B: 0},
	}, 5)
	if err != nil {
		t.Fatalf("SetPerKeyRGB WASD green: %v", err)
	}
	time.Sleep(2 * time.Second)
}

func TestHW_PerKeyEscRed(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	err := dev.SetPerKeyRGB([]aula.KeyColor{
		{LightIndex: aula.KeyEsc, R: 255, G: 0, B: 0},
	}, 5)
	if err != nil {
		t.Fatalf("SetPerKeyRGB esc red: %v", err)
	}
	time.Sleep(2 * time.Second)
}

func TestHW_PerKeyFunctionRowRainbow(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	// Rainbow across F1-F12.
	fkeys := []uint8{
		aula.KeyF1, aula.KeyF2, aula.KeyF3, aula.KeyF4,
		aula.KeyF5, aula.KeyF6, aula.KeyF7, aula.KeyF8,
		aula.KeyF9, aula.KeyF10, aula.KeyF11, aula.KeyF12,
	}
	colors := [][3]uint8{
		{255, 0, 0}, {255, 127, 0}, {255, 255, 0}, {127, 255, 0},
		{0, 255, 0}, {0, 255, 127}, {0, 255, 255}, {0, 127, 255},
		{0, 0, 255}, {127, 0, 255}, {255, 0, 255}, {255, 0, 127},
	}

	var keys []aula.KeyColor
	for i, idx := range fkeys {
		keys = append(keys, aula.KeyColor{
			LightIndex: idx,
			R:          colors[i][0],
			G:          colors[i][1],
			B:          colors[i][2],
		})
	}

	err := dev.SetPerKeyRGB(keys, 5)
	if err != nil {
		t.Fatalf("SetPerKeyRGB F-row rainbow: %v", err)
	}
	time.Sleep(3 * time.Second)
}

func TestHW_PerKeyAllWhite(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	var keys []aula.KeyColor
	for _, idx := range aula.KeyNameToIndex {
		keys = append(keys, aula.KeyColor{
			LightIndex: idx, R: 255, G: 255, B: 255,
		})
	}

	err := dev.SetPerKeyRGB(keys, 5)
	if err != nil {
		t.Fatalf("SetPerKeyRGB all white: %v", err)
	}
	time.Sleep(2 * time.Second)
}

func TestHW_PerKeyAllOff(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()
	defer resetToDefault(t, dev)

	// Empty list = all keys off.
	err := dev.SetPerKeyRGB(nil, 5)
	if err != nil {
		t.Fatalf("SetPerKeyRGB all off: %v", err)
	}
	time.Sleep(2 * time.Second)
}

// --- Clock sync test ---

func TestHW_ClockSync(t *testing.T) {
	dev := openOrSkip(t)
	defer dev.Close()

	now := time.Now()
	err := dev.SyncClock(now)
	if err != nil {
		t.Fatalf("SyncClock: %v", err)
	}
	t.Logf("clock synced to %s", now.Format("2006-01-02 15:04:05"))
}
