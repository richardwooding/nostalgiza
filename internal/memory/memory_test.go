package memory

import (
	"testing"
)

func TestNewBus(t *testing.T) {
	bus := NewBus()

	if bus == nil {
		t.Fatal("NewBus() returned nil")
	}

	// Cartridge should be nil initially
	if bus.cartridge != nil {
		t.Error("cartridge should be nil initially")
	}
}

func TestROMAccess(t *testing.T) {
	bus := NewBus()

	// Load a simple ROM
	rom := make([]byte, 0x8000) // 32 KiB
	rom[0x0100] = 0x42
	rom[0x4000] = 0x84

	// Setup minimal header
	setupTestROMHeader(rom)

	err := bus.LoadROM(rom)
	if err != nil {
		t.Fatalf("LoadROM() error = %v", err)
	}

	// Test ROM Bank 00 (0000-3FFF)
	value := bus.Read(0x0100)
	if value != 0x42 {
		t.Errorf("Read(0x0100) = %02X, want 0x42", value)
	}

	// Test ROM Bank 01 (4000-7FFF)
	value = bus.Read(0x4000)
	if value != 0x84 {
		t.Errorf("Read(0x4000) = %02X, want 0x84", value)
	}

	// Writing to ROM should be ignored (MBC control)
	bus.Write(0x0100, 0xFF)
	value = bus.Read(0x0100)
	if value != 0x42 {
		t.Errorf("ROM should be read-only, got %02X", value)
	}
}

func TestWRAMAccess(t *testing.T) {
	bus := NewBus()

	// Test WRAM Bank 0 (C000-CFFF)
	bus.Write(0xC123, 0xAB)
	value := bus.Read(0xC123)
	if value != 0xAB {
		t.Errorf("Read(0xC123) = %02X, want 0xAB", value)
	}

	// Test WRAM Bank 1 (D000-DFFF)
	bus.Write(0xD456, 0xCD)
	value = bus.Read(0xD456)
	if value != 0xCD {
		t.Errorf("Read(0xD456) = %02X, want 0xCD", value)
	}
}

func TestEchoRAM(t *testing.T) {
	bus := NewBus()

	// Write to WRAM
	bus.Write(0xC123, 0x55)

	// Read from Echo RAM
	value := bus.Read(0xE123)
	if value != 0x55 {
		t.Errorf("Echo RAM Read(0xE123) = %02X, want 0x55", value)
	}

	// Write to Echo RAM
	bus.Write(0xE456, 0xAA)

	// Read from WRAM
	value = bus.Read(0xC456)
	if value != 0xAA {
		t.Errorf("WRAM Read(0xC456) = %02X, want 0xAA", value)
	}
}

func TestVRAMAccess(t *testing.T) {
	bus := NewBus()

	// Test VRAM (8000-9FFF)
	bus.Write(0x8000, 0x12)
	value := bus.Read(0x8000)
	if value != 0x12 {
		t.Errorf("Read(0x8000) = %02X, want 0x12", value)
	}

	bus.Write(0x9FFF, 0x34)
	value = bus.Read(0x9FFF)
	if value != 0x34 {
		t.Errorf("Read(0x9FFF) = %02X, want 0x34", value)
	}
}

func TestOAMAccess(t *testing.T) {
	bus := NewBus()

	// Test OAM (FE00-FE9F)
	bus.Write(0xFE00, 0x56)
	value := bus.Read(0xFE00)
	if value != 0x56 {
		t.Errorf("Read(0xFE00) = %02X, want 0x56", value)
	}

	bus.Write(0xFE9F, 0x78)
	value = bus.Read(0xFE9F)
	if value != 0x78 {
		t.Errorf("Read(0xFE9F) = %02X, want 0x78", value)
	}
}

