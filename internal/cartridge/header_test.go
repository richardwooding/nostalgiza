package cartridge

import (
	"errors"
	"testing"
)

func TestParseHeader(t *testing.T) {
	// Create a minimal valid ROM with header
	rom := make([]byte, 0x8000) // 32 KiB

	// Set up a valid header
	// Entry point (0x0100-0x0103) - typically a NOP and JP instruction
	rom[0x0100] = 0x00 // NOP
	rom[0x0101] = 0xC3 // JP
	rom[0x0102] = 0x50
	rom[0x0103] = 0x01

	// Nintendo logo (0x0104-0x0133) - partial, just for testing
	nintendoLogo := []byte{
		0xCE, 0xED, 0x66, 0x66, 0xCC, 0x0D, 0x00, 0x0B,
		0x03, 0x73, 0x00, 0x83, 0x00, 0x0C, 0x00, 0x0D,
		0x00, 0x08, 0x11, 0x1F, 0x88, 0x89, 0x00, 0x0E,
		0xDC, 0xCC, 0x6E, 0xE6, 0xDD, 0xDD, 0xD9, 0x99,
		0xBB, 0xBB, 0x67, 0x63, 0x6E, 0x0E, 0xEC, 0xCC,
		0xDD, 0xDC, 0x99, 0x9F, 0xBB, 0xB9, 0x33, 0x3E,
	}
	copy(rom[0x0104:], nintendoLogo)

	// Title (0x0134-0x0143)
	title := "TETRIS"
	copy(rom[0x0134:], []byte(title))

	// CGB flag (0x0143) - 0x00 for DMG only
	rom[0x0143] = 0x00

	// New licensee code (0x0144-0x0145)
	rom[0x0144] = '0'
	rom[0x0145] = '1'

	// SGB flag (0x0146) - 0x00 for no SGB support
	rom[0x0146] = 0x00

	// Cartridge type (0x0147) - 0x00 for ROM only
	rom[0x0147] = 0x00

	// ROM size (0x0148) - 0x00 for 32 KiB
	rom[0x0148] = 0x00

	// RAM size (0x0149) - 0x00 for no RAM
	rom[0x0149] = 0x00

	// Destination code (0x014A) - 0x01 for overseas
	rom[0x014A] = 0x01

	// Old licensee code (0x014B)
	rom[0x014B] = 0x33 // Use new licensee code

	// Mask ROM version (0x014C)
	rom[0x014C] = 0x00

	// Calculate and set header checksum (0x014D)
	checksum := byte(0)
	for addr := 0x0134; addr <= 0x014C; addr++ {
		checksum = checksum - rom[addr] - 1
	}
	rom[0x014D] = checksum

	// Global checksum (0x014E-0x014F) - calculate for completeness
	globalSum := uint16(0)
	for i, b := range rom {
		if i != 0x014E && i != 0x014F {
			globalSum += uint16(b)
		}
	}
	rom[0x014E] = byte(globalSum >> 8)
	rom[0x014F] = byte(globalSum & 0xFF)

	// Parse header
	header, err := ParseHeader(rom)
	if err != nil {
		t.Fatalf("ParseHeader() error = %v", err)
	}

	// Verify parsed values
	if header.GetTitle() != title {
		t.Errorf("Title = %q, want %q", header.GetTitle(), title)
	}

	if header.CartridgeType != 0x00 {
		t.Errorf("CartridgeType = 0x%02X, want 0x00", header.CartridgeType)
	}

	if header.ROMSize != 0x00 {
		t.Errorf("ROMSize = 0x%02X, want 0x00", header.ROMSize)
	}

	if header.RAMSize != 0x00 {
		t.Errorf("RAMSize = 0x%02X, want 0x00", header.RAMSize)
	}

	if header.CGBFlag != 0x00 {
		t.Errorf("CGBFlag = 0x%02X, want 0x00", header.CGBFlag)
	}
}

func TestParseHeaderTooSmall(t *testing.T) {
	rom := make([]byte, 0x0100) // Too small

	_, err := ParseHeader(rom)
	if err == nil {
		t.Error("ParseHeader() expected error for too small ROM, got nil")
	}

	if !errors.Is(err, ErrInvalidROMSize) {
		t.Errorf("ParseHeader() error = %v, want error wrapping %v", err, ErrInvalidROMSize)
	}
}

