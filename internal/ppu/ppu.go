// Package ppu implements the Game Boy Picture Processing Unit (PPU).
// The PPU handles all graphics rendering including background, window, and sprite layers.
package ppu

const (
	// ScreenWidth is the Game Boy screen width in pixels.
	ScreenWidth = 160
	// ScreenHeight is the Game Boy screen height in pixels.
	ScreenHeight = 144
)

const (
	// ModeHBlank is the PPU mode for H-Blank (end of scanline).
	ModeHBlank = 0
	// ModeVBlank is the PPU mode for V-Blank (vertical blank period).
	ModeVBlank = 1
	// ModeOAMScan is the PPU mode for OAM Scan (searching for sprites).
	ModeOAMScan = 2
	// ModeDrawing is the PPU mode for drawing pixels.
	ModeDrawing = 3
)

const (
	// DotsPerScanline is the total number of dots per scanline.
	DotsPerScanline = 456
	// DotsOAMScan is the duration of Mode 2 (OAM Scan) in dots.
	DotsOAMScan = 80
	// DotsDrawing is the duration of Mode 3 (Drawing) in dots.
	DotsDrawing = 172
	// DotsHBlank is the duration of Mode 0 (H-Blank) in dots.
	DotsHBlank = 204
	// ScanlinesVisible is the number of visible scanlines.
	ScanlinesVisible = 144
	// ScanlinesVBlank is the number of V-Blank scanlines.
	ScanlinesVBlank = 10
	// ScanlinesTotal is the total number of scanlines per frame.
	ScanlinesTotal = 154
	// DotsPerFrame is the total number of dots per frame.
	DotsPerFrame = 70224
)

const (
	// VRAMSize is the size of VRAM in bytes (8KB).
	VRAMSize = 0x2000
	// OAMSize is the size of OAM in bytes (160 bytes).
	OAMSize = 0xA0
)

const (
	// LCDCLCDEnable is the LCDC bit for LCD Display Enable.
	LCDCLCDEnable = 1 << 7
	// LCDCWindowTileMap is the LCDC bit for Window Tile Map select.
	LCDCWindowTileMap = 1 << 6
	// LCDCWindowEnable is the LCDC bit for Window Display Enable.
	LCDCWindowEnable = 1 << 5
	// LCDCBGTileData is the LCDC bit for BG & Window Tile Data select.
	LCDCBGTileData = 1 << 4
	// LCDCBGTileMap is the LCDC bit for BG Tile Map select.
	LCDCBGTileMap = 1 << 3
	// LCDCOBJSize is the LCDC bit for OBJ (sprite) size (0=8x8, 1=8x16).
	LCDCOBJSize = 1 << 2
	// LCDCOBJEnable is the LCDC bit for OBJ (sprite) Display Enable.
	LCDCOBJEnable = 1 << 1
	// LCDCBGWindowEnable is the LCDC bit for BG & Window Display Enable.
	LCDCBGWindowEnable = 1 << 0
)

const (
	// STATLYCInterrupt is the STAT bit for LYC=LY Interrupt.
	STATLYCInterrupt = 1 << 6
	// STATMode2Interrupt is the STAT bit for Mode 2 OAM Interrupt.
	STATMode2Interrupt = 1 << 5
	// STATMode1Interrupt is the STAT bit for Mode 1 V-Blank Interrupt.
	STATMode1Interrupt = 1 << 4
	// STATMode0Interrupt is the STAT bit for Mode 0 H-Blank Interrupt.
	STATMode0Interrupt = 1 << 3
	// STATLYCFlag is the STAT bit for LYC=LY Flag.
	STATLYCFlag = 1 << 2
	// STATModeMask is the mask for STAT mode bits.
	STATModeMask = 0x03
)

const (
	// SpriteAttrPriority is the sprite attribute bit for priority (0=Above BG, 1=Behind BG colors 1-3).
	SpriteAttrPriority = 1 << 7
	// SpriteAttrYFlip is the sprite attribute bit for vertical flip.
	SpriteAttrYFlip = 1 << 6
	// SpriteAttrXFlip is the sprite attribute bit for horizontal flip.
	SpriteAttrXFlip = 1 << 5
	// SpriteAttrPalette is the sprite attribute bit for palette number (0=OBP0, 1=OBP1).
	SpriteAttrPalette = 1 << 4
)

