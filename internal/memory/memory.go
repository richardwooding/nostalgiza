// Package memory implements the Game Boy memory bus and address space mapping.
package memory

import (
	"errors"
	"fmt"

	"github.com/richardwooding/nostalgiza/internal/cartridge"
)

// Bus represents the Game Boy memory bus.
type Bus struct {
	// Cartridge (ROM and external RAM are handled by cartridge)
	cartridge cartridge.Cartridge

	// VRAM (8 KiB)
	vram [0x2000]uint8 // 8000-9FFF: Video RAM

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

	// PPU mode for access restrictions (stub for now)
	ppuMode uint8
}

// NewBus creates a new memory bus.
func NewBus() *Bus {
	return &Bus{
		ppuMode: 0,
	}
}

// SetCartridge sets the cartridge for the memory bus.
func (b *Bus) SetCartridge(cart cartridge.Cartridge) {
	b.cartridge = cart
}

// Read reads a byte from the memory bus.
func (b *Bus) Read(addr uint16) uint8 {
	switch {
	// ROM Bank 00 (0000-3FFF) and ROM Bank 01-NN (4000-7FFF)
	// Handled by cartridge
	case addr < 0x8000:
		if b.cartridge != nil {
			return b.cartridge.Read(addr)
		}
		return 0xFF

	// VRAM (8000-9FFF)
	case addr < 0xA000:
		// TODO: Check PPU mode for access restrictions (Phase 3)
		return b.vram[addr-0x8000]

	// External RAM (A000-BFFF) - Handled by cartridge
	case addr < 0xC000:
		if b.cartridge != nil {
			return b.cartridge.Read(addr)
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
	// Handled by cartridge
	case addr < 0x8000:
		if b.cartridge != nil {
			b.cartridge.Write(addr, value)
		}

	// VRAM (8000-9FFF)
	case addr < 0xA000:
		// TODO: Check PPU mode for access restrictions (Phase 3)
		b.vram[addr-0x8000] = value

	// External RAM (A000-BFFF) - Handled by cartridge
	case addr < 0xC000:
		if b.cartridge != nil {
			b.cartridge.Write(addr, value)
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

// ErrROMLoadFailed indicates ROM loading failed.
var ErrROMLoadFailed = errors.New("ROM loading failed")

// LoadROM loads ROM data by creating a cartridge and attaching it to the bus.
func (b *Bus) LoadROM(rom []byte) error {
	cart, err := cartridge.New(rom)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrROMLoadFailed, err)
	}

	b.cartridge = cart
	return nil
}

// GetCartridge returns the currently loaded cartridge.
func (b *Bus) GetCartridge() cartridge.Cartridge {
	return b.cartridge
}

// Reset clears all RAM while keeping the cartridge loaded.
func (b *Bus) Reset() {
	// Clear VRAM
	for i := range b.vram {
		b.vram[i] = 0
	}

	// Clear Work RAM
	for i := range b.wram {
		b.wram[i] = 0
	}

	// Clear OAM
	for i := range b.oam {
		b.oam[i] = 0
	}

	// Clear I/O registers
	for i := range b.io {
		b.io[i] = 0
	}

	// Clear High RAM
	for i := range b.hram {
		b.hram[i] = 0
	}

	// Clear Interrupt Enable
	b.ie = 0

	// Reset PPU mode
	b.ppuMode = 0
}
