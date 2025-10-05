// Package cartridge implements Game Boy cartridge loading and Memory Bank Controllers (MBCs).
package cartridge

import (
	"errors"
	"fmt"
)

// Header represents the Game Boy cartridge header (0x0100-0x014F).
type Header struct {
	// Entry point (0x0100-0x0103)
	EntryPoint [4]byte

	// Nintendo logo (0x0104-0x0133) - 48 bytes
	NintendoLogo [48]byte

	// Title (0x0134-0x0143) - 16 bytes
	// In older cartridges, this is 16 bytes
	// In newer cartridges, bytes 0x013F-0x0142 are manufacturer code
	// and 0x0143 is CGB flag
	Title [16]byte

	// Manufacturer code (0x013F-0x0142) - 4 bytes
	// Only in newer cartridges, overlaps with Title
	ManufacturerCode [4]byte

	// CGB flag (0x0143)
	// 0x80 = Game supports CGB functions, but works on old Game Boy
	// 0xC0 = Game works on CGB only
	CGBFlag byte

	// New licensee code (0x0144-0x0145)
	NewLicenseeCode [2]byte

	// SGB flag (0x0146)
	// 0x03 = Game supports SGB functions
	// 0x00 = No SGB support
	SGBFlag byte

	// Cartridge type (0x0147)
	CartridgeType byte

	// ROM size (0x0148)
	// 0x00 = 32 KiB (2 banks, no banking)
	// 0x01 = 64 KiB (4 banks)
	// 0x02 = 128 KiB (8 banks)
	// ... up to 0x08 = 8 MiB (512 banks)
	ROMSize byte

	// RAM size (0x0149)
	// 0x00 = No RAM
	// 0x01 = Unused
	// 0x02 = 8 KiB (1 bank)
	// 0x03 = 32 KiB (4 banks of 8 KiB)
	// 0x04 = 128 KiB (16 banks of 8 KiB)
	// 0x05 = 64 KiB (8 banks of 8 KiB)
	RAMSize byte

	// Destination code (0x014A)
	// 0x00 = Japan
	// 0x01 = Overseas
	DestinationCode byte

	// Old licensee code (0x014B)
	// 0x33 = Check new licensee code
	OldLicenseeCode byte

	// Mask ROM version (0x014C)
	MaskROMVersion byte

	// Header checksum (0x014D)
	HeaderChecksum byte

	// Global checksum (0x014E-0x014F)
	GlobalChecksum [2]byte
}

// CartridgeType represents the type of cartridge and MBC.
//
//nolint:revive // CartridgeType is intentionally explicit for clarity
type CartridgeType byte

// Cartridge types as defined in the header at 0x0147.
const (
	TypeROMOnly                    CartridgeType = 0x00
	TypeMBC1                       CartridgeType = 0x01
	TypeMBC1RAM                    CartridgeType = 0x02
	TypeMBC1RAMBattery             CartridgeType = 0x03
	TypeMBC2                       CartridgeType = 0x05
	TypeMBC2Battery                CartridgeType = 0x06
	TypeROMRAM                     CartridgeType = 0x08
	TypeROMRAMBattery              CartridgeType = 0x09
	TypeMMM01                      CartridgeType = 0x0B
	TypeMMM01RAM                   CartridgeType = 0x0C
	TypeMMM01RAMBattery            CartridgeType = 0x0D
	TypeMBC3TimerBattery           CartridgeType = 0x0F
	TypeMBC3TimerRAMBattery        CartridgeType = 0x10
	TypeMBC3                       CartridgeType = 0x11
	TypeMBC3RAM                    CartridgeType = 0x12
	TypeMBC3RAMBattery             CartridgeType = 0x13
	TypeMBC5                       CartridgeType = 0x19
	TypeMBC5RAM                    CartridgeType = 0x1A
	TypeMBC5RAMBattery             CartridgeType = 0x1B
	TypeMBC5Rumble                 CartridgeType = 0x1C
	TypeMBC5RumbleRAM              CartridgeType = 0x1D
	TypeMBC5RumbleRAMBattery       CartridgeType = 0x1E
	TypeMBC6                       CartridgeType = 0x20
	TypeMBC7SensorRumbleRAMBattery CartridgeType = 0x22
	TypePocketCamera               CartridgeType = 0xFC
	TypeBandaiTAMA5                CartridgeType = 0xFD
	TypeHuC3                       CartridgeType = 0xFE
	TypeHuC1RAMBattery             CartridgeType = 0xFF
)

