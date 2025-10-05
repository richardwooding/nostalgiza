package ppu

import (
	"testing"
)

// stepMany steps the PPU by the given number of cycles (handles any value).
func stepMany(p *PPU, cycles int) {
	for cycles > 0 {
		step := 200
		if cycles < 200 {
			step = cycles
		}
		p.Step(uint8(step)) //nolint:gosec // Test helper, values are controlled
		cycles -= step
	}
}

// TestPPUInitialization tests PPU creation and initial state.
func TestPPUInitialization(t *testing.T) {
	ppu := New(nil)

	if ppu == nil {
		t.Fatal("New() returned nil")
	}

	// Check initial register values
	if ppu.lcdc != 0x91 {
		t.Errorf("LCDC initial value = 0x%02X, want 0x91", ppu.lcdc)
	}

	if ppu.stat != 0x00 {
		t.Errorf("STAT initial value = 0x%02X, want 0x00", ppu.stat)
	}

	if ppu.bgp != 0xFC {
		t.Errorf("BGP initial value = 0x%02X, want 0xFC", ppu.bgp)
	}

	if ppu.obp0 != 0xFF {
		t.Errorf("OBP0 initial value = 0x%02X, want 0xFF", ppu.obp0)
	}

	if ppu.obp1 != 0xFF {
		t.Errorf("OBP1 initial value = 0x%02X, want 0xFF", ppu.obp1)
	}

	// Check initial state
	if ppu.mode != ModeOAMScan {
		t.Errorf("Initial mode = %d, want %d (OAM Scan)", ppu.mode, ModeOAMScan)
	}

	if ppu.ly != 0 {
		t.Errorf("Initial LY = %d, want 0", ppu.ly)
	}

	if ppu.dots != 0 {
		t.Errorf("Initial dots = %d, want 0", ppu.dots)
	}
}

// TestPPUModeTransitions tests PPU mode state machine.
func TestPPUModeTransitions(t *testing.T) {
	ppu := New(nil)

	// Start in OAM Scan (mode 2)
	if ppu.mode != ModeOAMScan {
		t.Fatalf("Expected initial mode OAM Scan, got %d", ppu.mode)
	}

	// Step through OAM Scan -> Drawing
	ppu.Step(DotsOAMScan)
	if ppu.mode != ModeDrawing {
		t.Errorf("After %d dots, mode = %d, want %d (Drawing)", DotsOAMScan, ppu.mode, ModeDrawing)
	}

	// Step through Drawing -> H-Blank
	ppu.Step(DotsDrawing)
	if ppu.mode != ModeHBlank {
		t.Errorf("After drawing, mode = %d, want %d (H-Blank)", ppu.mode, ModeHBlank)
	}

	// Step through H-Blank -> OAM Scan (next scanline)
	stepMany(ppu, DotsHBlank)
	if ppu.mode != ModeOAMScan {
		t.Errorf("After H-Blank, mode = %d, want %d (OAM Scan)", ppu.mode, ModeOAMScan)
	}

	if ppu.ly != 1 {
		t.Errorf("After first scanline, LY = %d, want 1", ppu.ly)
	}
}

// TestPPUVBlank tests V-Blank transition and interrupt.
func TestPPUVBlank(t *testing.T) {
	interruptTriggered := false
	interruptType := uint8(0xFF)

	ppu := New(func(interrupt uint8) {
		interruptTriggered = true
		interruptType = interrupt
	})

	// Advance through 144 scanlines to reach V-Blank
	for i := 0; i < ScanlinesVisible; i++ {
		stepMany(ppu, DotsPerScanline)
	}

	// Should now be in V-Blank
	if ppu.mode != ModeVBlank {
		t.Errorf("After %d scanlines, mode = %d, want %d (V-Blank)", ScanlinesVisible, ppu.mode, ModeVBlank)
	}

	if ppu.ly != ScanlinesVisible {
		t.Errorf("At V-Blank start, LY = %d, want %d", ppu.ly, ScanlinesVisible)
	}

	// V-Blank interrupt should have been triggered
	if !interruptTriggered {
		t.Error("V-Blank interrupt was not triggered")
	}

	if interruptType != InterruptVBlank {
		t.Errorf("Interrupt type = %d, want %d (V-Blank)", interruptType, InterruptVBlank)
	}
}

