package cartridge

import (
	"testing"
)

// setupMBC1Header sets up a minimal header and recalculates checksum.
// Call this after modifying any header fields (like ROM size).
func setupMBC1Header(rom []byte, cartType, ramSize, romSize byte) {
	setupMinimalHeader(rom, cartType, ramSize)
	rom[0x0148] = romSize

	// Recalculate checksum
	checksum := byte(0)
	for addr := 0x0134; addr <= 0x014C; addr++ {
		checksum = checksum - rom[addr] - 1
	}
	rom[0x014D] = checksum
}

func TestMBC1BasicROMBanking(t *testing.T) {
	// Create a 64 KiB ROM (4 banks)
	rom := make([]byte, 0x10000)

	// Put distinct values in each bank
	rom[0x0000] = 0x00 // Bank 0, first byte
	rom[0x4000] = 0x01 // Bank 1, first byte
	rom[0x8000] = 0x02 // Bank 2, first byte
	rom[0xC000] = 0x03 // Bank 3, first byte

	setupMBC1Header(rom, 0x01, 0x00, 0x01) // MBC1, no RAM, 64 KiB

	header, err := ParseHeader(rom)
	if err != nil {
		t.Fatalf("ParseHeader() error = %v", err)
	}

	cart, err := newMBC1(rom, header)
	if err != nil {
		t.Fatalf("newMBC1() error = %v", err)
	}

	// Initially, bank 0 should be at 0x0000-0x3FFF, bank 1 at 0x4000-0x7FFF
	if got := cart.Read(0x0000); got != 0x00 {
		t.Errorf("Read(0x0000) = 0x%02X, want 0x00", got)
	}
	if got := cart.Read(0x4000); got != 0x01 {
		t.Errorf("Read(0x4000) default bank 1 = 0x%02X, want 0x01", got)
	}

	// Switch to bank 2 (write to 0x2000-0x3FFF)
	cart.Write(0x2000, 0x02)
	if got := cart.Read(0x4000); got != 0x02 {
		t.Errorf("Read(0x4000) after switching to bank 2 = 0x%02X, want 0x02", got)
	}

	// Switch to bank 3
	cart.Write(0x2000, 0x03)
	if got := cart.Read(0x4000); got != 0x03 {
		t.Errorf("Read(0x4000) after switching to bank 3 = 0x%02X, want 0x03", got)
	}

	// Bank 0 should still be at 0x0000-0x3FFF
	if got := cart.Read(0x0000); got != 0x00 {
		t.Errorf("Read(0x0000) should still be bank 0 = 0x%02X, want 0x00", got)
	}
}

func TestMBC1BankZeroHandling(t *testing.T) {
	rom := make([]byte, 0x10000) // 64 KiB

	rom[0x4000] = 0x01 // Bank 1
	rom[0x8000] = 0x02 // Bank 2

	setupMBC1Header(rom, 0x01, 0x00, 0x01) // MBC1, no RAM, 64 KiB

	header, err := ParseHeader(rom)
	if err != nil {
		t.Fatalf("ParseHeader() error = %v", err)
	}
	cart, err := newMBC1(rom, header)
	if err != nil {
		t.Fatalf("newMBC1() error = %v", err)
	}

	// Writing 0x00 to ROM bank register should select bank 0x01
	cart.Write(0x2000, 0x00)
	if got := cart.Read(0x4000); got != 0x01 {
		t.Errorf("Read(0x4000) after writing 0x00 = 0x%02X, want 0x01 (bank 0 redirects to 1)", got)
	}

	// Explicitly writing 0x01 should also select bank 0x01
	cart.Write(0x2000, 0x01)
	if got := cart.Read(0x4000); got != 0x01 {
		t.Errorf("Read(0x4000) after writing 0x01 = 0x%02X, want 0x01", got)
	}
}

func TestMBC1RAMEnableDisable(t *testing.T) {
	rom := make([]byte, 0x8000)
	setupMinimalHeader(rom, 0x02, 0x02) // MBC1+RAM, 8 KiB

	header, _ := ParseHeader(rom)
	cart, _ := newMBC1(rom, header)

	// RAM should be disabled by default
	if cart.ramEnabled {
		t.Error("RAM should be disabled by default")
	}

	// Reading from RAM when disabled should return 0xFF
	if got := cart.Read(0xA000); got != 0xFF {
		t.Errorf("Read(0xA000) with RAM disabled = 0x%02X, want 0xFF", got)
	}

	// Writing should be ignored when disabled
	cart.Write(0xA000, 0x42)
	if got := cart.Read(0xA000); got != 0xFF {
		t.Errorf("RAM write when disabled should be ignored")
	}

	// Enable RAM (write 0x0A to 0x0000-0x1FFF)
	cart.Write(0x0000, 0x0A)
	if !cart.ramEnabled {
		t.Error("RAM should be enabled after writing 0x0A")
	}

	// Now RAM should be writable
	cart.Write(0xA000, 0x42)
	if got := cart.Read(0xA000); got != 0x42 {
		t.Errorf("Read(0xA000) with RAM enabled = 0x%02X, want 0x42", got)
	}

	// Disable RAM (write anything else)
	cart.Write(0x0000, 0x00)
	if cart.ramEnabled {
		t.Error("RAM should be disabled after writing 0x00")
	}

	// Reading should return 0xFF again
	if got := cart.Read(0xA000); got != 0xFF {
		t.Errorf("Read(0xA000) after disabling RAM = 0x%02X, want 0xFF", got)
	}
}