const (
	// InterruptVBlank is the V-Blank interrupt bit.
	InterruptVBlank = 0
	// InterruptSTAT is the LCD STAT interrupt bit.
	InterruptSTAT = 1
)

// PPU represents the Game Boy Picture Processing Unit.
type PPU struct {
	// Video memory
	vram [VRAMSize]uint8 // VRAM (0x8000-0x9FFF)
	oam  [OAMSize]uint8  // Object Attribute Memory (0xFE00-0xFE9F)

	// Registers
	lcdc uint8 // LCD Control (0xFF40)
	stat uint8 // LCD Status (0xFF41)
	scy  uint8 // Scroll Y (0xFF42)
	scx  uint8 // Scroll X (0xFF43)
	ly   uint8 // Current Scanline (0xFF44)
	lyc  uint8 // LY Compare (0xFF45)
	bgp  uint8 // Background Palette (0xFF47)
	obp0 uint8 // Object Palette 0 (0xFF48)
	obp1 uint8 // Object Palette 1 (0xFF49)
	wy   uint8 // Window Y Position (0xFF4A)
	wx   uint8 // Window X Position + 7 (0xFF4B)

	// State
	mode uint8  // Current PPU mode (0-3)
	dots uint16 // Dot counter for current scanline

	// Framebuffer: 160x144 pixels, 2 bits per pixel (color index 0-3)
	framebuffer [ScreenWidth * ScreenHeight]uint8

	// Interrupt request callback
	requestInterrupt func(interrupt uint8)
}

// New creates a new PPU instance.
func New(requestInterrupt func(uint8)) *PPU {
	ppu := &PPU{
		requestInterrupt: requestInterrupt,
		mode:             ModeOAMScan,
		ly:               0,
		dots:             0,
	}

	// Initialize registers to power-up state
	ppu.lcdc = 0x91 // LCD on, BG on
	ppu.stat = 0x00
	ppu.bgp = 0xFC // Default palette: 11 11 11 00
	ppu.obp0 = 0xFF
	ppu.obp1 = 0xFF

	return ppu
}

// Step advances the PPU by the specified number of dots (T-cycles).
func (p *PPU) Step(cycles uint8) {
	// Only run if LCD is enabled
	if p.lcdc&LCDCLCDEnable == 0 {
		return
	}

	p.dots += uint16(cycles)

	// Check if we need to transition modes or scanlines
	switch p.mode {
	case ModeOAMScan:
		if p.dots >= DotsOAMScan {
			p.setMode(ModeDrawing)
			p.dots -= DotsOAMScan
		}

	case ModeDrawing:
		if p.dots >= DotsDrawing {
			p.setMode(ModeHBlank)
			p.dots -= DotsDrawing
			// Render the current scanline
			p.renderScanline()
		}

	case ModeHBlank:
		if p.dots >= DotsHBlank {
			p.dots -= DotsHBlank
			p.ly++

			if p.ly >= ScanlinesVisible {
				// Enter V-Blank
				p.setMode(ModeVBlank)
				if p.requestInterrupt != nil {
					p.requestInterrupt(InterruptVBlank)
				}
			} else {
				// Next scanline
				p.setMode(ModeOAMScan)
			}
		}

	case ModeVBlank:
		if p.dots >= DotsPerScanline {
			p.dots -= DotsPerScanline
			p.ly++

			if p.ly >= ScanlinesTotal {
				// Start new frame
				p.ly = 0
				p.setMode(ModeOAMScan)
			}
		}
	}

	// Update LYC=LY flag
	p.updateLYCFlag()
}

// setMode changes the PPU mode and updates STAT register.
func (p *PPU) setMode(mode uint8) {
	p.mode = mode
	p.stat = (p.stat &^ STATModeMask) | (mode & STATModeMask)

	// Trigger STAT interrupt if enabled for this mode
	if p.requestInterrupt != nil {
		triggerInterrupt := false

		switch mode {
		case ModeHBlank:
			triggerInterrupt = p.stat&STATMode0Interrupt != 0
		case ModeVBlank:
			triggerInterrupt = p.stat&STATMode1Interrupt != 0
		case ModeOAMScan:
			triggerInterrupt = p.stat&STATMode2Interrupt != 0
		}

		if triggerInterrupt {
			p.requestInterrupt(InterruptSTAT)
		}
	}
}