func TestHRAMAccess(t *testing.T) {
	bus := NewBus()

	// Test HRAM (FF80-FFFE)
	bus.Write(0xFF80, 0x9A)
	value := bus.Read(0xFF80)
	if value != 0x9A {
		t.Errorf("Read(0xFF80) = %02X, want 0x9A", value)
	}

	bus.Write(0xFFFE, 0xBC)
	value = bus.Read(0xFFFE)
	if value != 0xBC {
		t.Errorf("Read(0xFFFE) = %02X, want 0xBC", value)
	}
}

func TestInterruptEnableRegister(t *testing.T) {
	bus := NewBus()

	// Test IE register (FFFF)
	bus.Write(0xFFFF, 0x1F)
	value := bus.Read(0xFFFF)
	if value != 0x1F {
		t.Errorf("Read(0xFFFF) = %02X, want 0x1F", value)
	}
}

func TestExternalRAM(t *testing.T) {
	bus := NewBus()

	// Load a ROM with MBC1+RAM (so RAM can be enabled/disabled)
	rom := make([]byte, 0x8000)
	setupTestROMHeader(rom)
	rom[0x0147] = 0x02 // MBC1+RAM
	rom[0x0149] = 0x02 // 8 KiB RAM

	// Recalculate checksum
	checksum := byte(0)
	for addr := 0x0134; addr <= 0x014C; addr++ {
		checksum = checksum - rom[addr] - 1
	}
	rom[0x014D] = checksum

	err := bus.LoadROM(rom)
	if err != nil {
		t.Fatalf("LoadROM() error = %v", err)
	}

	// External RAM should not be accessible when disabled
	bus.Write(0xA000, 0x42)
	value := bus.Read(0xA000)
	if value != 0xFF {
		t.Errorf("Read(0xA000) with RAM disabled = %02X, want 0xFF", value)
	}

	// Enable external RAM (write 0x0A to 0x0000-0x1FFF for MBC1 RAM enable)
	bus.Write(0x0000, 0x0A)

	// Now it should be accessible
	bus.Write(0xA000, 0x42)
	value = bus.Read(0xA000)
	if value != 0x42 {
		t.Errorf("Read(0xA000) with RAM enabled = %02X, want 0x42", value)
	}

	// Disable again
	bus.Write(0x0000, 0x00)
	value = bus.Read(0xA000)
	if value != 0xFF {
		t.Errorf("Read(0xA000) after disabling RAM = %02X, want 0xFF", value)
	}
}

func TestNotUsableMemory(t *testing.T) {
	bus := NewBus()

	// Not usable memory (FEA0-FEFF) should return 0xFF
	value := bus.Read(0xFEA0)
	if value != 0xFF {
		t.Errorf("Read(0xFEA0) = %02X, want 0xFF", value)
	}

	value = bus.Read(0xFEFF)
	if value != 0xFF {
		t.Errorf("Read(0xFEFF) = %02X, want 0xFF", value)
	}

	// Writes should be ignored
	bus.Write(0xFEA0, 0x42)
	value = bus.Read(0xFEA0)
	if value != 0xFF {
		t.Errorf("Not usable memory should ignore writes, got %02X", value)
	}
}

func TestIORegisters(t *testing.T) {
	bus := NewBus()

	// Test basic I/O register read/write
	bus.Write(0xFF40, 0x91) // LCDC
	value := bus.Read(0xFF40)
	if value != 0x91 {
		t.Errorf("Read(0xFF40) = %02X, want 0x91", value)
	}

	// Test DIV register (writing resets to 0)
	bus.Write(0xFF04, 0x42) // Write any value
	value = bus.Read(0xFF04)
	if value != 0x00 {
		t.Errorf("Read(0xFF04) after write = %02X, want 0x00 (DIV resets on write)", value)
	}

	// Test joypad register (returns 0xFF by default - no input)
	value = bus.Read(0xFF00)
	if value != 0xFF {
		t.Errorf("Read(0xFF00) = %02X, want 0xFF (no input)", value)
	}
}

