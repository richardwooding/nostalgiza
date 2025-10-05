package cartridge

// MBC1 represents a cartridge with MBC1 (Memory Bank Controller 1).
// MBC1 is the most common MBC type, supporting up to 2 MiB of ROM and 32 KiB of RAM.
//
// Memory Map:
// - 0x0000-0x3FFF: ROM Bank 00 (fixed, or bank 0x00/0x20/0x40/0x60 in advanced mode)
// - 0x4000-0x7FFF: ROM Bank 01-7F (switchable)
// - 0xA000-0xBFFF: RAM Bank 00-03 (switchable, if present)
//
// Control Registers (write-only):
// - 0x0000-0x1FFF: RAM Enable (write 0x0A to enable, anything else disables)
// - 0x2000-0x3FFF: ROM Bank Number (lower 5 bits)
// - 0x4000-0x5FFF: RAM Bank Number / ROM Bank Number (upper 2 bits)
// - 0x6000-0x7FFF: Banking Mode Select (0 = simple ROM, 1 = advanced RAM/ROM)
type MBC1 struct {
	header *Header
	rom    []byte
	ram    []byte

	// Banking control
	ramEnabled bool   // RAM enable flag (0x0000-0x1FFF)
	romBank    uint8  // ROM bank number (0x2000-0x3FFF), 5 bits
	ramBank    uint8  // RAM bank number (0x4000-0x5FFF), 2 bits
	bankingMode uint8 // Banking mode (0x6000-0x7FFF): 0 = simple, 1 = advanced

	// Calculated values
	numROMBanks int
	numRAMBanks int
}

// newMBC1 creates a new MBC1 cartridge.
func newMBC1(rom []byte, header *Header) (*MBC1, error) {
	cart := &MBC1{
		header:      header,
		rom:         rom,
		ramEnabled:  false,
		romBank:     1, // Bank 0 is not allowed, so default to 1
		ramBank:     0,
		bankingMode: 0,
		numROMBanks: header.GetROMBanks(),
		numRAMBanks: header.GetRAMBanks(),
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
func (c *MBC1) Read(addr uint16) uint8 {
	switch {
	// ROM Bank 00 (0x0000-0x3FFF)
	case addr < 0x4000:
		// In advanced banking mode, this can be bank 0x00/0x20/0x40/0x60
		bankNumber := 0
		if c.bankingMode == 1 {
			// Use upper 2 bits from ramBank as upper bits of ROM bank
			bankNumber = int(c.ramBank) << 5
		}

		// Ensure bank is within bounds
		if bankNumber >= c.numROMBanks {
			bankNumber = bankNumber % c.numROMBanks
		}

		offset := bankNumber*0x4000 + int(addr)
		if offset < len(c.rom) {
			return c.rom[offset]
		}
		return 0xFF

	// ROM Bank 01-7F (0x4000-0x7FFF)
	case addr < 0x8000:
		// Combine lower 5 bits (romBank) with upper 2 bits (ramBank)
		bankNumber := int(c.romBank) | (int(c.ramBank) << 5)

		// Handle special case: banks 0x00, 0x20, 0x40, 0x60 are not accessible here
		// If lower 5 bits are 0, use 1 instead
		if (bankNumber & 0x1F) == 0 {
			bankNumber |= 0x01
		}

		// Wrap to available ROM banks
		if bankNumber >= c.numROMBanks {
			bankNumber = bankNumber % c.numROMBanks
		}

		offset := bankNumber*0x4000 + int(addr-0x4000)
		if offset < len(c.rom) {
			return c.rom[offset]
		}
		return 0xFF

	// External RAM (0xA000-0xBFFF)
	case addr >= 0xA000 && addr < 0xC000:
		if !c.ramEnabled || c.ram == nil {
			return 0xFF
		}

		// In advanced banking mode, ramBank selects RAM bank
		// In simple mode, ramBank is always 0
		bankNumber := 0
		if c.bankingMode == 1 && c.numRAMBanks > 1 {
			bankNumber = int(c.ramBank)
			if bankNumber >= c.numRAMBanks {
				bankNumber = bankNumber % c.numRAMBanks
			}
		}

		offset := bankNumber*0x2000 + int(addr-0xA000)
		if offset < len(c.ram) {
			return c.ram[offset]
		}
		return 0xFF

	default:
		return 0xFF
	}
}

// Write writes a byte to the cartridge (MBC control registers or RAM).
func (c *MBC1) Write(addr uint16, value uint8) {
	switch {
	// RAM Enable (0x0000-0x1FFF)
	case addr < 0x2000:
		// Lower 4 bits must be 0x0A to enable RAM
		c.ramEnabled = (value & 0x0F) == 0x0A

	// ROM Bank Number - lower 5 bits (0x2000-0x3FFF)
	case addr < 0x4000:
		// Only lower 5 bits are used
		c.romBank = value & 0x1F

		// Special case: writing 0x00 is treated as 0x01
		if c.romBank == 0 {
			c.romBank = 1
		}

	// RAM Bank Number / ROM Bank Number upper bits (0x4000-0x5FFF)
	case addr < 0x6000:
		// Only lower 2 bits are used
		c.ramBank = value & 0x03

	// Banking Mode Select (0x6000-0x7FFF)
	case addr < 0x8000:
		// Only bit 0 is used
		c.bankingMode = value & 0x01

	// External RAM (0xA000-0xBFFF)
	case addr >= 0xA000 && addr < 0xC000:
		if !c.ramEnabled || c.ram == nil {
			return
		}

		// In advanced banking mode, ramBank selects RAM bank
		// In simple mode, ramBank is always 0
		bankNumber := 0
		if c.bankingMode == 1 && c.numRAMBanks > 1 {
			bankNumber = int(c.ramBank)
			if bankNumber >= c.numRAMBanks {
				bankNumber = bankNumber % c.numRAMBanks
			}
		}

		offset := bankNumber*0x2000 + int(addr-0xA000)
		if offset < len(c.ram) {
			c.ram[offset] = value
		}
	}
}

// Header returns the cartridge header.
func (c *MBC1) Header() *Header {
	return c.header
}

// HasBattery returns true if the cartridge has battery-backed RAM.
func (c *MBC1) HasBattery() bool {
	return CartridgeType(c.header.CartridgeType).HasBattery()
}

// GetRAM returns the cartridge RAM for saving.
func (c *MBC1) GetRAM() []byte {
	if c.ram == nil {
		return nil
	}
	// Return a copy to prevent external modification
	ramCopy := make([]byte, len(c.ram))
	copy(ramCopy, c.ram)
	return ramCopy
}

// SetRAM loads save data into the cartridge RAM.
func (c *MBC1) SetRAM(data []byte) error {
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
