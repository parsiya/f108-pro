package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/parsiya/f108-pro/pkg/aula"
)

// --- buildKeysFromLayout tests ---

func TestBuildKeysFromLayout_BaseOnly(t *testing.T) {
	base := [3]uint8{0, 0, 50}
	keys := buildKeysFromLayout(base, nil)

	if len(keys) != len(aula.KeyNameToIndex) {
		t.Fatalf("expected %d keys, got %d", len(aula.KeyNameToIndex), len(keys))
	}

	for _, k := range keys {
		if k.R != 0 || k.G != 0 || k.B != 50 {
			t.Fatalf("key idx %d: expected (0,0,50), got (%d,%d,%d)", k.LightIndex, k.R, k.G, k.B)
		}
	}
}

func TestBuildKeysFromLayout_OverridesOnly(t *testing.T) {
	overrides := map[string][3]uint8{
		"esc": {255, 0, 0},
		"w":   {0, 255, 0},
	}
	keys := buildKeysFromLayout([3]uint8{0, 0, 0}, overrides)

	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	byIndex := map[uint8]aula.KeyColor{}
	for _, k := range keys {
		byIndex[k.LightIndex] = k
	}

	escIdx := aula.KeyNameToIndex["esc"]
	if k, ok := byIndex[escIdx]; !ok {
		t.Fatal("esc not in output")
	} else if k.R != 255 || k.G != 0 || k.B != 0 {
		t.Fatalf("esc: expected (255,0,0), got (%d,%d,%d)", k.R, k.G, k.B)
	}

	wIdx := aula.KeyNameToIndex["w"]
	if k, ok := byIndex[wIdx]; !ok {
		t.Fatal("w not in output")
	} else if k.R != 0 || k.G != 255 || k.B != 0 {
		t.Fatalf("w: expected (0,255,0), got (%d,%d,%d)", k.R, k.G, k.B)
	}
}

func TestBuildKeysFromLayout_BaseWithOverrides(t *testing.T) {
	base := [3]uint8{0, 0, 50}
	overrides := map[string][3]uint8{
		"esc": {255, 0, 0},
	}
	keys := buildKeysFromLayout(base, overrides)

	// Should have all keys (base applied to all).
	if len(keys) != len(aula.KeyNameToIndex) {
		t.Fatalf("expected %d keys, got %d", len(aula.KeyNameToIndex), len(keys))
	}

	byIndex := map[uint8]aula.KeyColor{}
	for _, k := range keys {
		byIndex[k.LightIndex] = k
	}

	// Esc should be overridden.
	escIdx := aula.KeyNameToIndex["esc"]
	if k := byIndex[escIdx]; k.R != 255 || k.G != 0 || k.B != 0 {
		t.Fatalf("esc: expected (255,0,0), got (%d,%d,%d)", k.R, k.G, k.B)
	}

	// A random non-overridden key should have base color.
	spaceIdx := aula.KeyNameToIndex["space"]
	if k := byIndex[spaceIdx]; k.R != 0 || k.G != 0 || k.B != 50 {
		t.Fatalf("space: expected (0,0,50), got (%d,%d,%d)", k.R, k.G, k.B)
	}
}

func TestBuildKeysFromLayout_EmptyBase_EmptyOverrides(t *testing.T) {
	keys := buildKeysFromLayout([3]uint8{0, 0, 0}, nil)
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys, got %d", len(keys))
	}
}

// --- loadPerKeyYAML tests ---

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadPerKeyYAML_AllOnly(t *testing.T) {
	path := writeTemp(t, "test.yaml", `
all: [255, 0, 0]
`)
	keys, brightness := loadPerKeyYAML(path)

	if len(keys) != len(aula.KeyNameToIndex) {
		t.Fatalf("expected %d keys, got %d", len(aula.KeyNameToIndex), len(keys))
	}
	for _, k := range keys {
		if k.R != 255 || k.G != 0 || k.B != 0 {
			t.Fatalf("key idx %d: expected (255,0,0), got (%d,%d,%d)", k.LightIndex, k.R, k.G, k.B)
		}
	}
	if brightness != nil {
		t.Fatalf("expected nil brightness, got %d", *brightness)
	}
}

