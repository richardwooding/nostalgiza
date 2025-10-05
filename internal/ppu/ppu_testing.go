package ppu

// SetModeForTesting sets the PPU mode directly for testing purposes.
// This is used by memory tests to set the PPU to H-Blank mode so VRAM/OAM are accessible.
func (p *PPU) SetModeForTesting(mode uint8) {
	p.mode = mode
	p.stat = (p.stat &^ STATModeMask) | (mode & STATModeMask)
}
