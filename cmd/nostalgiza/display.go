package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/richardwooding/nostalgiza/internal/emulator"
	"github.com/richardwooding/nostalgiza/internal/ppu"
)

// DMG palette colors (classic Game Boy green tones).
var dmgPalette = [4]color.RGBA{
	{0xE0, 0xF8, 0xD0, 0xFF}, // White (lightest)
	{0x88, 0xC0, 0x70, 0xFF}, // Light gray
	{0x34, 0x68, 0x56, 0xFF}, // Dark gray
	{0x08, 0x18, 0x20, 0xFF}, // Black (darkest)
}

// Display implements the Ebiten game interface for the Game Boy emulator.
type Display struct {
	emulator *emulator.Emulator
	screen   *ebiten.Image
}

// NewDisplay creates a new display for the emulator.
func NewDisplay(emu *emulator.Emulator) *Display {
	return &Display{
		emulator: emu,
		screen:   ebiten.NewImage(ppu.ScreenWidth, ppu.ScreenHeight),
	}
}

// Update updates the game logic (runs one frame worth of cycles).
// This is called 60 times per second by Ebiten.
func (d *Display) Update() error {
	// Game Boy runs at ~59.73 Hz, which is close to 60 Hz
	// One frame = 70,224 cycles
	d.emulator.RunCycles(ppu.DotsPerFrame)

	return nil
}

// Draw draws the game screen.
// This is called after Update.
func (d *Display) Draw(screen *ebiten.Image) {
	// Get framebuffer from PPU
	framebuffer := d.emulator.PPU.GetFramebuffer()

	// Convert framebuffer to RGBA image
	for y := 0; y < ppu.ScreenHeight; y++ {
		for x := 0; x < ppu.ScreenWidth; x++ {
			// Get color index (0-3)
			colorIndex := framebuffer[y*ppu.ScreenWidth+x]

			// Map to DMG palette
			c := dmgPalette[colorIndex&0x03]

			// Set pixel
			d.screen.Set(x, y, c)
		}
	}

	// Draw the screen to the window
	screen.DrawImage(d.screen, nil)
}

// Layout returns the game screen size.
func (d *Display) Layout(_, _ int) (int, int) {
	return ppu.ScreenWidth, ppu.ScreenHeight
}
