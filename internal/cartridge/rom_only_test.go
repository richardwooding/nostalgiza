package cartridge

import (
	"testing"
)

func TestROMOnlyRead(t *testing.T) {
	// Create a minimal ROM
	rom := make([]byte, 0x8000) // 32 KiB
	rom[0x0100] = 0x42
	rom[0x4000] = 0x84
	rom[0x7FFF] = 0xFF

	// Set up header
	setupMinimalHeader(rom, 0x00, 0x00) // ROM only, no RAM

	header, err := ParseHeader(rom)
	if err != nil {
		t.Fatalf("ParseHeader() error = %v", err)
	}

	cart, err := newROMOnly(rom, header)
	if err != nil {
		t.Fatalf("newROMOnly() error = %v", err)
	}

	// Test ROM reads
	if got := cart.Read(0x0100); got != 0x42 {
		t.Errorf("Read(0x0100) = 0x%02X, want 0x42", got)
	}

	if got := cart.Read(0x4000); got != 0x84 {
		t.Errorf("Read(0x4000) = 0x%02X, want 0x84", got)
	}

	if got := cart.Read(0x7FFF); got != 0xFF {
		t.Errorf("Read(0x7FFF) = 0x%02X, want 0xFF", got)
	}

	// Test out of bounds - should return 0xFF
	if got := cart.Read(0x8000); got != 0xFF {
		t.Errorf("Read(0x8000) out of bounds = 0x%02X, want 0xFF", got)
	}
}

func TestROMOnlyWriteIgnored(t *testing.T) {
	rom := make([]byte, 0x8000)
	rom[0x0100] = 0x42

	setupMinimalHeader(rom, 0x00, 0x00)

	header, err := ParseHeader(rom)
	if err != nil {
		t.Fatalf("ParseHeader() error = %v", err)
	}

	cart, err := newROMOnly(rom, header)
	if err != nil {
		t.Fatalf("newROMOnly() error = %v", err)
	}

	// Try to write to ROM - should be ignored
	cart.Write(0x0100, 0xFF)

	// Verify ROM is unchanged
	if got := cart.Read(0x0100); got != 0x42 {
		t.Errorf("Read(0x0100) after write = 0x%02X, want 0x42 (write should be ignored)", got)
	}
}

func TestROMOnlyWithRAM(t *testing.T) {
	rom := make([]byte, 0x8000)

	// Set up header with RAM (type 0x08 = ROM+RAM)
	setupMinimalHeader(rom, 0x08, 0x02) // ROM+RAM, 8 KiB RAM

	header, err := ParseHeader(rom)
	if err != nil {
		t.Fatalf("ParseHeader() error = %v", err)
	}

	cart, err := newROMOnly(rom, header)
	if err != nil {
		t.Fatalf("newROMOnly() error = %v", err)
	}

	// Verify RAM is initialized
	if cart.ram == nil {
		t.Fatal("RAM should be initialized for ROM+RAM cartridge")
	}

	if len(cart.ram) != 8192 {
		t.Errorf("RAM size = %d, want 8192", len(cart.ram))
	}

	// Test RAM write and read
	cart.Write(0xA000, 0x42)
	if got := cart.Read(0xA000); got != 0x42 {
		t.Errorf("Read(0xA000) after write = 0x%02X, want 0x42", got)
	}

	cart.Write(0xBFFF, 0x99)
	if got := cart.Read(0xBFFF); got != 0x99 {
		t.Errorf("Read(0xBFFF) after write = 0x%02X, want 0x99", got)
	}

	// Test RAM out of bounds
	cart.Write(0xA000+8192, 0xFF)
	if got := cart.Read(0xA000 + 8192); got != 0xFF {
		t.Errorf("Read out of RAM bounds = 0x%02X, want 0xFF", got)
	}
}

func TestROMOnlyNoRAM(t *testing.T) {
	rom := make([]byte, 0x8000)
	setupMinimalHeader(rom, 0x00, 0x00) // ROM only, no RAM

	header, err := ParseHeader(rom)
	if err != nil {
		t.Fatalf("ParseHeader() error = %v", err)
	}

	cart, err := newROMOnly(rom, header)
	if err != nil {
		t.Fatalf("newROMOnly() error = %v", err)
	}

	// Verify no RAM
	if cart.ram != nil {
		t.Error("RAM should be nil for ROM-only cartridge")
	}

	// Reading from RAM area should return 0xFF
	if got := cart.Read(0xA000); got != 0xFF {
		t.Errorf("Read(0xA000) with no RAM = 0x%02X, want 0xFF", got)
	}

	// Writing to RAM area should be ignored (no crash)
	cart.Write(0xA000, 0x42) // Should not panic
}