func TestLoadPerKeyYAML_WithBrightness(t *testing.T) {
	path := writeTemp(t, "test.yaml", `
all: [0, 0, 50]
brightness: 3
keys:
  w: [0, 255, 0]
`)
	keys, brightness := loadPerKeyYAML(path)

	if len(keys) != len(aula.KeyNameToIndex) {
		t.Fatalf("expected %d keys, got %d", len(aula.KeyNameToIndex), len(keys))
	}
	if brightness == nil || *brightness != 3 {
		t.Fatalf("expected brightness=3, got %v", brightness)
	}

	byIndex := map[uint8]aula.KeyColor{}
	for _, k := range keys {
		byIndex[k.LightIndex] = k
	}
	wIdx := aula.KeyNameToIndex["w"]
	if k := byIndex[wIdx]; k.R != 0 || k.G != 255 || k.B != 0 {
		t.Fatalf("w: expected (0,255,0), got (%d,%d,%d)", k.R, k.G, k.B)
	}
}

func TestLoadPerKeyYAML_KeysOnly(t *testing.T) {
	path := writeTemp(t, "test.yaml", `
keys:
  esc: [255, 0, 0]
  f1: [0, 255, 0]
`)
	keys, brightness := loadPerKeyYAML(path)

	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if brightness != nil {
		t.Fatalf("expected nil brightness, got %d", *brightness)
	}
}

func TestLoadPerKeyYAML_ExampleFile(t *testing.T) {
	// Test the actual example file shipped with the repo.
	// Tests run with cwd = the package directory (cmd/aula/).
	path := filepath.Join("..", "..", "examples", "wasd-green.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("examples/wasd-green.yaml not found")
	}

	keys, _ := loadPerKeyYAML(path)

	if len(keys) != len(aula.KeyNameToIndex) {
		t.Fatalf("expected %d keys (has 'all' base), got %d", len(aula.KeyNameToIndex), len(keys))
	}

	byIndex := map[uint8]aula.KeyColor{}
	for _, k := range keys {
		byIndex[k.LightIndex] = k
	}

	// W should be green.
	wIdx := aula.KeyNameToIndex["w"]
	if k := byIndex[wIdx]; k.R != 0 || k.G != 255 || k.B != 0 {
		t.Fatalf("w: expected (0,255,0), got (%d,%d,%d)", k.R, k.G, k.B)
	}

	// A non-WASD key should have the base color (0,0,50).
	escIdx := aula.KeyNameToIndex["esc"]
	if k := byIndex[escIdx]; k.R != 0 || k.G != 0 || k.B != 50 {
		t.Fatalf("esc: expected (0,0,50), got (%d,%d,%d)", k.R, k.G, k.B)
	}
}

// --- loadRemapYAML tests ---