func TestHeaderChecksumValidation(t *testing.T) {
	rom := make([]byte, 0x8000)

	// Set up minimal header
	copy(rom[0x0134:], []byte("TEST"))
	rom[0x0147] = 0x00 // ROM only
	rom[0x0148] = 0x00 // 32 KiB
	rom[0x0149] = 0x00 // No RAM

	// Calculate correct checksum
	checksum := byte(0)
	for addr := 0x0134; addr <= 0x014C; addr++ {
		checksum = checksum - rom[addr] - 1
	}
	rom[0x014D] = checksum

	// Should parse successfully
	header, err := ParseHeader(rom)
	if err != nil {
		t.Fatalf("ParseHeader() with valid checksum error = %v", err)
	}

	if !header.VerifyHeaderChecksum(rom) {
		t.Error("VerifyHeaderChecksum() = false, want true")
	}

	// Now corrupt the checksum
	rom[0x014D] = 0xFF

	// Should fail
	_, err = ParseHeader(rom)
	if err == nil {
		t.Error("ParseHeader() with invalid checksum expected error, got nil")
	}

	if !errors.Is(err, ErrInvalidHeaderChecksum) {
		t.Errorf("ParseHeader() error = %v, want %v", err, ErrInvalidHeaderChecksum)
	}
}

func TestGetROMBanks(t *testing.T) {
	tests := []struct {
		romSize byte
		want    int
	}{
		{0x00, 2},   // 32 KiB = 2 banks
		{0x01, 4},   // 64 KiB = 4 banks
		{0x02, 8},   // 128 KiB = 8 banks
		{0x03, 16},  // 256 KiB = 16 banks
		{0x04, 32},  // 512 KiB = 32 banks
		{0x05, 64},  // 1 MiB = 64 banks
		{0x06, 128}, // 2 MiB = 128 banks
		{0x07, 256}, // 4 MiB = 256 banks
		{0x08, 512}, // 8 MiB = 512 banks
		{0x09, 0},   // Invalid
		{0xFF, 0},   // Invalid
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			h := &Header{ROMSize: tt.romSize}
			got := h.GetROMBanks()
			if got != tt.want {
				t.Errorf("GetROMBanks() with ROMSize=0x%02X = %d, want %d",
					tt.romSize, got, tt.want)
			}
		})
	}
}

func TestGetRAMBanks(t *testing.T) {
	tests := []struct {
		ramSize byte
		want    int
	}{
		{0x00, 0},  // No RAM
		{0x01, 0},  // Unused
		{0x02, 1},  // 8 KiB = 1 bank
		{0x03, 4},  // 32 KiB = 4 banks
		{0x04, 16}, // 128 KiB = 16 banks
		{0x05, 8},  // 64 KiB = 8 banks
		{0x06, 0},  // Invalid
		{0xFF, 0},  // Invalid
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			h := &Header{RAMSize: tt.ramSize}
			got := h.GetRAMBanks()
			if got != tt.want {
				t.Errorf("GetRAMBanks() with RAMSize=0x%02X = %d, want %d",
					tt.ramSize, got, tt.want)
			}
		})
	}
}

func TestGetROMSizeBytes(t *testing.T) {
	tests := []struct {
		romSize byte
		want    int
	}{
		{0x00, 32768},    // 32 KiB
		{0x01, 65536},    // 64 KiB
		{0x02, 131072},   // 128 KiB
		{0x05, 1048576},  // 1 MiB
		{0x08, 8388608},  // 8 MiB
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			h := &Header{ROMSize: tt.romSize}
			got := h.GetROMSizeBytes()
			if got != tt.want {
				t.Errorf("GetROMSizeBytes() with ROMSize=0x%02X = %d, want %d",
					tt.romSize, got, tt.want)
			}
		})
	}
}

