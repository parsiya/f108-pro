package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/parsiya/f108-pro/pkg/aula"
	"gopkg.in/yaml.v3"
)

// CLI defines the top-level command structure.
var CLI struct {
	Light      LightCmd      `cmd:"" help:"Set backlight mode."`
	Brightness BrightnessCmd `cmd:"" help:"Set brightness without changing mode."`
	Off        OffCmd        `cmd:"" help:"Turn off backlight."`
	Modes      ModesCmd      `cmd:"" help:"List available lighting modes."`
	Clock      ClockCmd      `cmd:"" help:"Sync LCD clock to system time."`
	Perkey     PerkeyCmd     `cmd:"" help:"Set per-key RGB colors."`
	Keys       KeysCmd       `cmd:"" help:"List available key names."`
	LCD        LCDCmd        `cmd:"" name:"lcd" help:"Upload image to LCD screen."`
	Remap      RemapCmd      `cmd:"" help:"Remap keys (supports key swap, media, mouse, combos)."`
}

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("f108-pro"),
		kong.Description("Aula F108 Pro keyboard configurator."),
		kong.UsageOnError(),
	)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

// LightCmd sets the keyboard backlight mode.
type LightCmd struct {
	Direction *uint8   `flag:"" short:"d" help:"Direction: 0-3 (modes: Scrolling, Rolling, Rotating, Flowing, Tilt)."`
	Args      []string `arg:"" optional:"" help:"<mode> [brightness] [speed] [r g b | colorful]"`
}

// Run executes the light command.
func (c *LightCmd) Run() error {
	if len(c.Args) < 1 {
		return fmt.Errorf("mode is required (0-19 or name, run 'aula modes' to list)")
	}

	mode := parseMode(c.Args[0])

	cfg := aula.LightingConfig{
		Mode:       mode,
		Brightness: 5,
		Speed:      3,
	}

	if len(c.Args) >= 2 {
		cfg.Brightness = parseByte(c.Args[1], "brightness")
	}
	if len(c.Args) >= 3 {
		cfg.Speed = parseByte(c.Args[2], "speed")
	}
	if len(c.Args) >= 4 {
		if strings.EqualFold(c.Args[3], "colorful") || strings.EqualFold(c.Args[3], "rainbow") {
			cfg.Colorful = true
		} else {
			cfg.R = parseByte(c.Args[3], "red")
			if len(c.Args) >= 5 {
				cfg.G = parseByte(c.Args[4], "green")
			}
			if len(c.Args) >= 6 {
				cfg.B = parseByte(c.Args[5], "blue")
			}
		}
	}

	if c.Direction != nil {
		cfg.Direction = *c.Direction
	}

	dev, err := aula.Open()
	if err != nil {
		return err
	}
	defer dev.Close()

	modeName := aula.ModeNames[cfg.Mode]
	fmt.Printf("Setting: mode=%s(%d) brightness=%d speed=%d direction=%d",
		modeName, cfg.Mode, cfg.Brightness, cfg.Speed, cfg.Direction)
	if cfg.Colorful {
		fmt.Print(" color=rainbow")
	} else {
		fmt.Printf(" color=(%d,%d,%d)", cfg.R, cfg.G, cfg.B)
	}
	fmt.Println()

	if err := dev.SetLighting(cfg); err != nil {
		return err
	}
	fmt.Println("OK")
	return nil
}

// OffCmd turns off the backlight.
type OffCmd struct{}

// Run executes the off command.
func (c *OffCmd) Run() error {
	dev, err := aula.Open()
	if err != nil {
		return err
	}
	defer dev.Close()

	if err := dev.SetLighting(aula.LightingConfig{Mode: aula.ModeOff}); err != nil {
		return err
	}
	fmt.Println("Backlight off.")
	return nil
}

// BrightnessCmd sets brightness without changing the current mode.
// Since the keyboard doesn't support reading current state, this re-sends
// the specified mode (default: static white) at the given brightness.
type BrightnessCmd struct {
	Level uint8  `arg:"" help:"Brightness level (0-5)."`
	Mode  string `flag:"" short:"m" default:"static" help:"Mode to use (default: static)."`
}