// TestPPUFrameTiming tests complete frame timing.
func TestPPUFrameTiming(t *testing.T) {
	ppu := New(nil)

	// Run through a complete frame (154 scanlines)
	// After 154 scanlines we should be back at LY=0
	for i := 0; i < ScanlinesTotal; i++ {
		stepMany(ppu, DotsPerScanline)
	}

	// Should be back at scanline 0, mode OAM Scan
	if ppu.ly != 0 {
		t.Errorf("After one frame, LY = %d, want 0", ppu.ly)
	}

	if ppu.mode != ModeOAMScan {
		t.Errorf("After one frame, mode = %d, want %d (OAM Scan)", ppu.mode, ModeOAMScan)
	}
}

// TestPPURegisterReadWrite tests PPU register access.
func TestPPURegisterReadWrite(t *testing.T) {
	ppu := New(nil)

	tests := []struct {
		addr  uint16
		value uint8
		name  string
	}{
		{0xFF40, 0x80, "LCDC"},
		{0xFF42, 0x12, "SCY"},
		{0xFF43, 0x34, "SCX"},
		{0xFF45, 0x90, "LYC"},
		{0xFF47, 0xE4, "BGP"},
		{0xFF48, 0xD2, "OBP0"},
		{0xFF49, 0xA0, "OBP1"},
		{0xFF4A, 0x50, "WY"},
		{0xFF4B, 0x07, "WX"},
	}

	for _, tt := range tests {
		ppu.WriteRegister(tt.addr, tt.value)
		got := ppu.ReadRegister(tt.addr)

		// Special case for STAT (only bits 6-3 are writable)
		if tt.addr == 0xFF41 {
			want := tt.value & 0x78
			if got&0x78 != want {
				t.Errorf("Register %s (0x%04X) writable bits = 0x%02X, want 0x%02X", tt.name, tt.addr, got&0x78, want)
			}
		} else if tt.addr != 0xFF44 { // LY is read-only
			if got != tt.value {
				t.Errorf("Register %s (0x%04X) = 0x%02X, want 0x%02X", tt.name, tt.addr, got, tt.value)
			}
		}
	}
}

// TestPPULYReadOnly tests that LY register is read-only (writes reset to 0).
func TestPPULYReadOnly(t *testing.T) {
	ppu := New(nil)

	// Advance to scanline 10
	for i := 0; i < 10; i++ {
		stepMany(ppu, DotsPerScanline)
	}

	if ppu.ly != 10 {
		t.Fatalf("Setup failed: LY = %d, want 10", ppu.ly)
	}

	// Writing to LY should reset it to 0
	ppu.WriteRegister(0xFF44, 0xFF)

	if ppu.ly != 0 {
		t.Errorf("After write to LY, LY = %d, want 0", ppu.ly)
	}
}

// TestPPUVRAMAccess tests VRAM read/write.
func TestPPUVRAMAccess(t *testing.T) {
	ppu := New(nil)

	// VRAM should be accessible in mode 0 (H-Blank)
	ppu.mode = ModeHBlank
	ppu.WriteVRAM(0x0000, 0x42)
	if got := ppu.ReadVRAM(0x0000); got != 0x42 {
		t.Errorf("VRAM[0x0000] in H-Blank = 0x%02X, want 0x42", got)
	}

	// VRAM should be inaccessible in mode 3 (Drawing)
	ppu.mode = ModeDrawing
	ppu.WriteVRAM(0x0000, 0xFF)
	if got := ppu.ReadVRAM(0x0000); got != 0xFF {
		t.Errorf("VRAM read in Drawing mode = 0x%02X, want 0xFF (blocked)", got)
	}

	// Value should not have changed
	ppu.mode = ModeHBlank
	if got := ppu.ReadVRAM(0x0000); got != 0x42 {
		t.Errorf("VRAM[0x0000] after blocked write = 0x%02X, want 0x42", got)
	}
}

