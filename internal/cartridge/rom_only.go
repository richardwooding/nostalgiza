package cartridge

// ROMOnly represents a simple ROM-only cartridge with no MBC.
// Supports up to 32 KiB of ROM and optional 8 KiB of RAM.
type ROMOnly struct {
	header *Header
	rom    []byte
	ram    []byte
}

// newROMOnly creates a new ROM-only cartridge.
func newROMOnly(rom []byte, header *Header) (*ROMOnly, error) {
	cart := &ROMOnly{
		header: header,
		rom:    rom,
	}

	// Initialize RAM if present
	if CartridgeType(header.CartridgeType).HasRAM() {
		ramSize := header.GetRAMSizeBytes()
		if ramSize > 0 {
			cart.ram = make([]byte, ramSize)
		}
	}

	return cart, nil
}

// Read reads a byte from the cartridge.
func (c *ROMOnly) Read(addr uint16) uint8 {
	switch {
	// ROM: 0x0000-0x7FFF
	case addr < 0x8000:
		if int(addr) < len(c.rom) {
			return c.rom[addr]
		}
		return 0xFF

	// External RAM: 0xA000-0xBFFF
	case addr >= 0xA000 && addr < 0xC000:
		if c.ram != nil {
			ramAddr := addr - 0xA000
			if int(ramAddr) < len(c.ram) {
				return c.ram[ramAddr]
			}
		}
		return 0xFF

	default:
		return 0xFF
	}
}

// Write writes a byte to the cartridge (only RAM is writable).
func (c *ROMOnly) Write(addr uint16, value uint8) {
	switch {
	// ROM: 0x0000-0x7FFF (read-only, writes ignored)
	case addr < 0x8000:
		// Writes to ROM are ignored

	// External RAM: 0xA000-0xBFFF
	case addr >= 0xA000 && addr < 0xC000:
		if c.ram != nil {
			ramAddr := addr - 0xA000
			if int(ramAddr) < len(c.ram) {
				c.ram[ramAddr] = value
			}
		}
	}
}

// Header returns the cartridge header.
func (c *ROMOnly) Header() *Header {
	return c.header
}

// HasBattery returns true if the cartridge has battery-backed RAM.
func (c *ROMOnly) HasBattery() bool {
	return CartridgeType(c.header.CartridgeType).HasBattery()
}

// GetRAM returns the cartridge RAM for saving.
func (c *ROMOnly) GetRAM() []byte {
	if c.ram == nil {
		return nil
	}
	// Return a copy to prevent external modification
	ramCopy := make([]byte, len(c.ram))
	copy(ramCopy, c.ram)
	return ramCopy
}

// SetRAM loads save data into the cartridge RAM.
func (c *ROMOnly) SetRAM(data []byte) error {
	if c.ram == nil {
		return nil // No RAM to load
	}

	// Copy data into RAM (up to RAM size)
	copyLen := len(data)
	if copyLen > len(c.ram) {
		copyLen = len(c.ram)
	}
	copy(c.ram, data[:copyLen])

	return nil
}