func TestROMOnlyHasBattery(t *testing.T) {
	tests := []struct {
		name     string
		cartType byte
		want     bool
	}{
		{"ROM only", 0x00, false},
		{"ROM+RAM", 0x08, false},
		{"ROM+RAM+Battery", 0x09, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rom := make([]byte, 0x8000)
			setupMinimalHeader(rom, tt.cartType, 0x00)

			header, err := ParseHeader(rom)
			if err != nil {
				t.Fatalf("ParseHeader() error = %v", err)
			}

			cart, err := newROMOnly(rom, header)
			if err != nil {
				t.Fatalf("newROMOnly() error = %v", err)
			}

			if got := cart.HasBattery(); got != tt.want {
				t.Errorf("HasBattery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestROMOnlyGetSetRAM(t *testing.T) {
	rom := make([]byte, 0x8000)
	setupMinimalHeader(rom, 0x09, 0x02) // ROM+RAM+Battery, 8 KiB RAM

	header, err := ParseHeader(rom)
	if err != nil {
		t.Fatalf("ParseHeader() error = %v", err)
	}

	cart, err := newROMOnly(rom, header)
	if err != nil {
		t.Fatalf("newROMOnly() error = %v", err)
	}

	// Write some data to RAM
	cart.Write(0xA000, 0x11)
	cart.Write(0xA001, 0x22)
	cart.Write(0xA100, 0x33)

	// Get RAM
	ramData := cart.GetRAM()
	if ramData == nil {
		t.Fatal("GetRAM() returned nil")
	}

	if len(ramData) != 8192 {
		t.Errorf("GetRAM() length = %d, want 8192", len(ramData))
	}

	// Verify data
	if ramData[0] != 0x11 {
		t.Errorf("GetRAM()[0] = 0x%02X, want 0x11", ramData[0])
	}
	if ramData[1] != 0x22 {
		t.Errorf("GetRAM()[1] = 0x%02X, want 0x22", ramData[1])
	}
	if ramData[0x100] != 0x33 {
		t.Errorf("GetRAM()[0x100] = 0x%02X, want 0x33", ramData[0x100])
	}

	// Modify returned data should not affect internal RAM
	ramData[0] = 0xFF
	if cart.ram[0] != 0x11 {
		t.Error("Modifying GetRAM() result should not affect internal RAM")
	}

	// Test SetRAM
	newData := make([]byte, 8192)
	newData[0] = 0xAA
	newData[1] = 0xBB

	err = cart.SetRAM(newData)
	if err != nil {
		t.Errorf("SetRAM() error = %v", err)
	}

	// Verify data was loaded
	if got := cart.Read(0xA000); got != 0xAA {
		t.Errorf("Read(0xA000) after SetRAM = 0x%02X, want 0xAA", got)
	}
	if got := cart.Read(0xA001); got != 0xBB {
		t.Errorf("Read(0xA001) after SetRAM = 0x%02X, want 0xBB", got)
	}
}

func TestROMOnlyGetSetRAMNoRAM(t *testing.T) {
	rom := make([]byte, 0x8000)
	setupMinimalHeader(rom, 0x00, 0x00) // ROM only, no RAM

	header, err := ParseHeader(rom)
	if err != nil {
		t.Fatalf("ParseHeader() error = %v", err)
	}

	cart, err := newROMOnly(rom, header)
	if err != nil {
		t.Fatalf("newROMOnly() error = %v", err)
	}

	// GetRAM should return nil
	if ramData := cart.GetRAM(); ramData != nil {
		t.Errorf("GetRAM() with no RAM = %v, want nil", ramData)
	}

	// SetRAM should not error
	err = cart.SetRAM([]byte{0x11, 0x22})
	if err != nil {
		t.Errorf("SetRAM() with no RAM error = %v, want nil", err)
	}
}

func TestROMOnlyHeader(t *testing.T) {
	rom := make([]byte, 0x8000)
	title := "TESTROM"
	copy(rom[0x0134:], []byte(title))
	setupMinimalHeader(rom, 0x00, 0x00)

	header, err := ParseHeader(rom)
	if err != nil {
		t.Fatalf("ParseHeader() error = %v", err)
	}

	cart, err := newROMOnly(rom, header)
	if err != nil {
		t.Fatalf("newROMOnly() error = %v", err)
	}

	// Verify header is accessible
	h := cart.Header()
	if h == nil {
		t.Fatal("Header() returned nil")
	}

	if got := h.GetTitle(); got != title {
		t.Errorf("Header().GetTitle() = %q, want %q", got, title)
	}
}

// Helper function to set up minimal valid header for testing.
func setupMinimalHeader(rom []byte, cartType, ramSize byte) {
	// Title
	copy(rom[0x0134:], []byte("TEST"))

	// Cartridge type
	rom[0x0147] = cartType

	// ROM size (0x00 = 32 KiB)
	rom[0x0148] = 0x00

	// RAM size
	rom[0x0149] = ramSize

	// Calculate header checksum
	checksum := byte(0)
	for addr := 0x0134; addr <= 0x014C; addr++ {
		checksum = checksum - rom[addr] - 1
	}
	rom[0x014D] = checksum

	// Global checksum (optional, but we'll calculate it)
	globalSum := uint16(0)
	for i, b := range rom {
		if i != 0x014E && i != 0x014F {
			globalSum += uint16(b)
		}
	}
	rom[0x014E] = byte(globalSum >> 8)
	rom[0x014F] = byte(globalSum & 0xFF)
}
