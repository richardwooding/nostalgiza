// Package memory implements the Game Boy memory bus and address space mapping.
package memory

import (
	"errors"
	"fmt"

	"github.com/richardwooding/nostalgiza/internal/cartridge"
	"github.com/richardwooding/nostalgiza/internal/timer"
)

// PPU is an interface for the Picture Processing Unit.
type PPU interface {
	ReadVRAM(addr uint16) uint8
	WriteVRAM(addr uint16, value uint8)
	ReadOAM(addr uint16) uint8
	WriteOAM(addr uint16, value uint8)
	ReadRegister(addr uint16) uint8
	WriteRegister(addr uint16, value uint8)
}

// Joypad is an interface for joypad input handling.
type Joypad interface {
	Read() uint8
	Write(value uint8)
}

// Bus represents the Game Boy memory bus.
type Bus struct {
	// Cartridge (ROM and external RAM are handled by cartridge)
	cartridge cartridge.Cartridge

	// PPU for video memory and registers
	ppu PPU

	// Joypad for input handling
	joypad Joypad

	// Timer for DIV, TIMA, TMA, TAC registers
	timer *timer.Timer

	// Work RAM (8 KiB)
	wram [0x2000]uint8 // C000-DFFF: Work RAM

	// I/O Registers (128 bytes)
	io [0x80]uint8 // FF00-FF7F: I/O Registers

	// High RAM (127 bytes)
	hram [0x7F]uint8 // FF80-FFFE: High RAM

	// Interrupt Enable Register (1 byte)
	ie uint8 // FFFF: Interrupt Enable

	// DMA state (Phase 3.5)
	dmaActive bool   // DMA transfer in progress
	dmaSource uint16 // DMA source address (XX00)
	dmaCycles uint16 // Remaining DMA cycles (160 total)
}

// NewBus creates a new memory bus.
func NewBus() *Bus {
	return &Bus{}
}

// SetCartridge sets the cartridge for the memory bus.
func (b *Bus) SetCartridge(cart cartridge.Cartridge) {
	b.cartridge = cart
}

// SetPPU sets the PPU for the memory bus.
func (b *Bus) SetPPU(ppu PPU) {
	b.ppu = ppu
}

// SetJoypad sets the joypad for the memory bus.
func (b *Bus) SetJoypad(joypad Joypad) {
	b.joypad = joypad
}

// SetTimer sets the timer for the memory bus.
func (b *Bus) SetTimer(t *timer.Timer) {
	b.timer = t
}

// Read reads a byte from the memory bus.
func (b *Bus) Read(addr uint16) uint8 {
	// During DMA transfer, only HRAM (0xFF80-0xFFFE) is accessible to CPU
	// All other reads return 0xFF (including OAM)
	if b.dmaActive && (addr < 0xFF80 || addr == 0xFFFF) {
		return 0xFF
	}

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
		if b.ppu != nil {
			return b.ppu.ReadVRAM(addr - 0x8000)
		}
		return 0xFF

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
		if b.ppu != nil {
			return b.ppu.ReadOAM(addr - 0xFE00)
		}
		return 0xFF

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
		if b.ppu != nil {
			b.ppu.WriteVRAM(addr-0x8000, value)
		}

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
		if b.ppu != nil {
			b.ppu.WriteOAM(addr-0xFE00, value)
		}

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
		if b.joypad != nil {
			return b.joypad.Read()
		}
		return 0xFF // No input pressed
	case 0xFF04, 0xFF05, 0xFF06, 0xFF07: // Timer registers
		if b.timer != nil {
			return b.timer.Read(addr)
		}
		return b.io[offset]
	case 0xFF0F: // IF - Interrupt flags
		return b.io[offset]
	case 0xFF40, 0xFF41, 0xFF42, 0xFF43, 0xFF44, 0xFF45, 0xFF47, 0xFF48, 0xFF49, 0xFF4A, 0xFF4B:
		// PPU registers (0xFF40-0xFF4B except 0xFF46)
		if b.ppu != nil {
			return b.ppu.ReadRegister(addr)
		}
		return 0xFF
	case 0xFF46: // DMA - DMA transfer
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
	case 0xFF00: // Joypad (P1)
		if b.joypad != nil {
			b.joypad.Write(value)
		}
	case 0xFF04, 0xFF05, 0xFF06, 0xFF07: // Timer registers
		if b.timer != nil {
			b.timer.Write(addr, value)
		} else {
			// Fallback for DIV reset behavior
			if addr == 0xFF04 {
				b.io[offset] = 0
			} else {
				b.io[offset] = value
			}
		}
	case 0xFF40, 0xFF41, 0xFF42, 0xFF43, 0xFF44, 0xFF45, 0xFF47, 0xFF48, 0xFF49, 0xFF4A, 0xFF4B:
		// PPU registers (0xFF40-0xFF4B except 0xFF46)
		if b.ppu != nil {
			b.ppu.WriteRegister(addr, value)
		}
	case 0xFF46: // DMA - DMA transfer
		// Initiate DMA transfer
		// Valid DMA source addresses are 0x00-0xF1 (0x0000-0xF100)
		// Addresses above 0xF1 would attempt to copy from restricted regions
		if value <= 0xF1 {
			b.dmaActive = true
			b.dmaSource = uint16(value) << 8 // Source address is XX00
			b.dmaCycles = 160                // DMA takes 160 M-cycles
		}
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

// Reset clears all RAM while keeping the cartridge and PPU loaded.
// Note: Cartridge RAM is not cleared as it may be battery-backed.
func (b *Bus) Reset() {
	// Clear Work RAM
	clear(b.wram[:])

	// Clear I/O registers
	clear(b.io[:])

	// Clear High RAM
	clear(b.hram[:])

	// Clear Interrupt Enable
	b.ie = 0

	// Clear DMA state
	b.dmaActive = false
	b.dmaSource = 0
	b.dmaCycles = 0
}

// StepDMA advances the DMA transfer by one M-cycle.
// Returns true if DMA is still active, false if transfer is complete or inactive.
// Should be called once per M-cycle when DMA is active.
func (b *Bus) StepDMA() bool {
	if !b.dmaActive {
		return false
	}

	// Calculate which byte to transfer (160 - remaining cycles)
	byteOffset := 160 - b.dmaCycles

	// Read from source address
	srcAddr := b.dmaSource + byteOffset
	value := b.dmaRead(srcAddr)

	// Write to OAM
	if b.ppu != nil {
		b.ppu.WriteOAM(byteOffset, value)
	}

	// Decrement cycles
	b.dmaCycles--

	// Check if transfer complete
	if b.dmaCycles == 0 {
		b.dmaActive = false
		return false
	}

	return true
}

// dmaRead performs a read for DMA transfer (bypasses DMA access restriction).
func (b *Bus) dmaRead(addr uint16) uint8 {
	switch {
	// ROM Bank 00 (0000-3FFF) and ROM Bank 01-NN (4000-7FFF)
	case addr < 0x8000:
		if b.cartridge != nil {
			return b.cartridge.Read(addr)
		}
		return 0xFF

	// VRAM (8000-9FFF)
	case addr < 0xA000:
		if b.ppu != nil {
			return b.ppu.ReadVRAM(addr - 0x8000)
		}
		return 0xFF

	// External RAM (A000-BFFF)
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

	// Echo RAM (E000-FDFF)
	case addr < 0xFE00:
		return b.wram[addr-0xE000]

	default:
		return 0xFF
	}
}