// Run executes the brightness command.
func (c *BrightnessCmd) Run() error {
	mode := parseMode(c.Mode)

	dev, err := aula.Open()
	if err != nil {
		return err
	}
	defer dev.Close()

	cfg := aula.LightingConfig{
		Mode:       mode,
		R:          255,
		G:          255,
		B:          255,
		Brightness: c.Level,
		Speed:      3,
		Colorful:   true,
	}
	if err := dev.SetLighting(cfg); err != nil {
		return err
	}
	fmt.Printf("Brightness set to %d (mode=%s)\n", c.Level, aula.ModeNames[mode])
	return nil
}

// ModesCmd lists all available lighting modes.
type ModesCmd struct{}

// Run executes the modes command.
func (c *ModesCmd) Run() error {
	fmt.Println("Available lighting modes:")
	fmt.Println()
	for i := aula.LightingMode(0); i <= 19; i++ {
		name := aula.ModeNames[i]
		fmt.Printf("  %2d  %s\n", i, name)
	}
	fmt.Println()
	fmt.Println("Modes that support direction: Scrolling(10), Rolling(11),")
	fmt.Println("Rotating(12), Flowing(16), Tilt(18)")
	fmt.Println()
	fmt.Println("Modes with forced rainbow (no color picker): Colourful(6), Spectrum(8)")
	return nil
}

// ClockCmd syncs the LCD clock to the system time.
type ClockCmd struct{}

// Run executes the clock command.
func (c *ClockCmd) Run() error {
	dev, err := aula.Open()
	if err != nil {
		return err
	}
	defer dev.Close()

	now := time.Now()
	if err := dev.SyncClock(now); err != nil {
		return err
	}
	fmt.Printf("Clock synced to %s\n", now.Format("2006-01-02 15:04:05"))
	return nil
}

// LCDCmd uploads a raw image file to the keyboard's LCD screen.
type LCDCmd struct {
	File string `arg:"" help:"Raw image file to upload (generate with mkimage)."`
}

// Run executes the lcd command.
func (c *LCDCmd) Run() error {
	fmt.Fprintln(os.Stderr, "WARNING: Uploading images to the LCD can permanently corrupt the")
	fmt.Fprintln(os.Stderr, "keyboard's built-in menu graphics if the frame limit is exceeded.")
	fmt.Fprintln(os.Stderr, "This damage is NOT recoverable by factory reset or firmware update.")
	fmt.Fprint(os.Stderr, "Continue? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println("Aborted.")
		return nil
	}

	buf, err := os.ReadFile(c.File)
	if err != nil {
		return fmt.Errorf("reading %s: %w", c.File, err)
	}

	// Safety check: enforce 141-frame limit to prevent SPI flash overflow.
	const headerSize = 256
	const frameSize = 240 * 135 * 2 // RGB565.
	const maxFrames = 141
	if len(buf) > headerSize {
		frames := (len(buf) - headerSize) / frameSize
		if frames > maxFrames {
			return fmt.Errorf("%d frames exceeds safe limit of %d, upload blocked", frames, maxFrames)
		}
	}

	fmt.Printf("Uploading %s (%d bytes, %d pages)\n", c.File, len(buf), len(buf)/4096)

	dev, err := aula.Open()
	if err != nil {
		return err
	}
	defer dev.Close()

	progress := func(sent, total int) {
		fmt.Printf("\r  Page %d/%d", sent, total)
	}

	if err := dev.UploadLCDImage(buf, 1, progress); err != nil {
		return err
	}
	fmt.Println("\nLCD image uploaded.")
	return nil
}

// KeysCmd lists all available key names.
type KeysCmd struct{}

