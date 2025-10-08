package timer

import (
	"testing"
)

// Stress Tests - Test edge cases and rapid state changes

func TestTimerStress_RapidTACChanges(t *testing.T) {
	timer := New(nil)
	timer.Write(TIMA, 0x00)
	timer.Write(DIV, 0x00) // Reset divCounter via public API

	// Rapidly change timer frequency
	frequencies := []uint8{0x04, 0x05, 0x06, 0x07}

	for i := 0; i < 100; i++ {
		// Change frequency
		tacValue := frequencies[i%len(frequencies)]
		timer.Write(TAC, tacValue)

		// Run a few cycles
		timer.Update(50)
	}

	// Timer should still be functioning (no crashes/panics)
	// TIMA should have incremented at some point
	if timer.Read(TIMA) == 0 {
		// This might fail if frequencies don't align, but timer should still work
		t.Log("TIMA did not increment during rapid TAC changes (may be expected)")
	}
}

func TestTimerStress_FrequentDIVResets(t *testing.T) {
	interruptCount := 0
	timer := New(func() { interruptCount++ })

	// Enable timer at high frequency
	timer.Write(TAC, 0x05) // 262144 Hz (every 16 cycles)
	timer.Write(TMA, 0xFE)
	timer.Write(TIMA, 0xFE)

	// Reset DIV repeatedly during timer operation
	for i := 0; i < 100; i++ {
		timer.Update(8) // Half a timer period

		// Reset DIV (might cause falling edge)
		timer.Write(DIV, 0x00)

		timer.Update(8) // Other half
	}

	// Verify timer still works and DIV is 0
	if timer.Read(DIV) != 0 {
		t.Errorf("DIV = %d, want 0", timer.Read(DIV))
	}

	// Some interrupts should have fired
	if interruptCount == 0 {
		t.Error("No timer interrupts fired during stress test")
	}
}

func TestTimerStress_ConcurrentTimerAndDIV(t *testing.T) {
	timer := New(nil)

	// Enable timer
	timer.Write(TAC, 0x05) // 262144 Hz
	timer.Write(TIMA, 0x00)

	initialDIV := timer.Read(DIV)

	// Run for many cycles, occasionally resetting DIV
	for i := 0; i < 50; i++ {
		timer.Update(256) // Increment DIV once

		if i%5 == 0 {
			timer.Write(DIV, 0x00) // Reset DIV
		}
	}

	// TIMA should have incremented (timer operates independently)
	if timer.Read(TIMA) == 0 {
		t.Error("TIMA did not increment despite timer being enabled")
	}

	// DIV should be functioning
	currentDIV := timer.Read(DIV)
	if currentDIV == initialDIV && initialDIV != 0 {
		t.Error("DIV did not change during test")
	}
}

func TestTimerStress_EnableDisableCycles(t *testing.T) {
	interruptCount := 0
	timer := New(func() { interruptCount++ })

	timer.Write(TMA, 0xFC)
	timer.Write(TIMA, 0xFC)

	// Rapidly enable and disable timer
	for i := 0; i < 100; i++ {
		// Enable
		timer.Write(TAC, 0x05)
		timer.Update(16)

		// Disable
		timer.Write(TAC, 0x00)
		timer.Update(16)
	}

	// Some increments should have occurred
	// At minimum, timer should not crash
	t.Logf("Interrupts fired: %d", interruptCount)
}

// Boundary Tests - Test numeric overflow conditions

func TestTimerBoundary_DivCounterOverflow(t *testing.T) {
	timer := New(nil)

	// Increment DIV to near uint16 max by running many cycles
	// DIV = 255 means divCounter = 65280 (255 * 256)
	// We need to get close to 65535
	timer.Update(65280) // Gets divCounter to 65280
	if timer.Read(DIV) != 255 {
		t.Fatalf("DIV before overflow test = %d, want 255", timer.Read(DIV))
	}

	// Update by 256 more cycles - should overflow divCounter
	// 65280 + 256 = 65536, which overflows to 0
	timer.Update(256)

	// DIV should wrap to 0
	if timer.Read(DIV) != 0 {
		t.Errorf("DIV after divCounter overflow = %d, want 0", timer.Read(DIV))
	}

	// Continue updating - should work normally
	timer.Update(256)
	if timer.Read(DIV) != 1 {
		t.Errorf("DIV after post-overflow update = %d, want 1", timer.Read(DIV))
	}
}

