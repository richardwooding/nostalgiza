package apu

import (
	"testing"
)

func TestNoiseChannel_LFSR(t *testing.T) {
	n := NewNoiseChannel()

	// Verify initial LFSR state
	if n.lfsr != 0x7FFF {
		t.Errorf("Initial LFSR: got 0x%04X, want 0x7FFF", n.lfsr)
	}

	// Clock LFSR and verify it changes
	initialLFSR := n.lfsr
	n.clockLFSR()

	if n.lfsr == initialLFSR {
		t.Error("LFSR should change after clocking")
	}

	// Verify bit 14 is set based on XOR of bits 0 and 1
	xorResult := (initialLFSR & 0x01) ^ ((initialLFSR >> 1) & 0x01)
	expectedBit14 := (n.lfsr >> 14) & 0x01
	if expectedBit14 != xorResult {
		t.Errorf("Bit 14: got %d, want %d", expectedBit14, xorResult)
	}
}

func TestNoiseChannel_LFSRWidth(t *testing.T) {
	tests := []struct {
		name      string
		width7bit bool
	}{
		{"15-bit mode", false},
		{"7-bit mode", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewNoiseChannel()
			n.lfsrWidth = tt.width7bit
			n.lfsr = 0x7FFF

			// Clock multiple times and check behavior
			for i := 0; i < 100; i++ {
				n.clockLFSR()
			}

			if tt.width7bit {
				// In 7-bit mode, upper bits should match bit 6 pattern
				// This is a simplified check - just verify LFSR is still changing
				if n.lfsr == 0x7FFF {
					t.Error("LFSR should have changed in 7-bit mode")
				}
			}
		})
	}
}

func TestNoiseChannel_LengthTimer(t *testing.T) {
	n := NewNoiseChannel()

	// Set length to 1 (63 internal counter)
	n.WriteNR41(0x3F)
	n.WriteNR42(0xF0) // Max volume
	n.WriteNR44(0xC0) // Trigger with length enabled

	if !n.IsEnabled() {
		t.Fatal("Channel should be enabled after trigger")
	}

	// Clock length timer once
	n.ClockLength()

	if n.IsEnabled() {
		t.Error("Channel should be disabled after length expires")
	}
}

func TestNoiseChannel_VolumeEnvelope(t *testing.T) {
	tests := []struct {
		name           string
		initialVolume  uint8
		envelopeAdd    bool
		envelopePeriod uint8
		expectedVolume uint8
	}{
		{"Increase from 0", 0, true, 1, 1},
		{"Decrease from 15", 15, false, 1, 14},
		{"No change (period 0)", 8, true, 0, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewNoiseChannel()

			// Set envelope
			nr42 := tt.initialVolume << 4
			if tt.envelopeAdd {
				nr42 |= 0x08
			}
			nr42 |= tt.envelopePeriod
			n.WriteNR42(nr42)
			n.WriteNR44(0x80) // Trigger

			// Clock envelope once
			n.ClockEnvelope()

			if n.envelopeVolume != tt.expectedVolume {
				t.Errorf("Volume: got %d, want %d", n.envelopeVolume, tt.expectedVolume)
			}
		})
	}
}

func TestNoiseChannel_FrequencyControl(t *testing.T) {
	n := NewNoiseChannel()

	// Test divisor codes
	tests := []struct {
		clockShift  uint8
		divisorCode uint8
	}{
		{0, 0}, // divisor 8
		{1, 1}, // divisor 16, shift 1
		{4, 7}, // divisor 112, shift 4
	}

	for _, tt := range tests {
		n.WriteNR43((tt.clockShift << 4) | tt.divisorCode)

		if n.clockShift != tt.clockShift {
			t.Errorf("Clock shift: got %d, want %d", n.clockShift, tt.clockShift)
		}
		if n.divisorCode != tt.divisorCode {
			t.Errorf("Divisor code: got %d, want %d", n.divisorCode, tt.divisorCode)
		}
	}
}