// Run executes the keys command.
func (c *KeysCmd) Run() error {
	fmt.Println("Available key names:")
	fmt.Println()

	rows := []struct {
		label string
		keys  []string
	}{
		{"Function row", []string{"esc", "f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12", "printscreen", "scrolllock", "pause"}},
		{"Number row", []string{"grave", "1", "2", "3", "4", "5", "6", "7", "8", "9", "0", "minus", "equal", "backspace", "insert", "home", "pageup", "numlock", "numslash", "numstar", "numminus"}},
		{"QWERTY row", []string{"tab", "q", "w", "e", "r", "t", "y", "u", "i", "o", "p", "lbracket", "rbracket", "backslash", "delete", "end", "pagedown", "num7", "num8", "num9", "numplus"}},
		{"Home row", []string{"capslock", "a", "s", "d", "f", "g", "h", "j", "k", "l", "semicolon", "quote", "enter", "num4", "num5", "num6"}},
		{"Shift row", []string{"lshift", "z", "x", "c", "v", "b", "n", "m", "comma", "dot", "slash", "rshift", "up", "num1", "num2", "num3", "numenter"}},
		{"Bottom row", []string{"lctrl", "lwin", "lalt", "space", "ralt", "fn", "menu", "rctrl", "left", "down", "right", "num0", "numdot"}},
	}
	for _, row := range rows {
		fmt.Printf("  %s: %s\n", row.label, strings.Join(row.keys, ", "))
	}
	return nil
}

// PerkeyCmd sets individual key colors.
type PerkeyCmd struct {
	All        []uint8  `flag:"" help:"Base color for all keys (r g b)." placeholder:"r g b"`
	Brightness uint8    `flag:"" short:"b" default:"5" help:"Overall brightness (0-5)."`
	Args       []string `arg:"" optional:"" help:"<layout.yaml> or <key> <r> <g> <b> ..."`
}

// Run executes the perkey command.
func (c *PerkeyCmd) Run() error {
	if len(c.Args) < 1 && len(c.All) == 0 {
		return fmt.Errorf("provide a YAML file, --all r g b, or key r g b groups (run 'aula keys' to list)")
	}

	var keys []aula.KeyColor
	brightness := c.Brightness

	// Check if first arg is a YAML file.
	if len(c.Args) == 1 && (strings.HasSuffix(c.Args[0], ".yaml") || strings.HasSuffix(c.Args[0], ".yml")) {
		var yamlBrightness *uint8
		keys, yamlBrightness = loadPerKeyYAML(c.Args[0])
		// YAML brightness is used only if --brightness was not explicitly set.
		if yamlBrightness != nil && c.Brightness == 5 {
			brightness = *yamlBrightness
		}
	} else {
		keys = parsePerKeyArgs(c.All, c.Args)
	}

	dev, err := aula.Open()
	if err != nil {
		return err
	}
	defer dev.Close()

	fmt.Printf("Setting %d key(s) brightness=%d\n", len(keys), brightness)

	if err := dev.SetPerKeyRGB(keys, brightness); err != nil {
		return err
	}
	fmt.Println("OK")
	return nil
}

// RemapCmd remaps keys on the normal or FN layer.
type RemapCmd struct {
	FN    bool     `flag:"" help:"Target FN layer instead of normal layer."`
	Reset bool     `flag:"" help:"Clear all key remaps on the target layer."`
	Args  []string `arg:"" optional:"" help:"<src> <dst> pairs, or a YAML file path."`
}