func TestGetRAMSizeBytes(t *testing.T) {
	tests := []struct {
		ramSize byte
		want    int
	}{
		{0x00, 0},      // No RAM
		{0x01, 2048},   // 2 KiB (unused value)
		{0x02, 8192},   // 8 KiB
		{0x03, 32768},  // 32 KiB
		{0x04, 131072}, // 128 KiB
		{0x05, 65536},  // 64 KiB
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			h := &Header{RAMSize: tt.ramSize}
			got := h.GetRAMSizeBytes()
			if got != tt.want {
				t.Errorf("GetRAMSizeBytes() with RAMSize=0x%02X = %d, want %d",
					tt.ramSize, got, tt.want)
			}
		})
	}
}

func TestCartridgeTypeString(t *testing.T) {
	tests := []struct {
		cartType CartridgeType
		want     string
	}{
		{TypeROMOnly, "ROM ONLY"},
		{TypeMBC1, "MBC1"},
		{TypeMBC1RAM, "MBC1+RAM"},
		{TypeMBC1RAMBattery, "MBC1+RAM+BATTERY"},
		{TypeMBC3RAMBattery, "MBC3+RAM+BATTERY"},
		{TypeMBC5, "MBC5"},
		{CartridgeType(0xAB), "UNKNOWN (0xAB)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.cartType.String()
			if got != tt.want {
				t.Errorf("CartridgeType(0x%02X).String() = %q, want %q",
					byte(tt.cartType), got, tt.want)
			}
		})
	}
}

func TestCartridgeTypeHasRAM(t *testing.T) {
	tests := []struct {
		cartType CartridgeType
		want     bool
	}{
		{TypeROMOnly, false},
		{TypeMBC1, false},
		{TypeMBC1RAM, true},
		{TypeMBC1RAMBattery, true},
		{TypeMBC2, true}, // MBC2 has built-in RAM
		{TypeMBC2Battery, true},
		{TypeMBC3, false},
		{TypeMBC3RAM, true},
		{TypeMBC3RAMBattery, true},
		{TypeMBC5, false},
		{TypeMBC5RAM, true},
	}

	for _, tt := range tests {
		t.Run(tt.cartType.String(), func(t *testing.T) {
			got := tt.cartType.HasRAM()
			if got != tt.want {
				t.Errorf("CartridgeType(0x%02X).HasRAM() = %v, want %v",
					byte(tt.cartType), got, tt.want)
			}
		})
	}
}

func TestCartridgeTypeHasBattery(t *testing.T) {
	tests := []struct {
		cartType CartridgeType
		want     bool
	}{
		{TypeROMOnly, false},
		{TypeMBC1, false},
		{TypeMBC1RAM, false},
		{TypeMBC1RAMBattery, true},
		{TypeMBC2, false},
		{TypeMBC2Battery, true},
		{TypeMBC3, false},
		{TypeMBC3RAM, false},
		{TypeMBC3RAMBattery, true},
		{TypeMBC5, false},
		{TypeMBC5RAM, false},
		{TypeMBC5RAMBattery, true},
	}

	for _, tt := range tests {
		t.Run(tt.cartType.String(), func(t *testing.T) {
			got := tt.cartType.HasBattery()
			if got != tt.want {
				t.Errorf("CartridgeType(0x%02X).HasBattery() = %v, want %v",
					byte(tt.cartType), got, tt.want)
			}
		})
	}
}

func TestGetTitle(t *testing.T) {
	tests := []struct {
		name  string
		title [16]byte
		want  string
	}{
		{
			name:  "Full title",
			title: [16]byte{'T', 'E', 'T', 'R', 'I', 'S', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			want:  "TETRIS",
		},
		{
			name:  "Title with no nulls",
			title: [16]byte{'S', 'U', 'P', 'E', 'R', 'M', 'A', 'R', 'I', 'O', 'L', 'A', 'N', 'D', '1', '2'},
			want:  "SUPERMARIOLAND12",
		},
		{
			name:  "Empty title",
			title: [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			want:  "",
		},
		{
			name:  "Short title",
			title: [16]byte{'G', 'B', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			want:  "GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Header{Title: tt.title}
			got := h.GetTitle()
			if got != tt.want {
				t.Errorf("GetTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}
