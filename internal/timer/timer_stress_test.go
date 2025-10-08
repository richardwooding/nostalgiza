package timer

import (
	"testing"
)

// This file contains stress tests and boundary tests for the timer implementation.
//
// Stress tests verify the timer's robustness under rapid state changes and
// concurrent operations, ensuring it handles edge cases that may occur in
// real Game Boy software.
//
// Boundary tests verify correct behavior at numeric limits and overflow
// conditions, ensuring the timer handles wrapping and edge values correctly.

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

	// Test passes if no crash/panic occurs during rapid TAC changes
	// Minimal assertions to verify timer remains operational
	if timer.Read(TIMA) == 0 {
		t.Error("TIMA should have incremented during rapid TAC changes")
	}
	// DIV should have advanced (100 iterations * 50 cycles = 5000 cycles = 19 DIV increments)
	if timer.Read(DIV) < 19 {
		t.Errorf("DIV = %d, expected at least 19 after 5000 cycles", timer.Read(DIV))
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

	// Verify interrupts fired
	// With 100 iterations of (8 cycles + DIV reset + 8 cycles):
	// Each iteration advances timer by 16 cycles at 262144 Hz (falling edge every 16 cycles)
	// Expected: at least 1 increment per iteration, with TIMA starting at 0xFE
	// After 2 increments: 0xFE->0xFF->0x00 (overflow), then continues from 0xFE (TMA value)
	// Minimum interrupts: 100/4 = 25 (conservative estimate accounting for resets)
	if interruptCount == 0 {
		t.Error("No timer interrupts fired during stress test")
	} else if interruptCount < 20 {
		t.Errorf("Only %d interrupts fired, expected at least 20", interruptCount)
	}
}

func TestTimerStress_ConcurrentTimerAndDIV(t *testing.T) {
	timer := New(nil)

	// Enable timer
	timer.Write(TAC, 0x05) // 262144 Hz
	timer.Write(TIMA, 0x00)

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
	// After 50 iterations of 256 cycles with resets every 5 iterations (10 total):
	// Each reset happens after iteration multiples of 5 (0, 5, 10, 15, 20, 25, 30, 35, 40, 45)
	// Between resets: 5 iterations * 256 cycles = 1,280 cycles = 5 DIV increments
	// After last reset (iteration 45), we have 4 more iterations = 1,024 cycles = 4 DIV increments
	// So DIV should be exactly 4 at the end
	currentDIV := timer.Read(DIV)
	if currentDIV != 4 {
		t.Errorf("DIV = %d, expected 4 after test sequence", currentDIV)
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
	// Note: cycles = (2 + 256*4) * 16 = 16,416 which fits in uint16 (max 65,535)
	// If overflowsExpected is increased beyond 6, cycles may exceed uint16 max
	cycles := (2 + (256 * (overflowsExpected - 1))) * 16

	timer.Update(uint16(cycles)) //nolint:gosec // Safe: cycles=16,416 for overflowsExpected=5

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
	// TIMA should have incremented 4095 times (65535 / 16)
	// This causes 15 overflows (reloading from TMA=0) plus 255 additional increments
	// Result: (4095 % 256) = 255
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