// Run executes the remap command.
func (c *RemapCmd) Run() error {
	if c.Reset {
		dev, err := aula.Open()
		if err != nil {
			return err
		}
		defer dev.Close()

		if c.FN {
			if err := dev.ResetFnKeyRemap(); err != nil {
				return err
			}
			fmt.Println("All FN layer remaps cleared.")
		} else {
			if err := dev.ResetKeyRemap(); err != nil {
				return err
			}
			fmt.Println("All key remaps cleared.")
		}
		return nil
	}

	if len(c.Args) < 1 {
		return fmt.Errorf("provide src dst pairs or a YAML file (run 'aula keys' to list key names)")
	}

	var remaps []aula.KeyRemap
	var fnLayer bool

	// Single arg ending in .yaml/.yml -> load from file.
	if len(c.Args) == 1 && (strings.HasSuffix(c.Args[0], ".yaml") || strings.HasSuffix(c.Args[0], ".yml")) {
		var err error
		remaps, fnLayer, err = loadRemapYAML(c.Args[0])
		if err != nil {
			return err
		}
	} else {
		fnLayer = c.FN
		if len(c.Args) < 2 {
			return fmt.Errorf("provide src dst pairs (run 'aula keys' to list key names)")
		}
		if len(c.Args)%2 != 0 {
			return fmt.Errorf("remap args must be pairs of <src> <dst>")
		}

		for i := 0; i+1 < len(c.Args); i += 2 {
			srcName := strings.ToLower(c.Args[i])
			dstSpec := strings.ToLower(c.Args[i+1])

			srcIdx, ok := aula.KeyNameToIndex[srcName]
			if !ok {
				return fmt.Errorf("unknown source key '%s' (run 'aula keys' to list)", c.Args[i])
			}

			remap, err := parseRemapTarget(srcIdx, dstSpec)
			if err != nil {
				return fmt.Errorf("target '%s': %w", c.Args[i+1], err)
			}

			remaps = append(remaps, remap)
			fmt.Printf("  %s -> %s\n", srcName, dstSpec)
		}
	}

	// --fn flag overrides the YAML layer setting.
	if c.FN {
		fnLayer = true
	}

	dev, err := aula.Open()
	if err != nil {
		return err
	}
	defer dev.Close()

	layer := "normal"
	if fnLayer {
		layer = "FN"
	}
	fmt.Printf("Sending %d remap(s) to %s layer\n", len(remaps), layer)

	if fnLayer {
		err = dev.SetFnKeyRemap(remaps)
	} else {
		err = dev.SetKeyRemap(remaps)
	}
	if err != nil {
		return err
	}
	fmt.Println("OK")
	return nil
}

// parseRemapTarget parses a destination specifier into a KeyRemap.
// Formats:
//   - Key name: "a", "esc", "lctrl" (key-to-key swap).
//   - Media: "media:play", "media:volup" (consumer control).
//   - Mouse: "mouse:lclick", "mouse:scrollup" (mouse action).
//   - Combo: "combo:ctrl+c", "combo:win+d" (key combination with modifiers).
//
// parseRemapTarget(55, "a")            // Key swap.
// parseRemapTarget(55, "media:play")   // Consumer control.
// parseRemapTarget(55, "mouse:lclick") // Mouse action.
// parseRemapTarget(55, "combo:ctrl+c") // Ctrl+C combo.
func parseRemapTarget(srcIdx uint8, spec string) (aula.KeyRemap, error) {
	if strings.HasPrefix(spec, "media:") {
		name := strings.TrimPrefix(spec, "media:")
		code, ok := aula.ConsumerNameToCode[name]
		if !ok {
			names := mapKeys(aula.ConsumerNameToCode)
			return aula.KeyRemap{}, fmt.Errorf("unknown media action '%s' (options: %s)", name, strings.Join(names, ", "))
		}
		return aula.NewConsumerRemap(srcIdx, code), nil
	}

	if strings.HasPrefix(spec, "mouse:") {
		name := strings.TrimPrefix(spec, "mouse:")
		params, ok := aula.MouseNameToParams[name]
		if !ok {
			names := mapKeys(aula.MouseNameToParams)
			return aula.KeyRemap{}, fmt.Errorf("unknown mouse action '%s' (options: %s)", name, strings.Join(names, ", "))
		}
		return aula.NewMouseRemap(srcIdx, params[0], params[1], params[2]), nil
	}

	if strings.HasPrefix(spec, "combo:") {
		expr := strings.TrimPrefix(spec, "combo:")
		return parseComboExpr(srcIdx, expr)
	}

	// Default: key-to-key swap.
	dstHID, ok := aula.KeyNameToHID[spec]
	if !ok {
		return aula.KeyRemap{}, fmt.Errorf("unknown key '%s' (run 'aula keys' to list)", spec)
	}
	return aula.NewKeySwap(srcIdx, dstHID), nil
}