// TestPPUOAMAccess tests OAM read/write.
func TestPPUOAMAccess(t *testing.T) {
	ppu := New(nil)

	// OAM should be accessible in mode 0 (H-Blank)
	ppu.mode = ModeHBlank
	ppu.WriteOAM(0x00, 0x12)
	if got := ppu.ReadOAM(0x00); got != 0x12 {
		t.Errorf("OAM[0x00] in H-Blank = 0x%02X, want 0x12", got)
	}

	// OAM should be inaccessible in mode 2 (OAM Scan)
	ppu.mode = ModeOAMScan
	ppu.WriteOAM(0x00, 0xFF)
	if got := ppu.ReadOAM(0x00); got != 0xFF {
		t.Errorf("OAM read in OAM Scan mode = 0x%02X, want 0xFF (blocked)", got)
	}

	// OAM should be inaccessible in mode 3 (Drawing)
	ppu.mode = ModeDrawing
	ppu.WriteOAM(0x00, 0xFF)
	if got := ppu.ReadOAM(0x00); got != 0xFF {
		t.Errorf("OAM read in Drawing mode = 0x%02X, want 0xFF (blocked)", got)
	}

	// Value should not have changed
	ppu.mode = ModeHBlank
	if got := ppu.ReadOAM(0x00); got != 0x12 {
		t.Errorf("OAM[0x00] after blocked writes = 0x%02X, want 0x12", got)
	}
}

// TestPPULYCFlag tests LYC=LY flag and interrupt.
func TestPPULYCFlag(t *testing.T) {
	interruptCount := 0

	ppu := New(func(interrupt uint8) {
		if interrupt == InterruptSTAT {
			interruptCount++
		}
	})

	// Enable LYC interrupt
	ppu.stat |= STATLYCInterrupt

	// Set LYC to 5
	ppu.WriteRegister(0xFF45, 5)

	// LYC flag should not be set yet
	if ppu.stat&STATLYCFlag != 0 {
		t.Error("LYC flag set before LY=LYC")
	}

	// Advance to scanline 5
	for i := 0; i < 5; i++ {
		stepMany(ppu, DotsPerScanline)
	}

	// LYC flag should now be set
	if ppu.stat&STATLYCFlag == 0 {
		t.Error("LYC flag not set when LY=LYC")
	}

	// Interrupt should have been triggered
	if interruptCount == 0 {
		t.Error("LYC interrupt not triggered when LY=LYC")
	}

	// Advance past scanline 5
	stepMany(ppu, DotsPerScanline)

	// LYC flag should be cleared
	if ppu.stat&STATLYCFlag != 0 {
		t.Error("LYC flag still set after LY!=LYC")
	}
}

// TestPPUReset tests PPU reset functionality.
func TestPPUReset(t *testing.T) {
	ppu := New(nil)
	// Set to H-Blank mode so VRAM/OAM are accessible
	ppu.SetModeForTesting(ModeHBlank)

	// Modify some state
	ppu.WriteVRAM(0x0000, 0x42)
	ppu.WriteOAM(0x00, 0x12)
	ppu.WriteRegister(0xFF42, 0x50)   // SCY
	stepMany(ppu, DotsPerScanline*10) // Advance 10 scanlines

	// Reset
	ppu.Reset()

	// Set to H-Blank mode again so we can read VRAM/OAM
	ppu.SetModeForTesting(ModeHBlank)

	// Check VRAM cleared
	if got := ppu.ReadVRAM(0x0000); got != 0x00 {
		t.Errorf("After reset, VRAM[0x0000] = 0x%02X, want 0x00", got)
	}

	// Check OAM cleared
	if got := ppu.ReadOAM(0x00); got != 0x00 {
		t.Errorf("After reset, OAM[0x00] = 0x%02X, want 0x00", got)
	}

	// Check registers reset
	if ppu.scy != 0 {
		t.Errorf("After reset, SCY = 0x%02X, want 0x00", ppu.scy)
	}

	// Check state reset
	if ppu.ly != 0 {
		t.Errorf("After reset, LY = %d, want 0", ppu.ly)
	}

	// Mode will be the one we set (H-Blank) for testing, not OAM Scan
	// since we manually set it to access VRAM/OAM
	if ppu.mode != ModeHBlank {
		t.Errorf("After reset then SetMode, mode = %d, want %d (H-Blank)", ppu.mode, ModeHBlank)
	}

	if ppu.dots != 0 {
		t.Errorf("After reset, dots = %d, want 0", ppu.dots)
	}
}