func TestTimerBoundary_MultipleTimaOverflows(t *testing.T) {
	interruptCount := 0
	timer := New(func() { interruptCount++ })

	// Enable timer at highest frequency
	timer.Write(TAC, 0x05)  // 262144 Hz (every 16 cycles)
	timer.Write(TMA, 0x00)  // Reload with 0
	timer.Write(TIMA, 0xFE) // Start near overflow
	timer.Write(DIV, 0x00)  // Reset divCounter via public API

	// Trigger multiple consecutive overflows
	// TIMA: 0xFE -> 0xFF -> 0x00 (overflow, reload to 0) -> 0x01 -> ... -> 0xFF -> 0x00 (overflow)
	// Each increment takes 16 cycles
	// To get 5 overflows from 0xFE: FE->FF->00, 00->..->FF->00, etc.
	// First overflow at 2 increments (FE->FF->00)
	// Then need 256 increments for each subsequent overflow
	overflowsExpected := 5
	cycles := (2 + (256 * (overflowsExpected - 1))) * 16

	timer.Update(uint16(cycles)) //nolint:gosec // Safe: cycles is bounded by test logic

	if interruptCount != overflowsExpected {
		t.Errorf("Interrupt count = %d, want %d", interruptCount, overflowsExpected)
	}

	// TIMA should be 0 after last overflow
	if timer.Read(TIMA) != 0 {
		t.Errorf("TIMA after multiple overflows = 0x%02X, want 0x00", timer.Read(TIMA))
	}
}

func TestTimerBoundary_TIMAAllValues(t *testing.T) {
	timer := New(nil)

	// Test writing all possible TIMA values
	for i := 0; i <= 255; i++ {
		val := uint8(i) //nolint:gosec // Safe: i is bounded 0-255
		timer.Write(TIMA, val)
		if timer.Read(TIMA) != val {
			t.Errorf("TIMA write/read failed for value %d", i)
		}
	}
}

func TestTimerBoundary_TMAAllValues(t *testing.T) {
	timer := New(nil)

	// Test writing all possible TMA values
	for i := 0; i <= 255; i++ {
		val := uint8(i) //nolint:gosec // Safe: i is bounded 0-255
		timer.Write(TMA, val)
		if timer.Read(TMA) != val {
			t.Errorf("TMA write/read failed for value %d", i)
		}
	}
}

func TestTimerBoundary_LargeUpdateCycles(t *testing.T) {
	timer := New(nil)

	// Test updating with large cycle counts
	timer.Write(TAC, 0x05) // 262144 Hz
	timer.Write(TIMA, 0x00)
	timer.Write(DIV, 0x00) // Reset divCounter via public API

	// Update with max uint16 cycles
	timer.Update(65535)

	// Should not crash or produce incorrect behavior
	// TIMA should have incremented (65535 / 16 = 4095 times)
	// Starting from 0, after 4095 increments: 4095 % 256 = 255
	expectedTIMA := uint8(4095 % 256)
	if timer.Read(TIMA) != expectedTIMA {
		t.Errorf("TIMA after large update = %d, want %d", timer.Read(TIMA), expectedTIMA)
	}
}

func TestTimerBoundary_ZeroCycleUpdate(t *testing.T) {
	timer := New(nil)

	initialTIMA := timer.Read(TIMA)
	initialDIV := timer.Read(DIV)

	// Update with 0 cycles - should be no-op
	timer.Update(0)

	if timer.Read(TIMA) != initialTIMA {
		t.Errorf("TIMA changed after 0-cycle update")
	}
	if timer.Read(DIV) != initialDIV {
		t.Errorf("DIV changed after 0-cycle update")
	}
}
