// mkimage generates a raw LCD image buffer file for the Aula F108 Pro.
//
// The output file matches the format the keyboard expects:
//   - 256-byte header (byte[0]=frame count, byte[1..N]=per-frame delay, rest 0xFF)
//   - RGB565 pixel data (240x135x2 = 64800 bytes per frame), little-endian
//   - Padded to a multiple of 4096 bytes (page size)
//
// Usage:
//
//	mkimage -o output.bin                      # solid cyan (default)
//	mkimage -o output.bin -r 255 -g 0 -b 0    # solid red
//	mkimage -o output.bin -hex FF00FF          # solid magenta (hex RGB)
//	mkimage -o output.bin -gif input.gif       # convert GIF file
package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"os"
)

const (
	screenWidth     = 240
	screenHeight    = 135
	bytesPerPixel   = 2
	frameSize       = screenWidth * screenHeight * bytesPerPixel // 64800
	gifHeaderLength = 256
	pageSize        = 4096
)

func rgb888ToRgb565(r, g, b uint8) uint16 {
	return (uint16(r>>3) << 11) | (uint16(g>>2) << 5) | uint16(b>>3)
}

func main() {
	outFile := flag.String("o", "lcd-image.bin", "output file path")
	rVal := flag.Int("r", 0, "red component (0-255)")
	gVal := flag.Int("g", 255, "green component (0-255)")
	bVal := flag.Int("b", 255, "blue component (0-255)")
	hexColor := flag.String("hex", "", "hex RGB color (e.g. FF00FF), overrides -r -g -b")
	gifFile := flag.String("gif", "", "GIF file to convert (overrides solid color)")
	flag.Parse()

	if *gifFile != "" {
		convertGIF(*gifFile, *outFile)
		return
	}

	r, g, b := uint8(*rVal), uint8(*gVal), uint8(*bVal)
	if *hexColor != "" {
		raw, err := hex.DecodeString(*hexColor)
		if err != nil || len(raw) != 3 {
			fmt.Fprintf(os.Stderr, "Invalid hex color %q (expected 6 hex digits like FF00FF)\n", *hexColor)
			os.Exit(1)
		}
		r, g, b = raw[0], raw[1], raw[2]
	}

	solidColor(r, g, b, *outFile)
}

func solidColor(r, g, b uint8, outFile string) {
	pixel := rgb888ToRgb565(r, g, b)
	fmt.Printf("Color: R=%d G=%d B=%d -> RGB565=0x%04X\n", r, g, b, pixel)

	// Calculate sizes.
	totalSize := gifHeaderLength + frameSize
	pageCount := (totalSize + pageSize - 1) / pageSize
	paddedSize := pageCount * pageSize
	fmt.Printf("Header: %d bytes\n", gifHeaderLength)
	fmt.Printf("Pixels: %d bytes (%dx%d, %d bytes/pixel)\n",
		frameSize, screenWidth, screenHeight, bytesPerPixel)
	fmt.Printf("Total: %d bytes, padded: %d bytes (%d pages of %d)\n",
		totalSize, paddedSize, pageCount, pageSize)

	// Build buffer filled with 0xFF.
	buf := make([]byte, paddedSize)
	for i := range buf {
		buf[i] = 0xFF
	}

	// Header: byte[0] = frame count, byte[1] = delay for frame 0.
	buf[0] = 1 // 1 frame.
	buf[1] = 1 // Delay value (frame_duration_centiseconds / 2, min 1).

	// Fill pixel data at offset 256.
	var pixelBytes [2]byte
	binary.LittleEndian.PutUint16(pixelBytes[:], pixel)
	for i := 0; i < screenWidth*screenHeight; i++ {
		offset := gifHeaderLength + i*2
		buf[offset] = pixelBytes[0]
		buf[offset+1] = pixelBytes[1]
	}

	writeAndVerify(buf, outFile)
}