func TestLoadROM(t *testing.T) {
	bus := NewBus()

	// Create a test ROM (32 KiB)
	rom := make([]byte, 0x8000)
	setupTestROMHeader(rom)

	rom[0x0100] = 0x00 // NOP at entry point
	rom[0x0104] = 0xCE // Nintendo logo byte
	rom[0x4000] = 0x42 // First byte of ROM bank 1

	err := bus.LoadROM(rom)
	if err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Check ROM Bank 00
	if bus.Read(0x0100) != 0x00 {
		t.Errorf("ROM Bank 00 not loaded correctly")
	}
	if bus.Read(0x0104) != 0xCE {
		t.Errorf("ROM Bank 00 header not loaded correctly")
	}

	// Check ROM Bank 01
	if bus.Read(0x4000) != 0x42 {
		t.Errorf("ROM Bank 01 not loaded correctly")
	}

	// Verify cartridge was loaded
	if bus.GetCartridge() == nil {
		t.Error("Cartridge should be loaded")
	}
}

func TestLoadROMSizeValidation(t *testing.T) {
	bus := NewBus()

	// Test ROM too small (less than 336 bytes for header)
	tooSmall := make([]byte, 0x100) // 256 bytes
	err := bus.LoadROM(tooSmall)
	if err == nil {
		t.Error("Expected error for ROM smaller than header size, got nil")
	}

	// Test standard 32 KiB ROM (matches header default)
	standardROM := make([]byte, 0x8000)
	setupTestROMHeader(standardROM)
	standardROM[0x4FFF] = 0x99
	err = bus.LoadROM(standardROM)
	if err != nil {
		t.Errorf("Expected no error for 32 KiB ROM, got: %v", err)
	}
	if bus.Read(0x4FFF) != 0x99 {
		t.Errorf("ROM not loaded correctly")
	}
}

func TestMemoryMap(t *testing.T) {
	bus := NewBus()

	// Test that each memory region is distinct
	testAddresses := []struct {
		addr  uint16
		value uint8
	}{
		{0x0100, 0x01}, // ROM Bank 00
		{0x4100, 0x02}, // ROM Bank 01
		{0x8100, 0x03}, // VRAM
		{0xC100, 0x04}, // WRAM Bank 0
		{0xD100, 0x05}, // WRAM Bank 1
		{0xFE00, 0x06}, // OAM
		{0xFF40, 0x07}, // I/O
		{0xFF80, 0x08}, // HRAM
		{0xFFFF, 0x09}, // IE
	}

	// Write to each region
	for _, tt := range testAddresses {
		bus.Write(tt.addr, tt.value)
	}

	// Read and verify
	for _, tt := range testAddresses {
		value := bus.Read(tt.addr)
		// ROM is read-only, so skip those checks
		if tt.addr < 0x8000 {
			continue
		}
		if value != tt.value {
			t.Errorf("Read(%04X) = %02X, want %02X", tt.addr, value, tt.value)
		}
	}
}