// TestGetTilePixel tests tile pixel decoding.
func TestGetTilePixel(t *testing.T) {
	ppu := New(nil)

	// Set up a simple 8x8 tile pattern (checkerboard)
	// Tile at address 0x0000 in VRAM
	// Each row is 2 bytes: byte1 (low bit) and byte2 (high bit)
	// Pattern: alternating pixels (color 0 and color 3)
	ppu.vram[0x0000] = 0xAA // 10101010
	ppu.vram[0x0001] = 0xAA // 10101010 -> pixels: 3,0,3,0,3,0,3,0

	ppu.vram[0x0002] = 0x55 // 01010101
	ppu.vram[0x0003] = 0x55 // 01010101 -> pixels: 0,3,0,3,0,3,0,3

	// Test first row (y=0)
	tests := []struct {
		x    uint16
		want uint8
	}{
		{0, 3}, {1, 0}, {2, 3}, {3, 0}, {4, 3}, {5, 0}, {6, 3}, {7, 0},
	}

	for _, tt := range tests {
		got := ppu.getTilePixel(0, tt.x, 0)
		if got != tt.want {
			t.Errorf("getTilePixel(0, %d, 0) = %d, want %d", tt.x, got, tt.want)
		}
	}

	// Test second row (y=1)
	tests2 := []struct {
		x    uint16
		want uint8
	}{
		{0, 0}, {1, 3}, {2, 0}, {3, 3}, {4, 0}, {5, 3}, {6, 0}, {7, 3},
	}

	for _, tt := range tests2 {
		got := ppu.getTilePixel(0, tt.x, 1)
		if got != tt.want {
			t.Errorf("getTilePixel(0, %d, 1) = %d, want %d", tt.x, got, tt.want)
		}
	}
}

// TestApplyPalette tests palette application.
func TestApplyPalette(t *testing.T) {
	ppu := New(nil)

	// Set up a test palette: 0xE4 = 11 10 01 00
	// Color 0 -> 00 (0)
	// Color 1 -> 01 (1)
	// Color 2 -> 10 (2)
	// Color 3 -> 11 (3)
	palette := uint8(0xE4)

	tests := []struct {
		colorIndex uint8
		want       uint8
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 3},
	}

	for _, tt := range tests {
		got := ppu.applyPalette(tt.colorIndex, palette)
		if got != tt.want {
			t.Errorf("applyPalette(%d, 0xE4) = %d, want %d", tt.colorIndex, got, tt.want)
		}
	}

	// Test different palette: 0x1B = 00 01 10 11
	palette = 0x1B
	tests2 := []struct {
		colorIndex uint8
		want       uint8
	}{
		{0, 3},
		{1, 2},
		{2, 1},
		{3, 0},
	}

	for _, tt := range tests2 {
		got := ppu.applyPalette(tt.colorIndex, palette)
		if got != tt.want {
			t.Errorf("applyPalette(%d, 0x1B) = %d, want %d", tt.colorIndex, got, tt.want)
		}
	}
}

// TestGetFramebuffer tests framebuffer access.
func TestGetFramebuffer(t *testing.T) {
	ppu := New(nil)

	fb := ppu.GetFramebuffer()

	if fb == nil {
		t.Fatal("GetFramebuffer() returned nil")
	}

	if len(fb) != ScreenWidth*ScreenHeight {
		t.Errorf("Framebuffer size = %d, want %d", len(fb), ScreenWidth*ScreenHeight)
	}

	// Framebuffer should be zeroed initially
	for i, pixel := range fb {
		if pixel != 0 {
			t.Errorf("Framebuffer[%d] = %d, want 0", i, pixel)
			break
		}
	}
}
