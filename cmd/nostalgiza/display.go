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
	emulator    *emulator.Emulator
	screen      *ebiten.Image
	pixels      []byte // Pre-allocated pixel buffer to avoid GC pressure
	audioPlayer *AudioPlayer
}

// NewDisplay creates a new display for the emulator.
func NewDisplay(emu *emulator.Emulator) *Display {
	// Create audio player
	audioPlayer, err := NewAudioPlayer(emu.APU)
	if err != nil {
		// Audio is optional - continue without it if initialization fails
		audioPlayer = nil
	} else {
		// Start audio playback
		audioPlayer.Start()
	}

	return &Display{
		emulator:    emu,
		screen:      ebiten.NewImage(ppu.ScreenWidth, ppu.ScreenHeight),
		pixels:      make([]byte, ppu.ScreenWidth*ppu.ScreenHeight*4), // RGBA format
		audioPlayer: audioPlayer,
	}
}

// Update updates the game logic (runs one frame worth of cycles).
// This is called 60 times per second by Ebiten.
func (d *Display) Update() error {
	// Handle keyboard input
	d.handleInput()

	// Game Boy runs at ~59.73 Hz, which is close to 60 Hz
	// One frame = 70,224 cycles
	d.emulator.RunCycles(ppu.DotsPerFrame)

	// Update audio player with new samples
	if d.audioPlayer != nil {
		d.audioPlayer.Update()
	}

	return nil
}

// handleInput processes keyboard input and updates joypad state.
func (d *Display) handleInput() {
	// Map keyboard keys to Game Boy buttons
	keyMap := map[ebiten.Key]string{
		ebiten.KeyArrowUp:    "Up",
		ebiten.KeyArrowDown:  "Down",
		ebiten.KeyArrowLeft:  "Left",
		ebiten.KeyArrowRight: "Right",
		ebiten.KeyZ:          "A",
		ebiten.KeyX:          "B",
		ebiten.KeyEnter:      "Start",
		ebiten.KeyShift:      "Select",
	}

	// Check each key and update joypad state
	for key, button := range keyMap {
		if ebiten.IsKeyPressed(key) {
			d.emulator.Joypad.PressButton(button)
		} else {
			d.emulator.Joypad.ReleaseButton(button)
		}
	}
}

// Draw draws the game screen.
// This is called after Update.
func (d *Display) Draw(screen *ebiten.Image) {
	// Get framebuffer from PPU
	framebuffer := d.emulator.PPU.GetFramebuffer()

	// Convert framebuffer to RGBA image using bulk pixel update
	// This is much faster than individual Set() calls per pixel
	// Reuse pre-allocated pixel buffer to avoid GC pressure

	for i, colorIndex := range framebuffer {
		// Map to DMG palette
		c := dmgPalette[colorIndex&0x03]

		// Write RGBA values
		offset := i * 4
		d.pixels[offset] = c.R
		d.pixels[offset+1] = c.G
		d.pixels[offset+2] = c.B
		d.pixels[offset+3] = c.A
	}

	// Write all pixels at once (much faster than 23,040 individual Set() calls)
	d.screen.WritePixels(d.pixels)

	// Draw the screen to the window
	screen.DrawImage(d.screen, nil)
}

// Layout returns the game screen size.
func (d *Display) Layout(_, _ int) (int, int) {
	return ppu.ScreenWidth, ppu.ScreenHeight
}
