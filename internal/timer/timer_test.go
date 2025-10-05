package timer

import (
	"testing"
)

func TestNew(t *testing.T) {
	interruptCalled := false
	callback := func() { interruptCalled = true }

	timer := New(callback)

	if timer == nil {
		t.Fatal("New() returned nil")
	}

	if timer.requestInterrupt == nil {
		t.Error("requestInterrupt callback not set")
	}

	// Verify callback works
	timer.requestInterrupt()
	if !interruptCalled {
		t.Error("Interrupt callback was not called")
	}
}

func TestDIVIncrement(t *testing.T) {
	timer := New(nil)

	// DIV increments every 256 CPU cycles (64 M-cycles)
	// DIV is the upper 8 bits of the 16-bit divCounter

	initialDIV := timer.Read(DIV)
	if initialDIV != 0 {
		t.Errorf("Initial DIV = %d, want 0", initialDIV)
	}

	// Update by 255 cycles - should not increment DIV yet
	timer.Update(255)
	if timer.Read(DIV) != 0 {
		t.Errorf("DIV after 255 cycles = %d, want 0", timer.Read(DIV))
	}

	// One more cycle should increment DIV
	timer.Update(1)
	if timer.Read(DIV) != 1 {
		t.Errorf("DIV after 256 cycles = %d, want 1", timer.Read(DIV))
	}

	// Another 256 cycles
	timer.Update(256)
	if timer.Read(DIV) != 2 {
		t.Errorf("DIV after 512 cycles = %d, want 2", timer.Read(DIV))
	}
}

func TestDIVReset(t *testing.T) {
	timer := New(nil)

	// Increment DIV
	timer.Update(256)
	if timer.Read(DIV) != 1 {
		t.Fatalf("DIV = %d, want 1", timer.Read(DIV))
	}

	// Writing any value to DIV resets it to 0
	timer.Write(DIV, 0xFF)
	if timer.Read(DIV) != 0 {
		t.Errorf("DIV after write = %d, want 0", timer.Read(DIV))
	}

	// Try different value
	timer.Update(512)
	if timer.Read(DIV) != 2 {
		t.Fatalf("DIV = %d, want 2", timer.Read(DIV))
	}

	timer.Write(DIV, 0x00)
	if timer.Read(DIV) != 0 {
		t.Errorf("DIV after write = %d, want 0", timer.Read(DIV))
	}
}

func TestTACReadWrite(t *testing.T) {
	timer := New(nil)

	// Upper bits should read as 1
	timer.Write(TAC, 0x00)
	if timer.Read(TAC) != 0xF8 {
		t.Errorf("TAC with value 0x00 = 0x%02X, want 0xF8", timer.Read(TAC))
	}

	// Only lower 3 bits are writable
	timer.Write(TAC, 0xFF)
	if timer.Read(TAC) != 0xFF {
		t.Errorf("TAC with value 0xFF = 0x%02X, want 0xFF", timer.Read(TAC))
	}

	timer.Write(TAC, 0x07)
	if timer.Read(TAC) != 0xFF {
		t.Errorf("TAC with value 0x07 = 0x%02X, want 0xFF", timer.Read(TAC))
	}

	timer.Write(TAC, 0x05)
	tacValue := timer.Read(TAC)
	if tacValue&0x07 != 0x05 {
		t.Errorf("TAC lower bits = 0x%02X, want 0x05", tacValue&0x07)
	}
}

func TestTimerDisabled(t *testing.T) {
	interruptCalled := false
	timer := New(func() { interruptCalled = true })

	// Timer disabled (TAC bit 2 = 0)
	timer.Write(TAC, 0x00)
	timer.Write(TMA, 0x00)
	timer.Write(TIMA, 0xFF)

	// Run many cycles - TIMA should not increment
	timer.Update(10000)

	if timer.Read(TIMA) != 0xFF {
		t.Errorf("TIMA with timer disabled = %d, want 255", timer.Read(TIMA))
	}

	if interruptCalled {
		t.Error("Interrupt called when timer disabled")
	}
}

