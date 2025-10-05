package cartridge

import (
	"errors"
	"fmt"
)

// Cartridge represents a Game Boy cartridge with ROM and optional RAM.
type Cartridge interface {
	// Read reads a byte from the cartridge address space (0x0000-0x7FFF for ROM, 0xA000-0xBFFF for RAM)
	Read(addr uint16) uint8

	// Write writes a byte to the cartridge address space (typically for MBC control or RAM)
	Write(addr uint16, value uint8)

	// Header returns the parsed cartridge header
	Header() *Header

	// HasBattery returns true if the cartridge has battery-backed RAM
	HasBattery() bool

	// GetRAM returns the cartridge RAM for saving (if battery-backed)
	GetRAM() []byte

	// SetRAM loads save data into the cartridge RAM (if battery-backed)
	SetRAM(data []byte) error
}

// ErrInvalidCartridgeType indicates an unsupported or unknown cartridge type.
var ErrInvalidCartridgeType = errors.New("invalid or unsupported cartridge type")

// ErrROMSizeMismatch indicates the ROM size doesn't match the header.
var ErrROMSizeMismatch = errors.New("ROM size does not match header")

// ErrROMTooLarge indicates the ROM size exceeds the maximum allowed size.
var ErrROMTooLarge = errors.New("ROM size exceeds maximum allowed size of 8 MiB")

// New creates a new cartridge from ROM data.
// It automatically detects the cartridge type from the header and creates
// the appropriate implementation (ROM-only, MBC1, MBC3, MBC5, etc.).
func New(rom []byte) (Cartridge, error) {
	// Check maximum ROM size (8 MiB)
	const maxROMSize = 8 * 1024 * 1024 // 8 MiB
	if len(rom) > maxROMSize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrROMTooLarge, len(rom))
	}

	// Parse header
	header, err := ParseHeader(rom)
	if err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	// Verify ROM size matches header
	expectedSize := header.GetROMSizeBytes()
	if len(rom) < expectedSize {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d",
			ErrROMSizeMismatch, expectedSize, len(rom))
	}

	// Create cartridge based on type
	cartType := CartridgeType(header.CartridgeType)

	switch cartType {
	case TypeROMOnly, TypeROMRAM, TypeROMRAMBattery:
		return newROMOnly(rom, header)

	case TypeMBC1, TypeMBC1RAM, TypeMBC1RAMBattery:
		return newMBC1(rom, header)

	default:
		return nil, fmt.Errorf("%w: type 0x%02X (%s)",
			ErrInvalidCartridgeType, byte(cartType), cartType.String())
	}
}