// updateLYCFlag updates the LYC=LY flag in STAT register.
func (p *PPU) updateLYCFlag() {
	if p.ly == p.lyc {
		p.stat |= STATLYCFlag
		// Trigger STAT interrupt if LYC interrupt is enabled
		if p.stat&STATLYCInterrupt != 0 && p.requestInterrupt != nil {
			p.requestInterrupt(InterruptSTAT)
		}
	} else {
		p.stat &^= STATLYCFlag
	}
}

// ReadVRAM reads a byte from VRAM.
func (p *PPU) ReadVRAM(addr uint16) uint8 {
	// VRAM is inaccessible during mode 3 (drawing)
	if p.mode == ModeDrawing {
		return 0xFF
	}
	if addr < VRAMSize {
		return p.vram[addr]
	}
	return 0xFF
}

// WriteVRAM writes a byte to VRAM.
func (p *PPU) WriteVRAM(addr uint16, value uint8) {
	// VRAM is inaccessible during mode 3 (drawing)
	if p.mode == ModeDrawing {
		return
	}
	if addr < VRAMSize {
		p.vram[addr] = value
	}
}

// ReadOAM reads a byte from OAM.
func (p *PPU) ReadOAM(addr uint16) uint8 {
	// OAM is inaccessible during modes 2 (OAM scan) and 3 (drawing)
	if p.mode == ModeOAMScan || p.mode == ModeDrawing {
		return 0xFF
	}
	if addr < OAMSize {
		return p.oam[addr]
	}
	return 0xFF
}

// WriteOAM writes a byte to OAM.
func (p *PPU) WriteOAM(addr uint16, value uint8) {
	// OAM is inaccessible during modes 2 (OAM scan) and 3 (drawing)
	if p.mode == ModeOAMScan || p.mode == ModeDrawing {
		return
	}
	if addr < OAMSize {
		p.oam[addr] = value
	}
}

// ReadRegister reads a PPU register.
func (p *PPU) ReadRegister(addr uint16) uint8 {
	switch addr {
	case 0xFF40:
		return p.lcdc
	case 0xFF41:
		return p.stat | 0x80 // Bit 7 is always 1
	case 0xFF42:
		return p.scy
	case 0xFF43:
		return p.scx
	case 0xFF44:
		return p.ly
	case 0xFF45:
		return p.lyc
	case 0xFF47:
		return p.bgp
	case 0xFF48:
		return p.obp0
	case 0xFF49:
		return p.obp1
	case 0xFF4A:
		return p.wy
	case 0xFF4B:
		return p.wx
	default:
		return 0xFF
	}
}

// WriteRegister writes to a PPU register.
func (p *PPU) WriteRegister(addr uint16, value uint8) {
	switch addr {
	case 0xFF40:
		p.lcdc = value
	case 0xFF41:
		// Only bits 6-3 are writable
		p.stat = (p.stat & 0x87) | (value & 0x78)
	case 0xFF42:
		p.scy = value
	case 0xFF43:
		p.scx = value
	case 0xFF44:
		// LY is read-only; writing resets it to 0
		p.ly = 0
	case 0xFF45:
		p.lyc = value
		p.updateLYCFlag()
	case 0xFF47:
		p.bgp = value
	case 0xFF48:
		p.obp0 = value
	case 0xFF49:
		p.obp1 = value
	case 0xFF4A:
		p.wy = value
	case 0xFF4B:
		p.wx = value
	}
}

// GetFramebuffer returns a pointer to the framebuffer.
func (p *PPU) GetFramebuffer() *[ScreenWidth * ScreenHeight]uint8 {
	return &p.framebuffer
}

// Reset resets the PPU to initial state.
func (p *PPU) Reset() {
	p.vram = [VRAMSize]uint8{}
	p.oam = [OAMSize]uint8{}
	p.lcdc = 0x91
	p.stat = 0x00
	p.scy = 0
	p.scx = 0
	p.ly = 0
	p.lyc = 0
	p.bgp = 0xFC
	p.obp0 = 0xFF
	p.obp1 = 0xFF
	p.wy = 0
	p.wx = 0
	p.mode = ModeOAMScan
	p.dots = 0
	p.framebuffer = [ScreenWidth * ScreenHeight]uint8{}
}