func TestMemoryBoundaries(t *testing.T) {
	bus := NewBus()

	tests := []struct {
		name     string
		addr     uint16
		value    uint8
		readable bool
		writable bool
	}{
		// ROM boundaries (read-only)
		{"ROM0 start", 0x0000, 0x11, true, false},
		{"ROM0 end", 0x3FFF, 0x22, true, false},
		{"ROM1 start", 0x4000, 0x33, true, false},
		{"ROM1 end", 0x7FFF, 0x44, true, false},

		// VRAM boundaries
		{"VRAM start", 0x8000, 0x55, true, true},
		{"VRAM end", 0x9FFF, 0x66, true, true},

		// External RAM boundaries
		{"ExtRAM start", 0xA000, 0x77, true, false}, // Disabled by default
		{"ExtRAM end", 0xBFFF, 0x88, true, false},

		// WRAM boundaries
		{"WRAM start", 0xC000, 0x99, true, true},
		{"WRAM end", 0xDFFF, 0xAA, true, true},

		// Echo RAM boundaries
		{"Echo start", 0xE000, 0xBB, true, true},
		{"Echo end", 0xFDFF, 0xCC, true, true},

		// OAM boundaries
		{"OAM start", 0xFE00, 0xDD, true, true},
		{"OAM end", 0xFE9F, 0xEE, true, true},

		// Not usable region
		{"Not usable start", 0xFEA0, 0xFF, true, false},
		{"Not usable end", 0xFEFF, 0xFF, true, false},

		// I/O boundaries (skip 0xFF00 joypad - has special default value 0xFF)
		{"I/O register", 0xFF01, 0x01, true, true},
		{"I/O end", 0xFF7F, 0x02, true, true},

		// HRAM boundaries
		{"HRAM start", 0xFF80, 0x03, true, true},
		{"HRAM end", 0xFFFE, 0x04, true, true},

		// IE register
		{"IE register", 0xFFFF, 0x05, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Try to write
			bus.Write(tt.addr, tt.value)

			// Try to read
			readValue := bus.Read(tt.addr)

			if tt.writable { //nolint:nestif // Test validation complexity is acceptable
				if readValue != tt.value {
					t.Errorf("Write/Read at 0x%04X: got 0x%02X, want 0x%02X", tt.addr, readValue, tt.value)
				}
			} else if tt.addr >= 0x8000 {
				// For writable=false (non-ROM), verify write was ignored
				// (Skip ROM regions as they may have initial data)
				if tt.addr >= 0xFEA0 && tt.addr < 0xFF00 {
					if readValue != 0xFF {
						t.Errorf("Not usable region at 0x%04X should return 0xFF, got 0x%02X", tt.addr, readValue)
					}
				}
			}
		})
	}
}

func TestEchoRAMMirroring(t *testing.T) {
	bus := NewBus()

	// Write to WRAM and verify it's mirrored in Echo RAM
	testCases := []struct {
		wramAddr uint16
		echoAddr uint16
		value    uint8
	}{
		{0xC000, 0xE000, 0x42},
		{0xC100, 0xE100, 0x99},
		{0xD000, 0xF000, 0xAB},
		{0xDDFF, 0xFDFF, 0xCD},
	}

	for _, tc := range testCases {
		t.Run("WRAM->Echo", func(t *testing.T) {
			// Write to WRAM
			bus.Write(tc.wramAddr, tc.value)

			// Read from Echo RAM - should be same value
			echoValue := bus.Read(tc.echoAddr)
			if echoValue != tc.value {
				t.Errorf("Echo RAM mirror failed: wrote 0x%02X to 0x%04X, read 0x%02X from 0x%04X",
					tc.value, tc.wramAddr, echoValue, tc.echoAddr)
			}
		})

		t.Run("Echo->WRAM", func(t *testing.T) {
			// Write to Echo RAM
			newValue := tc.value + 1
			bus.Write(tc.echoAddr, newValue)

			// Read from WRAM - should be same value
			wramValue := bus.Read(tc.wramAddr)
			if wramValue != newValue {
				t.Errorf("Echo RAM mirror failed: wrote 0x%02X to 0x%04X, read 0x%02X from 0x%04X",
					newValue, tc.echoAddr, wramValue, tc.wramAddr)
			}
		})
	}
}


// setupTestROMHeader sets up a minimal valid ROM header for testing.
func setupTestROMHeader(rom []byte) {
	// Title
	copy(rom[0x0134:], []byte("TEST"))

	// Cartridge type (0x00 = ROM only)
	rom[0x0147] = 0x00

	// ROM size (0x00 = 32 KiB)
	rom[0x0148] = 0x00

	// RAM size (0x00 = No RAM)
	rom[0x0149] = 0x00

	// Calculate header checksum
	checksum := byte(0)
	for addr := 0x0134; addr <= 0x014C; addr++ {
		checksum = checksum - rom[addr] - 1
	}
	rom[0x014D] = checksum
}