// parseComboExpr parses a modifier+key expression like "ctrl+c" or "win+shift+d".
// Modifier names: ctrl, shift, alt, win (left variants), rctrl, rshift, ralt, rwin.
//
// parseComboExpr(55, "ctrl+c")      // Ctrl+C.
// parseComboExpr(55, "win+d")       // Win+D.
// parseComboExpr(55, "ctrl+shift+a") // Ctrl+Shift+A.
func parseComboExpr(srcIdx uint8, expr string) (aula.KeyRemap, error) {
	modMap := map[string]byte{
		"ctrl": 0x01, "shift": 0x02, "alt": 0x04, "win": 0x08,
		"rctrl": 0x10, "rshift": 0x20, "ralt": 0x40, "rwin": 0x80,
	}

	parts := strings.Split(expr, "+")
	if len(parts) < 2 {
		return aula.KeyRemap{}, fmt.Errorf("combo must be modifier+key (e.g., ctrl+c)")
	}

	var modBits byte
	for _, part := range parts[:len(parts)-1] {
		bit, ok := modMap[part]
		if !ok {
			return aula.KeyRemap{}, fmt.Errorf("unknown modifier '%s' (options: ctrl, shift, alt, win, rctrl, rshift, ralt, rwin)", part)
		}
		modBits |= bit
	}

	keyName := parts[len(parts)-1]
	keyHID, ok := aula.KeyNameToHID[keyName]
	if !ok {
		return aula.KeyRemap{}, fmt.Errorf("unknown key '%s' in combo (run 'aula keys' to list)", keyName)
	}

	return aula.NewKeyCombo(srcIdx, modBits, keyHID), nil
}

// mapKeys returns a sorted list of keys from a map with string keys.
//
// mapKeys(map[string]byte{"a": 1, "b": 2}) // ["a", "b"].
func mapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// remapLayout is the YAML config format for key remap layouts.
type remapLayout struct {
	Layer string            `yaml:"layer"` // "normal" (default) or "fn".
	Keys  map[string]string `yaml:"keys"`  // source key -> target spec.
}

// loadRemapYAML reads a YAML remap file and returns remaps and the layer.
//
// loadRemapYAML("remap.yaml")
func loadRemapYAML(path string) ([]aula.KeyRemap, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("reading %s: %w", path, err)
	}

	var layout remapLayout
	if err := yaml.Unmarshal(data, &layout); err != nil {
		return nil, false, fmt.Errorf("parsing %s: %w", path, err)
	}

	if len(layout.Keys) == 0 {
		return nil, false, fmt.Errorf("no keys defined in %s", path)
	}

	fnLayer := strings.EqualFold(layout.Layer, "fn")

	var remaps []aula.KeyRemap
	for src, dst := range layout.Keys {
		srcName := strings.ToLower(src)
		dstSpec := strings.ToLower(dst)

		srcIdx, ok := aula.KeyNameToIndex[srcName]
		if !ok {
			return nil, false, fmt.Errorf("unknown source key '%s' in %s", src, path)
		}

		remap, err := parseRemapTarget(srcIdx, dstSpec)
		if err != nil {
			return nil, false, fmt.Errorf("key '%s' target '%s': %w", src, dst, err)
		}

		remaps = append(remaps, remap)
		fmt.Printf("  %s -> %s\n", srcName, dstSpec)
	}

	return remaps, fnLayer, nil
}

// perKeyLayout is the YAML config format for per-key RGB layouts.
type perKeyLayout struct {
	All        [3]uint8            `yaml:"all"`
	Keys       map[string][3]uint8 `yaml:"keys"`
	Brightness *uint8              `yaml:"brightness"`
}

// loadPerKeyYAML reads a YAML layout file and returns the key colors and
// optional brightness.
//
// loadPerKeyYAML("layout.yaml")
func loadPerKeyYAML(path string) ([]aula.KeyColor, *uint8) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", path, err)
		os.Exit(1)
	}

	var layout perKeyLayout
	if err := yaml.Unmarshal(data, &layout); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", path, err)
		os.Exit(1)
	}

	keys := buildKeysFromLayout(layout.All, layout.Keys)
	if len(keys) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no keys defined in %s\n", path)
		os.Exit(1)
	}
	return keys, layout.Brightness
}