func TestLoadRemapYAML_NormalLayer(t *testing.T) {
	path := writeTemp(t, "remap.yaml", `
layer: normal
keys:
  capslock: lctrl
  esc: grave
`)
	remaps, fnLayer, err := loadRemapYAML(path)
	if err != nil {
		t.Fatalf("loadRemapYAML: %v", err)
	}
	if fnLayer {
		t.Fatal("expected normal layer, got fn")
	}
	if len(remaps) != 2 {
		t.Fatalf("expected 2 remaps, got %d", len(remaps))
	}

	// Verify capslock -> lctrl.
	capIdx := aula.KeyNameToIndex["capslock"]
	found := false
	for _, r := range remaps {
		if r.SourceIndex == capIdx {
			if r.Action != aula.RemapKey {
				t.Fatalf("capslock: expected action RemapKey(0x02), got 0x%02x", r.Action)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("capslock remap not found")
	}
}

func TestLoadRemapYAML_FnLayer(t *testing.T) {
	path := writeTemp(t, "remap.yaml", `
layer: fn
keys:
  f1: media:play
`)
	remaps, fnLayer, err := loadRemapYAML(path)
	if err != nil {
		t.Fatalf("loadRemapYAML: %v", err)
	}
	if !fnLayer {
		t.Fatal("expected fn layer")
	}
	if len(remaps) != 1 {
		t.Fatalf("expected 1 remap, got %d", len(remaps))
	}
	if remaps[0].Action != aula.RemapConsumer {
		t.Fatalf("expected RemapConsumer(0x03), got 0x%02x", remaps[0].Action)
	}
}

func TestLoadRemapYAML_Combo(t *testing.T) {
	path := writeTemp(t, "remap.yaml", `
keys:
  f4: combo:ctrl+c
`)
	remaps, _, err := loadRemapYAML(path)
	if err != nil {
		t.Fatalf("loadRemapYAML: %v", err)
	}
	if len(remaps) != 1 {
		t.Fatalf("expected 1 remap, got %d", len(remaps))
	}
	r := remaps[0]
	if r.Action != aula.RemapKey {
		t.Fatalf("expected RemapKey, got 0x%02x", r.Action)
	}
	// Param1 should be ctrl modifier (0x01), Param2 should be HID for 'c'.
	if r.Param1 != 0x01 {
		t.Fatalf("expected modifier 0x01 (ctrl), got 0x%02x", r.Param1)
	}
	cHID := aula.KeyNameToHID["c"]
	if r.Param2 != cHID {
		t.Fatalf("expected HID 0x%02x (c), got 0x%02x", cHID, r.Param2)
	}
}

func TestLoadRemapYAML_Mouse(t *testing.T) {
	path := writeTemp(t, "remap.yaml", `
keys:
  pause: mouse:lclick
`)
	remaps, _, err := loadRemapYAML(path)
	if err != nil {
		t.Fatalf("loadRemapYAML: %v", err)
	}
	if len(remaps) != 1 {
		t.Fatalf("expected 1 remap, got %d", len(remaps))
	}
	if remaps[0].Action != aula.RemapMouse {
		t.Fatalf("expected RemapMouse(0x07), got 0x%02x", remaps[0].Action)
	}
}

func TestLoadRemapYAML_UnknownKey(t *testing.T) {
	path := writeTemp(t, "remap.yaml", `
keys:
  nonexistent: esc
`)
	_, _, err := loadRemapYAML(path)
	if err == nil {
		t.Fatal("expected error for unknown source key")
	}
}

func TestLoadRemapYAML_UnknownTarget(t *testing.T) {
	path := writeTemp(t, "remap.yaml", `
keys:
  esc: nonexistent
`)
	_, _, err := loadRemapYAML(path)
	if err == nil {
		t.Fatal("expected error for unknown target key")
	}
}

func TestLoadRemapYAML_EmptyKeys(t *testing.T) {
	path := writeTemp(t, "remap.yaml", `
layer: normal
keys:
`)
	_, _, err := loadRemapYAML(path)
	if err == nil {
		t.Fatal("expected error for empty keys")
	}
}

func TestLoadRemapYAML_ExampleFile(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "capslock-ctrl.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("examples/capslock-ctrl.yaml not found")
	}

	remaps, fnLayer, err := loadRemapYAML(path)
	if err != nil {
		t.Fatalf("loadRemapYAML example: %v", err)
	}
	if fnLayer {
		t.Fatal("capslock-ctrl.yaml should be normal layer")
	}
	if len(remaps) == 0 {
		t.Fatal("expected at least 1 remap from example file")
	}
}

// --- parseComboExpr tests ---

func TestParseComboExpr_CtrlC(t *testing.T) {
	r, err := parseComboExpr(55, "ctrl+c")
	if err != nil {
		t.Fatal(err)
	}
	if r.Action != aula.RemapKey {
		t.Fatalf("expected RemapKey, got 0x%02x", r.Action)
	}
	if r.Param1 != 0x01 {
		t.Fatalf("expected modifier 0x01, got 0x%02x", r.Param1)
	}
}

func TestParseComboExpr_MultiMod(t *testing.T) {
	r, err := parseComboExpr(55, "ctrl+shift+a")
	if err != nil {
		t.Fatal(err)
	}
	if r.Param1 != 0x03 { // ctrl(0x01) | shift(0x02)
		t.Fatalf("expected modifier 0x03, got 0x%02x", r.Param1)
	}
}

func TestParseComboExpr_RightMods(t *testing.T) {
	r, err := parseComboExpr(55, "rctrl+rshift+a")
	if err != nil {
		t.Fatal(err)
	}
	if r.Param1 != 0x30 { // rctrl(0x10) | rshift(0x20)
		t.Fatalf("expected modifier 0x30, got 0x%02x", r.Param1)
	}
}

func TestParseComboExpr_NoKey(t *testing.T) {
	_, err := parseComboExpr(55, "ctrl")
	if err == nil {
		t.Fatal("expected error for combo without key")
	}
}

func TestParseComboExpr_BadModifier(t *testing.T) {
	_, err := parseComboExpr(55, "superkey+a")
	if err == nil {
		t.Fatal("expected error for unknown modifier")
	}
}

// --- parseRemapTarget tests ---

func TestParseRemapTarget_KeySwap(t *testing.T) {
	r, err := parseRemapTarget(aula.KeyNameToIndex["esc"], "grave")
	if err != nil {
		t.Fatal(err)
	}
	if r.Action != aula.RemapKey {
		t.Fatalf("expected RemapKey, got 0x%02x", r.Action)
	}
}

func TestParseRemapTarget_Media(t *testing.T) {
	actions := []string{"play", "stop", "prev", "next", "volup", "voldown", "mute"}
	for _, a := range actions {
		r, err := parseRemapTarget(1, "media:"+a)
		if err != nil {
			t.Fatalf("media:%s: %v", a, err)
		}
		if r.Action != aula.RemapConsumer {
			t.Fatalf("media:%s: expected RemapConsumer, got 0x%02x", a, r.Action)
		}
	}
}

func TestParseRemapTarget_Mouse(t *testing.T) {
	actions := []string{"lclick", "rclick", "mclick", "scrollup", "scrolldn"}
	for _, a := range actions {
		r, err := parseRemapTarget(1, "mouse:"+a)
		if err != nil {
			t.Fatalf("mouse:%s: %v", a, err)
		}
		if r.Action != aula.RemapMouse {
			t.Fatalf("mouse:%s: expected RemapMouse, got 0x%02x", a, r.Action)
		}
	}
}

func TestParseRemapTarget_Combo(t *testing.T) {
	r, err := parseRemapTarget(1, "combo:win+d")
	if err != nil {
		t.Fatal(err)
	}
	if r.Action != aula.RemapKey {
		t.Fatalf("expected RemapKey, got 0x%02x", r.Action)
	}
	if r.Param1 != 0x08 { // win
		t.Fatalf("expected modifier 0x08 (win), got 0x%02x", r.Param1)
	}
}

func TestParseRemapTarget_UnknownKey(t *testing.T) {
	_, err := parseRemapTarget(1, "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestParseRemapTarget_UnknownMedia(t *testing.T) {
	_, err := parseRemapTarget(1, "media:rewind")
	if err == nil {
		t.Fatal("expected error for unknown media action")
	}
}

func TestParseRemapTarget_UnknownMouse(t *testing.T) {
	_, err := parseRemapTarget(1, "mouse:doubleclick")
	if err == nil {
		t.Fatal("expected error for unknown mouse action")
	}
}

// keep sort import used.
var _ = sort.Strings