func convertGIF(gifPath, outFile string) {
	f, err := os.Open(gifPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot open GIF: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	g, err := gif.DecodeAll(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot decode GIF: %v\n", err)
		os.Exit(1)
	}

	frameCount := len(g.Image)
	if frameCount > 255 {
		fmt.Fprintf(os.Stderr, "GIF has %d frames, max is 255. Truncating.\n", frameCount)
		frameCount = 255
	}

	fmt.Printf("GIF: %s\n", gifPath)
	fmt.Printf("  Canvas: %dx%d\n", g.Config.Width, g.Config.Height)
	fmt.Printf("  Frames: %d\n", frameCount)

	if g.Config.Width != screenWidth || g.Config.Height != screenHeight {
		fmt.Fprintf(os.Stderr, "Warning: GIF is %dx%d, keyboard expects %dx%d. Image will be cropped/padded.\n",
			g.Config.Width, g.Config.Height, screenWidth, screenHeight)
	}

	// Calculate sizes.
	totalSize := gifHeaderLength + frameSize*frameCount
	pageCount := (totalSize + pageSize - 1) / pageSize
	paddedSize := pageCount * pageSize
	fmt.Printf("  Raw size: %d bytes, padded: %d bytes (%d pages)\n",
		totalSize, paddedSize, pageCount)

	// Build buffer filled with 0xFF.
	buf := make([]byte, paddedSize)
	for i := range buf {
		buf[i] = 0xFF
	}

	// Header: byte[0] = frame count, byte[1..N] = per-frame delays.
	buf[0] = byte(frameCount)
	for i := 0; i < frameCount; i++ {
		delay := g.Delay[i] // in 100ths of a second.
		// Keyboard stores delay as frame_duration_centiseconds / 2, min 1.
		d := delay / 2
		if d < 1 {
			d = 1
		}
		if d > 255 {
			d = 255
		}
		buf[1+i] = byte(d)
	}

	// Composite frames. GIF frames can be partial (dispose method),
	// so we need to render each frame onto a canvas.
	canvas := image.NewRGBA(image.Rect(0, 0, g.Config.Width, g.Config.Height))

	for i := 0; i < frameCount; i++ {
		frame := g.Image[i]
		bounds := frame.Bounds()

		// Draw this frame onto the canvas.
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				c := frame.At(x, y)
				// Skip transparent pixels (keep canvas as-is).
				_, _, _, a := c.RGBA()
				if a == 0 {
					continue
				}
				canvas.Set(x, y, c)
			}
		}

		// Convert canvas to RGB565 for this frame.
		offset := gifHeaderLength + i*frameSize
		for y := 0; y < screenHeight; y++ {
			for x := 0; x < screenWidth; x++ {
				var r8, g8, b8 uint8
				if x < g.Config.Width && y < g.Config.Height {
					c := canvas.At(x, y)
					rr, gg, bb, _ := c.RGBA()
					r8 = uint8(rr >> 8)
					g8 = uint8(gg >> 8)
					b8 = uint8(bb >> 8)
				}
				px := rgb888ToRgb565(r8, g8, b8)
				binary.LittleEndian.PutUint16(buf[offset:], px)
				offset += 2
			}
		}

		// Handle disposal method for next frame.
		if i < len(g.Disposal) {
			switch g.Disposal[i] {
			case gif.DisposalBackground:
				// Clear the frame area to background.
				for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
					for x := bounds.Min.X; x < bounds.Max.X; x++ {
						canvas.Set(x, y, color.RGBA{0, 0, 0, 255})
					}
				}
			case gif.DisposalPrevious:
				// Should restore to previous — simplified: clear to black.
				for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
					for x := bounds.Min.X; x < bounds.Max.X; x++ {
						canvas.Set(x, y, color.RGBA{0, 0, 0, 255})
					}
				}
			}
			// DisposalNone (0) and default: leave canvas as-is.
		}

		if (i+1)%50 == 0 || i == frameCount-1 {
			fmt.Printf("  Converted frame %d/%d\n", i+1, frameCount)
		}
	}

	writeAndVerify(buf, outFile)
}

func writeAndVerify(buf []byte, outFile string) {
	if err := os.WriteFile(outFile, buf, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Write error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %s (%d bytes, %d pages)\n", outFile, len(buf), len(buf)/pageSize)

	// Print verification info.
	fmt.Println("\nVerification:")
	fmt.Printf("  Header (first 16 bytes): %s\n", hex.EncodeToString(buf[:16]))
	fmt.Printf("  Frame count: %d\n", buf[0])
	fmt.Printf("  First pixel (offset 256): %s\n", hex.EncodeToString(buf[256:258]))
}