// parsePerKeyArgs parses CLI args into key colors.
// The all parameter is the --all flag value (3 RGB bytes), nil if not set.
//
// parsePerKeyArgs([]uint8{0, 0, 50}, []string{"esc", "255", "0", "0"})
func parsePerKeyArgs(all []uint8, args []string) []aula.KeyColor {
	var baseColor [3]uint8
	hasBase := false

	if len(all) > 0 {
		if len(all) != 3 {
			fmt.Fprintln(os.Stderr, "Error: --all requires exactly 3 values (r g b)")
			os.Exit(1)
		}
		baseColor[0] = all[0]
		baseColor[1] = all[1]
		baseColor[2] = all[2]
		hasBase = true
	}

	remaining := args
	overrides := make(map[string][3]uint8)
	if len(remaining)%4 != 0 {
		fmt.Fprintln(os.Stderr, "Error: key args must be groups of <key> <r> <g> <b>")
		os.Exit(1)
	}
	for i := 0; i+3 < len(remaining); i += 4 {
		name := strings.ToLower(remaining[i])
		if _, ok := aula.KeyNameToIndex[name]; !ok {
			fmt.Fprintf(os.Stderr, "Error: unknown key '%s'. Run 'aula keys' for options.\n", remaining[i])
			os.Exit(1)
		}
		overrides[name] = [3]uint8{
			parseByte(remaining[i+1], "red"),
			parseByte(remaining[i+2], "green"),
			parseByte(remaining[i+3], "blue"),
		}
	}

	if !hasBase && len(overrides) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no keys specified")
		os.Exit(1)
	}

	return buildKeysFromLayout(baseColor, overrides)
}

// buildKeysFromLayout builds key color list from a base color and per-key overrides.
//
// buildKeysFromLayout([3]uint8{0, 0, 50}, map[string][3]uint8{"esc": {255, 0, 0}})
func buildKeysFromLayout(base [3]uint8, overrides map[string][3]uint8) []aula.KeyColor {
	hasBase := base[0] != 0 || base[1] != 0 || base[2] != 0
	var keys []aula.KeyColor

	if hasBase {
		for name, idx := range aula.KeyNameToIndex {
			color := base
			if ov, ok := overrides[name]; ok {
				color = ov
			}
			keys = append(keys, aula.KeyColor{LightIndex: idx, R: color[0], G: color[1], B: color[2]})
		}
	} else {
		for name, color := range overrides {
			idx := aula.KeyNameToIndex[name]
			keys = append(keys, aula.KeyColor{LightIndex: idx, R: color[0], G: color[1], B: color[2]})
		}
	}

	return keys
}

// parseMode converts a mode string (name or number) to a LightingMode.
//
// parseMode("static") // aula.ModeStatic.
// parseMode("7") // aula.ModeBreath.
func parseMode(s string) aula.LightingMode {
	if v, err := strconv.Atoi(s); err == nil {
		if v < 0 || v > 19 {
			fmt.Fprintf(os.Stderr, "Error: mode %d out of range (0-19)\n", v)
			os.Exit(1)
		}
		return aula.LightingMode(v)
	}

	for m, name := range aula.ModeNames {
		if strings.EqualFold(name, s) {
			return m
		}
	}

	fmt.Fprintf(os.Stderr, "Error: unknown mode '%s'. Run 'aula modes' to see options.\n", s)
	os.Exit(1)
	return 0
}

// parseByte converts a string to uint8.
//
// parseByte("255", "red") // 255.
func parseByte(s, name string) uint8 {
	v, err := strconv.Atoi(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid %s value '%s'\n", name, s)
		os.Exit(1)
	}
	if v < 0 || v > 255 {
		fmt.Fprintf(os.Stderr, "Error: %s %d out of range (0-255)\n", name, v)
		os.Exit(1)
	}
	return uint8(v)
}