// String returns a human-readable name for the cartridge type.
func (t CartridgeType) String() string {
	switch t {
	case TypeROMOnly:
		return "ROM ONLY"
	case TypeMBC1:
		return "MBC1"
	case TypeMBC1RAM:
		return "MBC1+RAM"
	case TypeMBC1RAMBattery:
		return "MBC1+RAM+BATTERY"
	case TypeMBC2:
		return "MBC2"
	case TypeMBC2Battery:
		return "MBC2+BATTERY"
	case TypeROMRAM:
		return "ROM+RAM"
	case TypeROMRAMBattery:
		return "ROM+RAM+BATTERY"
	case TypeMMM01:
		return "MMM01"
	case TypeMMM01RAM:
		return "MMM01+RAM"
	case TypeMMM01RAMBattery:
		return "MMM01+RAM+BATTERY"
	case TypeMBC3TimerBattery:
		return "MBC3+TIMER+BATTERY"
	case TypeMBC3TimerRAMBattery:
		return "MBC3+TIMER+RAM+BATTERY"
	case TypeMBC3:
		return "MBC3"
	case TypeMBC3RAM:
		return "MBC3+RAM"
	case TypeMBC3RAMBattery:
		return "MBC3+RAM+BATTERY"
	case TypeMBC5:
		return "MBC5"
	case TypeMBC5RAM:
		return "MBC5+RAM"
	case TypeMBC5RAMBattery:
		return "MBC5+RAM+BATTERY"
	case TypeMBC5Rumble:
		return "MBC5+RUMBLE"
	case TypeMBC5RumbleRAM:
		return "MBC5+RUMBLE+RAM"
	case TypeMBC5RumbleRAMBattery:
		return "MBC5+RUMBLE+RAM+BATTERY"
	case TypeMBC6:
		return "MBC6"
	case TypeMBC7SensorRumbleRAMBattery:
		return "MBC7+SENSOR+RUMBLE+RAM+BATTERY"
	case TypePocketCamera:
		return "POCKET CAMERA"
	case TypeBandaiTAMA5:
		return "BANDAI TAMA5"
	case TypeHuC3:
		return "HuC3"
	case TypeHuC1RAMBattery:
		return "HuC1+RAM+BATTERY"
	default:
		return fmt.Sprintf("UNKNOWN (0x%02X)", byte(t))
	}
}

// HasRAM returns true if the cartridge type includes RAM.
func (t CartridgeType) HasRAM() bool {
	switch t {
	case TypeMBC1RAM, TypeMBC1RAMBattery,
		TypeMBC2, TypeMBC2Battery, // MBC2 has built-in RAM
		TypeROMRAM, TypeROMRAMBattery,
		TypeMMM01RAM, TypeMMM01RAMBattery,
		TypeMBC3TimerRAMBattery, TypeMBC3RAM, TypeMBC3RAMBattery,
		TypeMBC5RAM, TypeMBC5RAMBattery,
		TypeMBC5RumbleRAM, TypeMBC5RumbleRAMBattery,
		TypeMBC7SensorRumbleRAMBattery,
		TypeHuC1RAMBattery:
		return true
	default:
		return false
	}
}

// HasBattery returns true if the cartridge type includes a battery for save data.
func (t CartridgeType) HasBattery() bool {
	switch t {
	case TypeMBC1RAMBattery,
		TypeMBC2Battery,
		TypeROMRAMBattery,
		TypeMMM01RAMBattery,
		TypeMBC3TimerBattery, TypeMBC3TimerRAMBattery, TypeMBC3RAMBattery,
		TypeMBC5RAMBattery, TypeMBC5RumbleRAMBattery,
		TypeMBC7SensorRumbleRAMBattery,
		TypeHuC1RAMBattery:
		return true
	default:
		return false
	}
}

// GetROMBanks returns the number of ROM banks based on the ROM size byte.
func (h *Header) GetROMBanks() int {
	// ROM size formula: 32 KiB << ROMSize = banks * 16 KiB
	// So banks = 2 << ROMSize
	if h.ROMSize <= 0x08 {
		return 2 << h.ROMSize
	}
	return 0 // Invalid
}

// GetROMSizeBytes returns the total ROM size in bytes.
func (h *Header) GetROMSizeBytes() int {
	return h.GetROMBanks() * 16384 // 16 KiB per bank
}