func TestMBC1RAMBanking(t *testing.T) {
	rom := make([]byte, 0x8000)
	setupMinimalHeader(rom, 0x03, 0x03) // MBC1+RAM+Battery, 32 KiB (4 banks)

	header, _ := ParseHeader(rom)
	cart, _ := newMBC1(rom, header)

	// Enable RAM
	cart.Write(0x0000, 0x0A)

	// Simple mode (default): only bank 0 is accessible
	cart.Write(0xA000, 0x11) // Write to bank 0

	// Switch to advanced mode
	cart.Write(0x6000, 0x01)

	// Now we can switch RAM banks
	cart.Write(0x4000, 0x00) // Select RAM bank 0
	cart.Write(0xA000, 0x22)

	cart.Write(0x4000, 0x01) // Select RAM bank 1
	cart.Write(0xA000, 0x33)

	cart.Write(0x4000, 0x02) // Select RAM bank 2
	cart.Write(0xA000, 0x44)

	cart.Write(0x4000, 0x03) // Select RAM bank 3
	cart.Write(0xA000, 0x55)

	// Read back from each bank
	cart.Write(0x4000, 0x00)
	if got := cart.Read(0xA000); got != 0x22 {
		t.Errorf("RAM bank 0 first byte = 0x%02X, want 0x22", got)
	}

	cart.Write(0x4000, 0x01)
	if got := cart.Read(0xA000); got != 0x33 {
		t.Errorf("RAM bank 1 first byte = 0x%02X, want 0x33", got)
	}

	cart.Write(0x4000, 0x02)
	if got := cart.Read(0xA000); got != 0x44 {
		t.Errorf("RAM bank 2 first byte = 0x%02X, want 0x44", got)
	}

	cart.Write(0x4000, 0x03)
	if got := cart.Read(0xA000); got != 0x55 {
		t.Errorf("RAM bank 3 first byte = 0x%02X, want 0x55", got)
	}
}

func TestMBC1AdvancedROMBanking(t *testing.T) {
	// Create a 2 MiB ROM (128 banks) to test upper bits
	rom := make([]byte, 2*1024*1024)

	// Mark specific banks
	rom[0x00000] = 0x00 // Bank 0x00
	rom[0x04000] = 0x01 // Bank 0x01
	rom[0x80000] = 0x20 // Bank 0x20 (32)
	rom[0x84000] = 0x21 // Bank 0x21 (33)

	setupMBC1Header(rom, 0x01, 0x00, 0x05) // MBC1, no RAM, 2 MiB

	header, err := ParseHeader(rom)
	if err != nil {
		t.Fatalf("ParseHeader() error = %v", err)
	}
	cart, err := newMBC1(rom, header)
	if err != nil {
		t.Fatalf("newMBC1() error = %v", err)
	}

	// Simple mode: can access banks 0x00-0x1F
	cart.Write(0x2000, 0x01) // Bank 0x01
	if got := cart.Read(0x4000); got != 0x01 {
		t.Errorf("Bank 0x01 = 0x%02X, want 0x01", got)
	}

	// To access bank 0x20+, need to use upper bits (ramBank register)
	cart.Write(0x4000, 0x01) // Upper 2 bits = 01 (banks 0x20-0x3F)
	cart.Write(0x2000, 0x00) // Lower 5 bits = 00 (but becomes 01)

	// This should select bank 0x21 (0x01 << 5 | 0x01)
	if got := cart.Read(0x4000); got != 0x21 {
		t.Errorf("Bank 0x21 = 0x%02X, want 0x21", got)
	}

	// Advanced mode: Bank 0 area can also be switched
	cart.Write(0x6000, 0x01) // Advanced mode
	cart.Write(0x4000, 0x01) // Upper bits = 01 (banks 0x20-0x3F)
	cart.Write(0x2000, 0x00) // Lower bits = 00

	// Now 0x0000-0x3FFF should point to bank 0x20
	if got := cart.Read(0x0000); got != 0x20 {
		t.Errorf("Advanced mode bank 0 area = 0x%02X, want 0x20", got)
	}

	// And 0x4000-0x7FFF should point to bank 0x21
	if got := cart.Read(0x4000); got != 0x21 {
		t.Errorf("Advanced mode bank 1 area = 0x%02X, want 0x21", got)
	}
}