func TestTimerEnabled_4096Hz(t *testing.T) {
	interruptCalled := false
	timer := New(func() { interruptCalled = true })

	// Enable timer at 4096 Hz (TAC = 0x04 | 0x00 = 0x04)
	timer.Write(TAC, 0x04)
	timer.Write(TIMA, 0x00)

	// 4096 Hz = every 1024 CPU cycles
	// Bit 9 of divCounter determines increment

	// Update by 1023 cycles - should not increment yet
	timer.Update(255)
	timer.Update(255)
	timer.Update(255)
	timer.Update(255) // 1020 cycles
	if timer.Read(TIMA) != 0 {
		t.Errorf("TIMA after 1020 cycles = %d, want 0", timer.Read(TIMA))
	}

	// Bit 9 flips at 512, 1024, 1536, etc.
	// We need a falling edge, so go from 511 to 512 (bit 9: 0->1)
	// then from 1023 to 1024 (bit 9: 1->0) - this is the falling edge
	timer.divCounter = 0
	timer.Update(1024) // Should cause falling edge on bit 9

	if timer.Read(TIMA) != 1 {
		t.Errorf("TIMA after 1024 cycles = %d, want 1", timer.Read(TIMA))
	}

	if interruptCalled {
		t.Error("Interrupt called without overflow")
	}
}

func TestTimerEnabled_262144Hz(t *testing.T) {
	timer := New(nil)

	// Enable timer at 262144 Hz (TAC = 0x04 | 0x01 = 0x05)
	timer.Write(TAC, 0x05)
	timer.Write(TIMA, 0x00)

	// 262144 Hz = every 16 CPU cycles (bit 3)
	// Falling edge on bit 3

	timer.divCounter = 0
	timer.Update(16) // Should increment TIMA

	if timer.Read(TIMA) != 1 {
		t.Errorf("TIMA after 16 cycles = %d, want 1", timer.Read(TIMA))
	}
}

func TestTimerEnabled_65536Hz(t *testing.T) {
	timer := New(nil)

	// Enable timer at 65536 Hz (TAC = 0x04 | 0x02 = 0x06)
	timer.Write(TAC, 0x06)
	timer.Write(TIMA, 0x00)

	// 65536 Hz = every 64 CPU cycles (bit 5)

	timer.divCounter = 0
	timer.Update(64) // Should increment TIMA

	if timer.Read(TIMA) != 1 {
		t.Errorf("TIMA after 64 cycles = %d, want 1", timer.Read(TIMA))
	}
}

func TestTimerEnabled_16384Hz(t *testing.T) {
	timer := New(nil)

	// Enable timer at 16384 Hz (TAC = 0x04 | 0x03 = 0x07)
	timer.Write(TAC, 0x07)
	timer.Write(TIMA, 0x00)

	// 16384 Hz = every 256 CPU cycles (bit 7)

	timer.divCounter = 0
	timer.Update(256) // Should increment TIMA

	if timer.Read(TIMA) != 1 {
		t.Errorf("TIMA after 256 cycles = %d, want 1", timer.Read(TIMA))
	}
}

func TestTimerOverflow(t *testing.T) {
	interruptCalled := false
	timer := New(func() { interruptCalled = true })

	// Enable timer, set TIMA to 0xFF, TMA to 0x42
	timer.Write(TAC, 0x05) // 262144 Hz
	timer.Write(TMA, 0x42)
	timer.Write(TIMA, 0xFF)

	// Update to trigger increment (overflow)
	timer.divCounter = 0
	timer.Update(16)

	// TIMA should wrap to 0 then reload with TMA
	if timer.Read(TIMA) != 0x42 {
		t.Errorf("TIMA after overflow = 0x%02X, want 0x42", timer.Read(TIMA))
	}

	if !interruptCalled {
		t.Error("Timer interrupt not called on overflow")
	}
}

func TestTimerMultipleOverflows(t *testing.T) {
	interruptCount := 0
	timer := New(func() { interruptCount++ })

	// Enable timer, set TMA to 0xFC (will overflow after 4 increments)
	timer.Write(TAC, 0x05) // 262144 Hz (every 16 cycles)
	timer.Write(TMA, 0xFC)
	timer.Write(TIMA, 0xFC)

	// Trigger multiple increments
	timer.divCounter = 0
	for i := 0; i < 20; i++ {
		timer.Update(16) // Each should increment TIMA
	}

	// TIMA starts at 0xFC, increments to 0xFD, 0xFE, 0xFF, then overflows to 0xFC
	// 20 increments = 5 complete cycles (FC->FD->FE->FF->FC) = 5 overflows
	if interruptCount != 5 {
		t.Errorf("Interrupt count = %d, want 5", interruptCount)
	}
}

