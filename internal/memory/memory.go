// Package memory implements the Game Boy memory bus and address space mapping.
package memory

import (
	"errors"
	"fmt"
)

// Bus represents the Game Boy memory bus.
type Bus struct {
	// ROM banks (16 KiB each)
	rom0 [0x4000]uint8 // 0000-3FFF: ROM Bank 00 (fixed)
	rom1 [0x4000]uint8 // 4000-7FFF: ROM Bank 01-NN (switchable)

	// VRAM (8 KiB)
	vram [0x2000]uint8 // 8000-9FFF: Video RAM

	// External RAM (8 KiB)
	extRAM [0x2000]uint8 // A000-BFFF: External RAM (cartridge)

	// Work RAM (8 KiB)
	wram [0x2000]uint8 // C000-DFFF: Work RAM

	// OAM (160 bytes)
	oam [0xA0]uint8 // FE00-FE9F: Object Attribute Memory

	// I/O Registers (128 bytes)
	io [0x80]uint8 // FF00-FF7F: I/O Registers

	// High RAM (127 bytes)
	hram [0x7F]uint8 // FF80-FFFE: High RAM

	// Interrupt Enable Register (1 byte)
	ie uint8 // FFFF: Interrupt Enable

	// Banking state
	currentROMBank int
	currentRAMBank int
	ramEnabled     bool

	// PPU mode for access restrictions (stub for now)
	ppuMode uint8
}

// NewBus creates a new memory bus.
func NewBus() *Bus {
	return &Bus{
		currentROMBank: 1,
		currentRAMBank: 0,
		ramEnabled:     false,
		ppuMode:        0,
	}
}

// Read reads a byte from the memory bus.
func (b *Bus) Read(addr uint16) uint8 {
	switch {
	// ROM Bank 00 (0000-3FFF)
	case addr < 0x4000:
		return b.rom0[addr]

	// ROM Bank 01-NN (4000-7FFF)
	case addr < 0x8000:
		return b.rom1[addr-0x4000]

	// VRAM (8000-9FFF)
	case addr < 0xA000:
		// TODO: Check PPU mode for access restrictions (Phase 3)
		return b.vram[addr-0x8000]

	// External RAM (A000-BFFF)
	case addr < 0xC000:
		if b.ramEnabled {
			return b.extRAM[addr-0xA000]
		}
		return 0xFF

	// Work RAM Bank 0 (C000-CFFF)
	case addr < 0xD000:
		return b.wram[addr-0xC000]

	// Work RAM Bank 1 (D000-DFFF)
	case addr < 0xE000:
		return b.wram[addr-0xC000]

	// Echo RAM (E000-FDFF) - Mirror of C000-DDFF
	case addr < 0xFE00:
		return b.wram[addr-0xE000]

	// OAM (FE00-FE9F)
	case addr < 0xFEA0:
		// TODO: Check PPU mode for access restrictions (Phase 3)
		return b.oam[addr-0xFE00]

	// Not Usable (FEA0-FEFF)
	case addr < 0xFF00:
		return 0xFF

	// I/O Registers (FF00-FF7F)
	case addr < 0xFF80:
		return b.readIO(addr)

	// High RAM (FF80-FFFE)
	case addr < 0xFFFF:
		return b.hram[addr-0xFF80]

	// Interrupt Enable Register (FFFF)
	case addr == 0xFFFF:
		return b.ie

	default:
		return 0xFF
	}
}