func TestMBC1HasBattery(t *testing.T) {
	tests := []struct {
		name     string
		cartType byte
		want     bool
	}{
		{"MBC1", 0x01, false},
		{"MBC1+RAM", 0x02, false},
		{"MBC1+RAM+Battery", 0x03, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			setupMinimalHeader(rom, tt.cartType, 0x00)

			header, _ := ParseHeader(rom)
			cart, _ := newMBC1(rom, header)

			if got := cart.HasBattery(); got != tt.want {
				t.Errorf("HasBattery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMBC1GetSetRAM(t *testing.T) {
	rom := make([]byte, 0x8000)
	setupMinimalHeader(rom, 0x03, 0x02) // MBC1+RAM+Battery, 8 KiB

	header, _ := ParseHeader(rom)
	cart, _ := newMBC1(rom, header)

	// Enable RAM
	cart.Write(0x0000, 0x0A)

	// Write some data
	cart.Write(0xA000, 0xAA)
	cart.Write(0xA001, 0xBB)
	cart.Write(0xA100, 0xCC)

	// Get RAM
	ramData := cart.GetRAM()
	if ramData == nil {
		t.Fatal("GetRAM() returned nil")
	}

	if ramData[0] != 0xAA || ramData[1] != 0xBB || ramData[0x100] != 0xCC {
		t.Error("GetRAM() did not return correct data")
	}

	// Verify it's a copy
	ramData[0] = 0xFF
	if cart.ram[0] != 0xAA {
		t.Error("GetRAM() should return a copy")
	}

	// Test SetRAM
	newData := make([]byte, 8192)
	newData[0] = 0x11
	newData[1] = 0x22

	err := cart.SetRAM(newData)
	if err != nil {
		t.Errorf("SetRAM() error = %v", err)
	}

	if got := cart.Read(0xA000); got != 0x11 {
		t.Errorf("Read after SetRAM = 0x%02X, want 0x11", got)
	}
	if got := cart.Read(0xA001); got != 0x22 {
		t.Errorf("Read after SetRAM = 0x%02X, want 0x22", got)
	}
}

func TestMBC1NoRAM(t *testing.T) {
	rom := make([]byte, 0x8000)
	setupMinimalHeader(rom, 0x01, 0x00) // MBC1, no RAM

	header, _ := ParseHeader(rom)
	cart, _ := newMBC1(rom, header)

	// Verify no RAM
	if cart.ram != nil {
		t.Error("MBC1 without RAM should have nil ram")
	}

	// GetRAM should return nil
	if ramData := cart.GetRAM(); ramData != nil {
		t.Error("GetRAM() should return nil when no RAM")
	}

	// SetRAM should not error
	err := cart.SetRAM([]byte{0x11, 0x22})
	if err != nil {
		t.Errorf("SetRAM() with no RAM error = %v", err)
	}

	// Reading from RAM area should return 0xFF
	if got := cart.Read(0xA000); got != 0xFF {
		t.Errorf("Read from RAM area with no RAM = 0x%02X, want 0xFF", got)
	}
}

func TestMBC1Header(t *testing.T) {
	rom := make([]byte, 0x8000)
	setupMinimalHeader(rom, 0x01, 0x00)

	// Overwrite title after setup
	title := "MBC1TEST"
	copy(rom[0x0134:], []byte(title))

	// Recalculate checksum
	checksum := byte(0)
	for addr := 0x0134; addr <= 0x014C; addr++ {
		checksum = checksum - rom[addr] - 1
	}
	rom[0x014D] = checksum

	header, err := ParseHeader(rom)
	if err != nil {
		t.Fatalf("ParseHeader() error = %v", err)
	}
	cart, err := newMBC1(rom, header)
	if err != nil {
		t.Fatalf("newMBC1() error = %v", err)
	}

	h := cart.Header()
	if h == nil {
		t.Fatal("Header() returned nil")
	}

	if got := h.GetTitle(); got != title {
		t.Errorf("Header().GetTitle() = %q, want %q", got, title)
	}
}

func TestMBC1BankMasking(t *testing.T) {
	// Test with 64 KiB ROM (4 banks) - banks should wrap
	rom := make([]byte, 0x10000)
	rom[0x0000] = 0x00 // Bank 0
	rom[0x4000] = 0x01 // Bank 1
	rom[0x8000] = 0x02 // Bank 2
	rom[0xC000] = 0x03 // Bank 3

	setupMBC1Header(rom, 0x01, 0x00, 0x01) // MBC1, no RAM, 64 KiB (4 banks)

	header, err := ParseHeader(rom)
	if err != nil {
		t.Fatalf("ParseHeader() error = %v", err)
	}
	cart, err := newMBC1(rom, header)
	if err != nil {
		t.Fatalf("newMBC1() error = %v", err)
	}

	// Try to select bank 5 (should wrap to bank 1 since we only have 4 banks)
	// Bank 5 % 4 = 1
	cart.Write(0x2000, 0x05)
	if got := cart.Read(0x4000); got != 0x01 {
		t.Errorf("Bank wrapping: bank 5 should wrap to bank 1, got 0x%02X, want 0x01", got)
	}

	// Try to select bank 6 (should wrap to bank 2)
	cart.Write(0x2000, 0x06)
	if got := cart.Read(0x4000); got != 0x02 {
		t.Errorf("Bank wrapping: bank 6 should wrap to bank 2, got 0x%02X, want 0x02", got)
	}
}
