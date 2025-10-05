package cartridge

import (
	"errors"
	"testing"
)

// TestNewUnsupportedMBCTypes verifies that attempting to load a ROM with an
// unsupported MBC type fails gracefully with ErrInvalidCartridgeType.
func TestNewUnsupportedMBCTypes(t *testing.T) {
	tests := []struct {
		name         string
		cartType     CartridgeType
		expectedType byte
	}{
		{"MBC2", TypeMBC2, 0x05},
		{"MBC2+Battery", TypeMBC2Battery, 0x06},
		{"MMM01", TypeMMM01, 0x0B},
		{"MMM01+RAM", TypeMMM01RAM, 0x0C},
		{"MMM01+RAM+Battery", TypeMMM01RAMBattery, 0x0D},
		{"MBC3+Timer+Battery", TypeMBC3TimerBattery, 0x0F},
		{"MBC3+Timer+RAM+Battery", TypeMBC3TimerRAMBattery, 0x10},
		{"MBC3", TypeMBC3, 0x11},
		{"MBC3+RAM", TypeMBC3RAM, 0x12},
		{"MBC3+RAM+Battery", TypeMBC3RAMBattery, 0x13},
		{"MBC5", TypeMBC5, 0x19},
		{"MBC5+RAM", TypeMBC5RAM, 0x1A},
		{"MBC5+RAM+Battery", TypeMBC5RAMBattery, 0x1B},
		{"MBC5+Rumble", TypeMBC5Rumble, 0x1C},
		{"MBC5+Rumble+RAM", TypeMBC5RumbleRAM, 0x1D},
		{"MBC5+Rumble+RAM+Battery", TypeMBC5RumbleRAMBattery, 0x1E},
		{"MBC6", TypeMBC6, 0x20},
		{"MBC7+Sensor+Rumble+RAM+Battery", TypeMBC7SensorRumbleRAMBattery, 0x22},
		{"Pocket Camera", TypePocketCamera, 0xFC},
		{"Bandai TAMA5", TypeBandaiTAMA5, 0xFD},
		{"HuC3", TypeHuC3, 0xFE},
		{"HuC1+RAM+Battery", TypeHuC1RAMBattery, 0xFF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal ROM with the unsupported cartridge type
			rom := make([]byte, 0x8000) // 32 KiB

			// Set cartridge type
			rom[0x0147] = tt.expectedType

			// Set ROM size to 0x00 (32 KiB, no banking)
			rom[0x0148] = 0x00

			// Calculate and set header checksum
			checksum := byte(0)
			for addr := 0x0134; addr <= 0x014C; addr++ {
				checksum = checksum - rom[addr] - 1
			}
			rom[0x014D] = checksum

			// Attempt to create cartridge - should fail
			cart, err := New(rom)

			// Verify error is ErrInvalidCartridgeType
			if !errors.Is(err, ErrInvalidCartridgeType) {
				t.Errorf("Expected ErrInvalidCartridgeType, got: %v", err)
			}

			// Verify cartridge is nil
			if cart != nil {
				t.Errorf("Expected nil cartridge for unsupported type %s, got: %T", tt.name, cart)
			}

			// Verify error message contains type information
			if err != nil && err.Error() == "" {
				t.Errorf("Error message should not be empty")
			}
		})
	}
}

// TestNewInvalidROMSize verifies that loading a ROM with mismatched size fails.
func TestNewInvalidROMSize(t *testing.T) {
	// Create a ROM that claims to be 64 KiB but is only 32 KiB
	rom := make([]byte, 0x8000) // 32 KiB

	// Set cartridge type to ROM only
	rom[0x0147] = 0x00

	// Set ROM size to 0x01 (64 KiB, 4 banks) - mismatch!
	rom[0x0148] = 0x01

	// Calculate and set header checksum
	checksum := byte(0)
	for addr := 0x0134; addr <= 0x014C; addr++ {
		checksum = checksum - rom[addr] - 1
	}
	rom[0x014D] = checksum

	// Attempt to create cartridge - should fail
	cart, err := New(rom)

	// Verify error is ErrROMSizeMismatch
	if !errors.Is(err, ErrROMSizeMismatch) {
		t.Errorf("Expected ErrROMSizeMismatch, got: %v", err)
	}

	// Verify cartridge is nil
	if cart != nil {
		t.Errorf("Expected nil cartridge for mismatched ROM size, got: %T", cart)
	}
}

// TestNewTooSmallROM verifies that loading a ROM smaller than header size fails.
func TestNewTooSmallROM(t *testing.T) {
	// Create a ROM that's too small to contain a valid header
	rom := make([]byte, 0x0100) // Only 256 bytes

	// Attempt to create cartridge - should fail
	cart, err := New(rom)

	// Verify error is related to header parsing
	if err == nil {
		t.Error("Expected error for too-small ROM, got nil")
	}

	// Verify cartridge is nil
	if cart != nil {
		t.Errorf("Expected nil cartridge for too-small ROM, got: %T", cart)
	}
}

// TestNewROMTooLarge verifies that loading a ROM larger than 8 MiB fails.
func TestNewROMTooLarge(t *testing.T) {
	// Create a ROM that exceeds the maximum allowed size (8 MiB)
	const maxROMSize = 8 * 1024 * 1024
	rom := make([]byte, maxROMSize+1) // 8 MiB + 1 byte

	// Attempt to create cartridge - should fail immediately
	cart, err := New(rom)

	// Verify error is ErrROMTooLarge
	if !errors.Is(err, ErrROMTooLarge) {
		t.Errorf("Expected ErrROMTooLarge, got: %v", err)
	}

	// Verify cartridge is nil
	if cart != nil {
		t.Errorf("Expected nil cartridge for too-large ROM, got: %T", cart)
	}
}

// TestNewROMExactly8MiB verifies that a ROM exactly 8 MiB is allowed.
func TestNewROMExactly8MiB(t *testing.T) {
	// Create a ROM that is exactly 8 MiB (maximum allowed size)
	const maxROMSize = 8 * 1024 * 1024
	rom := make([]byte, maxROMSize)

	// Set cartridge type to ROM only
	rom[0x0147] = 0x00

	// Set ROM size to 0x08 (8 MiB, 512 banks)
	rom[0x0148] = 0x08

	// Calculate and set header checksum
	checksum := byte(0)
	for addr := 0x0134; addr <= 0x014C; addr++ {
		checksum = checksum - rom[addr] - 1
	}
	rom[0x014D] = checksum

	// Attempt to create cartridge - should succeed
	cart, err := New(rom)

	// Verify no error
	if err != nil {
		t.Errorf("Expected no error for 8 MiB ROM, got: %v", err)
	}

	// Verify cartridge is not nil
	if cart == nil {
		t.Error("Expected valid cartridge for 8 MiB ROM, got nil")
	}
}