// Write writes a byte to the memory bus.
func (b *Bus) Write(addr uint16, value uint8) {
	switch {
	// ROM Bank 00 & 01 (0000-7FFF) - MBC control
	case addr < 0x8000:
		// TODO: Implement MBC banking (Phase 2)
		// For now, treat as read-only
		return

	// VRAM (8000-9FFF)
	case addr < 0xA000:
		// TODO: Check PPU mode for access restrictions (Phase 3)
		b.vram[addr-0x8000] = value

	// External RAM (A000-BFFF)
	case addr < 0xC000:
		if b.ramEnabled {
			b.extRAM[addr-0xA000] = value
		}

	// Work RAM Bank 0 (C000-CFFF)
	case addr < 0xD000:
		b.wram[addr-0xC000] = value

	// Work RAM Bank 1 (D000-DFFF)
	case addr < 0xE000:
		b.wram[addr-0xC000] = value

	// Echo RAM (E000-FDFF) - Mirror of C000-DDFF
	case addr < 0xFE00:
		b.wram[addr-0xE000] = value

	// OAM (FE00-FE9F)
	case addr < 0xFEA0:
		// TODO: Check PPU mode for access restrictions (Phase 3)
		b.oam[addr-0xFE00] = value

	// Not Usable (FEA0-FEFF)
	case addr < 0xFF00:
		// Ignore writes to unusable memory

	// I/O Registers (FF00-FF7F)
	case addr < 0xFF80:
		b.writeIO(addr, value)

	// High RAM (FF80-FFFE)
	case addr < 0xFFFF:
		b.hram[addr-0xFF80] = value

	// Interrupt Enable Register (FFFF)
	case addr == 0xFFFF:
		b.ie = value
	}
}

// readIO reads from I/O registers.
func (b *Bus) readIO(addr uint16) uint8 {
	// TODO: Implement proper I/O register handlers in later phases
	// For now, return stored value or default

	offset := addr - 0xFF00

	// Special cases for specific registers
	switch addr {
	case 0xFF00: // Joypad (P1)
		return 0xFF // No input pressed (Phase 4)
	case 0xFF04: // DIV - Divider register
		return b.io[offset]
	case 0xFF05: // TIMA - Timer counter
		return b.io[offset]
	case 0xFF06: // TMA - Timer modulo
		return b.io[offset]
	case 0xFF07: // TAC - Timer control
		return b.io[offset]
	case 0xFF0F: // IF - Interrupt flags
		return b.io[offset]
	case 0xFF40: // LCDC - LCD control
		return b.io[offset]
	case 0xFF41: // STAT - LCD status
		return b.io[offset]
	case 0xFF42: // SCY - Scroll Y
		return b.io[offset]
	case 0xFF43: // SCX - Scroll X
		return b.io[offset]
	case 0xFF44: // LY - LCD Y coordinate
		return b.io[offset]
	case 0xFF45: // LYC - LY compare
		return b.io[offset]
	case 0xFF46: // DMA - DMA transfer
		return b.io[offset]
	case 0xFF47: // BGP - BG palette
		return b.io[offset]
	case 0xFF48: // OBP0 - Object palette 0
		return b.io[offset]
	case 0xFF49: // OBP1 - Object palette 1
		return b.io[offset]
	case 0xFF4A: // WY - Window Y
		return b.io[offset]
	case 0xFF4B: // WX - Window X
		return b.io[offset]
	default:
		return b.io[offset]
	}
}

// writeIO writes to I/O registers.
func (b *Bus) writeIO(addr uint16, value uint8) {
	// TODO: Implement proper I/O register handlers in later phases
	// For now, just store the value

	offset := addr - 0xFF00

	// Special cases for specific registers
	switch addr {
	case 0xFF04: // DIV - Divider register (writing resets to 0)
		b.io[offset] = 0
	case 0xFF46: // DMA - DMA transfer
		// TODO: Implement DMA transfer (Phase 3)
		b.io[offset] = value
	default:
		b.io[offset] = value
	}
}

// ErrROMTooSmall indicates the ROM is smaller than the minimum 16 KiB.
var ErrROMTooSmall = errors.New("ROM too small: minimum is 16 KiB")

// LoadROM loads ROM data into memory banks.
// Minimum ROM size for Game Boy is 32 KiB (two 16 KiB banks).
func (b *Bus) LoadROM(rom []byte) error {
	if len(rom) < 0x4000 {
		return fmt.Errorf("%w: got %d bytes", ErrROMTooSmall, len(rom))
	}

	// Load ROM Bank 00 (first 16 KiB)
	copy(b.rom0[:], rom[:0x4000])

	// Load ROM Bank 01 (second 16 KiB if available)
	if len(rom) >= 0x8000 {
		copy(b.rom1[:], rom[0x4000:0x8000])
	} else if len(rom) > 0x4000 {
		// Partial bank - copy what's available
		copy(b.rom1[:], rom[0x4000:])
	}

	return nil
}
