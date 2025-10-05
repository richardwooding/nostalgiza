package memory

import (
	"testing"
)

func TestNewBus(t *testing.T) {
	bus := NewBus()

	if bus == nil {
		t.Fatal("NewBus() returned nil")
	}

	if bus.currentROMBank != 1 {
		t.Errorf("currentROMBank = %d, want 1", bus.currentROMBank)
	}

	if bus.currentRAMBank != 0 {
		t.Errorf("currentRAMBank = %d, want 0", bus.currentRAMBank)
	}

	if bus.ramEnabled {
		t.Error("ramEnabled should be false initially")
	}
}

func TestROMAccess(t *testing.T) {
	bus := NewBus()

	// Test ROM Bank 00 (0000-3FFF)
	bus.rom0[0x0100] = 0x42
	value := bus.Read(0x0100)
	if value != 0x42 {
		t.Errorf("Read(0x0100) = %02X, want 0x42", value)
	}

	// Test ROM Bank 01 (4000-7FFF)
	bus.rom1[0x0000] = 0x84
	value = bus.Read(0x4000)
	if value != 0x84 {
		t.Errorf("Read(0x4000) = %02X, want 0x84", value)
	}

	// Writing to ROM should be ignored
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

	// External RAM should not be accessible when disabled
	bus.Write(0xA000, 0x42)
	value := bus.Read(0xA000)
	if value != 0xFF {
		t.Errorf("Read(0xA000) with RAM disabled = %02X, want 0xFF", value)
	}

	// Enable external RAM
	bus.ramEnabled = true

	// Now it should be accessible
	bus.Write(0xA000, 0x42)
	value = bus.Read(0xA000)
	if value != 0x42 {
		t.Errorf("Read(0xA000) with RAM enabled = %02X, want 0x42", value)
	}

	// Disable again
	bus.ramEnabled = false
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
	rom[0x0100] = 0x00 // NOP at entry point
	rom[0x0104] = 0xCE // Nintendo logo byte
	rom[0x4000] = 0x42 // First byte of ROM bank 1

	bus.LoadROM(rom)

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