// GetRAMBanks returns the number of RAM banks based on the RAM size byte.
func (h *Header) GetRAMBanks() int {
	switch h.RAMSize {
	case 0x00:
		return 0 // No RAM
	case 0x01:
		return 0 // Unused (some sources say 2 KiB, but typically unused)
	case 0x02:
		return 1 // 8 KiB (1 bank)
	case 0x03:
		return 4 // 32 KiB (4 banks of 8 KiB)
	case 0x04:
		return 16 // 128 KiB (16 banks of 8 KiB)
	case 0x05:
		return 8 // 64 KiB (8 banks of 8 KiB)
	default:
		return 0 // Invalid
	}
}

// GetRAMSizeBytes returns the total RAM size in bytes.
func (h *Header) GetRAMSizeBytes() int {
	banks := h.GetRAMBanks()
	if banks == 0 && h.RAMSize == 0x01 {
		return 2048 // 2 KiB for the unused value
	}
	return banks * 8192 // 8 KiB per bank
}

// GetTitle returns the cartridge title as a string, trimmed of null bytes.
func (h *Header) GetTitle() string {
	// Find the first null byte
	end := len(h.Title)
	for i, b := range h.Title {
		if b == 0 {
			end = i
			break
		}
	}
	return string(h.Title[:end])
}

// ErrInvalidROMSize indicates the ROM data is too small to contain a valid header.
var ErrInvalidROMSize = errors.New("ROM too small: must be at least 336 bytes (0x0150)")

// ErrInvalidHeaderChecksum indicates the header checksum is invalid.
var ErrInvalidHeaderChecksum = errors.New("invalid header checksum")

// ParseHeader parses the cartridge header from ROM data.
func ParseHeader(rom []byte) (*Header, error) {
	if len(rom) < 0x0150 {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidROMSize, len(rom))
	}

	h := &Header{}

	// Entry point (0x0100-0x0103)
	copy(h.EntryPoint[:], rom[0x0100:0x0104])

	// Nintendo logo (0x0104-0x0133)
	copy(h.NintendoLogo[:], rom[0x0104:0x0134])

	// Title (0x0134-0x0143)
	copy(h.Title[:], rom[0x0134:0x0144])

	// Manufacturer code (overlaps with title at 0x013F-0x0142)
	copy(h.ManufacturerCode[:], rom[0x013F:0x0143])

	// CGB flag (0x0143)
	h.CGBFlag = rom[0x0143]

	// New licensee code (0x0144-0x0145)
	copy(h.NewLicenseeCode[:], rom[0x0144:0x0146])

	// SGB flag (0x0146)
	h.SGBFlag = rom[0x0146]

	// Cartridge type (0x0147)
	h.CartridgeType = rom[0x0147]

	// ROM size (0x0148)
	h.ROMSize = rom[0x0148]

	// RAM size (0x0149)
	h.RAMSize = rom[0x0149]

	// Destination code (0x014A)
	h.DestinationCode = rom[0x014A]

	// Old licensee code (0x014B)
	h.OldLicenseeCode = rom[0x014B]

	// Mask ROM version (0x014C)
	h.MaskROMVersion = rom[0x014C]

	// Header checksum (0x014D)
	h.HeaderChecksum = rom[0x014D]

	// Global checksum (0x014E-0x014F)
	copy(h.GlobalChecksum[:], rom[0x014E:0x0150])

	// Verify header checksum
	if !h.VerifyHeaderChecksum(rom) {
		return nil, ErrInvalidHeaderChecksum
	}

	return h, nil
}

// VerifyHeaderChecksum verifies the header checksum.
// The checksum is calculated over bytes 0x0134-0x014C.
// Formula: checksum = 0; for each byte: checksum = checksum - byte - 1.
func (h *Header) VerifyHeaderChecksum(rom []byte) bool {
	checksum := byte(0)
	for addr := 0x0134; addr <= 0x014C; addr++ {
		checksum = checksum - rom[addr] - 1
	}
	return checksum == h.HeaderChecksum
}

// VerifyGlobalChecksum verifies the global checksum.
// The global checksum is a 16-bit checksum of the entire ROM excluding the checksum bytes.
// Note: Many commercial games have incorrect global checksums, so this is often not enforced.
func (h *Header) VerifyGlobalChecksum(rom []byte) bool {
	sum := uint16(0)
	for i, b := range rom {
		// Skip the global checksum bytes at 0x014E-0x014F
		if i == 0x014E || i == 0x014F {
			continue
		}
		sum += uint16(b)
	}

	expected := (uint16(h.GlobalChecksum[0]) << 8) | uint16(h.GlobalChecksum[1])
	return sum == expected
}