func TestDIVWriteFallingEdge(t *testing.T) {
	timer := New(nil)

	// Enable timer at 262144 Hz (bit 3)
	timer.Write(TAC, 0x05)
	timer.Write(TIMA, 0x00)

	// Set divCounter so bit 3 is set
	timer.divCounter = 0x0008 // Bit 3 = 1

	// Writing to DIV resets counter, causing falling edge on bit 3
	timer.Write(DIV, 0x00)

	// TIMA should have incremented due to falling edge
	if timer.Read(TIMA) != 1 {
		t.Errorf("TIMA after DIV reset = %d, want 1 (falling edge increment)", timer.Read(TIMA))
	}

	if timer.Read(DIV) != 0 {
		t.Errorf("DIV after reset = %d, want 0", timer.Read(DIV))
	}
}

func TestTACChangeFallingEdge(t *testing.T) {
	timer := New(nil)

	// Set divCounter so different bits are set
	timer.divCounter = 0x0200 // Bit 9 = 1

	// Enable timer at 4096 Hz (bit 9)
	timer.Write(TAC, 0x04)
	timer.Write(TIMA, 0x00)

	// Change to disabled - should cause falling edge
	timer.Write(TAC, 0x00)

	if timer.Read(TIMA) != 1 {
		t.Errorf("TIMA after TAC disable = %d, want 1 (falling edge)", timer.Read(TIMA))
	}
}

func TestTIMAReadWrite(t *testing.T) {
	timer := New(nil)

	timer.Write(TIMA, 0x42)
	if timer.Read(TIMA) != 0x42 {
		t.Errorf("TIMA = 0x%02X, want 0x42", timer.Read(TIMA))
	}

	timer.Write(TIMA, 0xFF)
	if timer.Read(TIMA) != 0xFF {
		t.Errorf("TIMA = 0x%02X, want 0xFF", timer.Read(TIMA))
	}
}

func TestTMAReadWrite(t *testing.T) {
	timer := New(nil)

	timer.Write(TMA, 0x42)
	if timer.Read(TMA) != 0x42 {
		t.Errorf("TMA = 0x%02X, want 0x42", timer.Read(TMA))
	}

	timer.Write(TMA, 0xFF)
	if timer.Read(TMA) != 0xFF {
		t.Errorf("TMA = 0x%02X, want 0xFF", timer.Read(TMA))
	}
}

func TestReset(t *testing.T) {
	timer := New(nil)

	// Set all registers to non-zero values
	timer.divCounter = 0x1234
	timer.Write(TIMA, 0x42)
	timer.Write(TMA, 0x84)
	timer.Write(TAC, 0x07)

	// Reset
	timer.Reset()

	if timer.Read(DIV) != 0 {
		t.Errorf("DIV after reset = %d, want 0", timer.Read(DIV))
	}
	if timer.Read(TIMA) != 0 {
		t.Errorf("TIMA after reset = %d, want 0", timer.Read(TIMA))
	}
	if timer.Read(TMA) != 0 {
		t.Errorf("TMA after reset = %d, want 0", timer.Read(TMA))
	}
	if timer.Read(TAC)&0x07 != 0 {
		t.Errorf("TAC after reset = 0x%02X, want 0xF8", timer.Read(TAC))
	}
	if timer.enabled {
		t.Error("Timer should be disabled after reset")
	}
	if timer.clockSelect != 0 {
		t.Errorf("clockSelect after reset = %d, want 0", timer.clockSelect)
	}
}

func TestInvalidAddress(t *testing.T) {
	timer := New(nil)

	// Reading invalid address should return 0xFF
	if timer.Read(0xFF00) != 0xFF {
		t.Errorf("Invalid read = 0x%02X, want 0xFF", timer.Read(0xFF00))
	}

	// Writing to invalid address should not crash
	timer.Write(0xFF00, 0x42)
}