func TestNoiseChannel_DACDisable(t *testing.T) {
	n := NewNoiseChannel()

	// Set volume to 0, no envelope (DAC off)
	n.WriteNR42(0x00)
	n.WriteNR44(0x80) // Trigger

	if n.IsEnabled() {
		t.Error("Channel should not enable when DAC is off")
	}

	if n.GetSample() != 0 {
		t.Error("Sample should be 0 when DAC is disabled")
	}
}

func TestNoiseChannel_Trigger(t *testing.T) {
	n := NewNoiseChannel()

	n.WriteNR42(0xF0) // Max volume, DAC enabled
	n.WriteNR44(0x80) // Trigger

	if !n.IsEnabled() {
		t.Error("Channel should be enabled after trigger")
	}

	if n.lfsr != 0x7FFF {
		t.Error("LFSR should be reset to 0x7FFF after trigger")
	}

	if n.phaseTimer != 0 {
		t.Error("Phase timer should be reset to 0 after trigger")
	}
}

func TestNoiseChannel_GetSample(t *testing.T) {
	n := NewNoiseChannel()

	// Enable and trigger
	n.WriteNR42(0xF0) // Max volume
	n.WriteNR44(0x80) // Trigger

	// Get sample - should be based on inverted bit 0 of LFSR
	sample := n.GetSample()

	// LFSR starts at 0x7FFF (bit 0 = 1), inverted = 0
	// Bipolar: 0 * 2.0 - 1.0 = -1.0
	if sample != -1.0 {
		t.Errorf("Initial sample: got %f, want -1.0", sample)
	}

	// Clock LFSR until we get bit 0 = 0 (inverted = 1)
	for i := 0; i < 1000; i++ {
		n.clockLFSR()
		if (n.lfsr & 0x01) == 0 {
			break
		}
	}

	sample = n.GetSample()
	expected := float32(15) / 15.0 // Max volume
	if sample != expected {
		t.Errorf("Sample with bit 0 = 0: got %f, want %f", sample, expected)
	}
}

func TestNoiseChannel_Update(t *testing.T) {
	n := NewNoiseChannel()

	// Enable and trigger
	n.WriteNR42(0xF0) // Max volume
	n.WriteNR43(0x00) // Divisor 8, shift 0
	n.WriteNR44(0x80) // Trigger

	initialLFSR := n.lfsr

	// Update with enough cycles to trigger LFSR clock
	n.Update(8)

	if n.lfsr == initialLFSR {
		t.Error("LFSR should have changed after update")
	}
}

func TestNoiseChannel_Reset(t *testing.T) {
	n := NewNoiseChannel()

	// Set some state
	n.WriteNR41(0xFF)
	n.WriteNR42(0xFF)
	n.WriteNR43(0xFF)
	n.WriteNR44(0xFF)

	// Reset
	n.Reset()

	// Check state is cleared
	if n.enabled {
		t.Error("Channel should be disabled after reset")
	}
	if n.dacEnabled {
		t.Error("DAC should be disabled after reset")
	}
	if n.lfsr != 0x7FFF {
		t.Errorf("LFSR should be 0x7FFF after reset, got 0x%04X", n.lfsr)
	}
	if n.lengthCounter != 0 {
		t.Error("Length counter should be 0 after reset")
	}
}

func TestNoiseChannel_RegisterReadback(t *testing.T) {
	n := NewNoiseChannel()

	// Test NR42 readback
	n.WriteNR42(0xF8)
	if got := n.ReadNR42(); got != 0xF8 {
		t.Errorf("NR42 readback: got 0x%02X, want 0xF8", got)
	}

	// Test NR43 readback
	n.WriteNR43(0xFF)
	if got := n.ReadNR43(); got != 0xFF {
		t.Errorf("NR43 readback: got 0x%02X, want 0xFF", got)
	}

	// Test NR41 is write-only
	n.WriteNR41(0x3F)
	if got := n.ReadNR41(); got != 0xFF {
		t.Errorf("NR41 should return 0xFF (write-only), got 0x%02X", got)
	}
}
