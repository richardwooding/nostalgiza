package apu

import (
	"testing"
)

func TestPulseChannel_DutyPatterns(t *testing.T) {
	tests := []struct {
		name        string
		dutyPattern uint8
		expectedOns int // Number of '1's in 8-step cycle
	}{
		{"12.5% duty", 0, 1},
		{"25% duty", 1, 2},
		{"50% duty", 2, 4},
		{"75% duty", 3, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPulseChannel(false)
			p.WriteNR21(tt.dutyPattern << 6)
			p.WriteNR22(0xF0) // Max volume, no envelope
			p.WriteNR24(0x80) // Trigger

			ones := 0
			for i := 0; i < 8; i++ {
				sample := p.GetSample()
				if sample > 0 {
					ones++
				}
				// Advance to next step in duty cycle
				p.phaseTimer = 2048 * 2
				p.Update(2048 * 2)
			}

			if ones != tt.expectedOns {
				t.Errorf("Duty pattern %d: got %d ones, want %d", tt.dutyPattern, ones, tt.expectedOns)
			}
		})
	}
}

func TestPulseChannel_LengthTimer(t *testing.T) {
	p := NewPulseChannel(false)

	// Set length to 1 (63 internal counter)
	p.WriteNR21(0x3F) // Length = 1
	p.WriteNR22(0xF0) // Max volume
	p.WriteNR24(0xC0) // Trigger with length enabled

	if !p.IsEnabled() {
		t.Fatal("Channel should be enabled after trigger")
	}

	// Clock length timer once
	p.ClockLength()

	if p.IsEnabled() {
		t.Error("Channel should be disabled after length expires")
	}
}

func TestPulseChannel_VolumeEnvelope(t *testing.T) {
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
			p := NewPulseChannel(false)

			// Set envelope
			nr22 := tt.initialVolume << 4
			if tt.envelopeAdd {
				nr22 |= 0x08
			}
			nr22 |= tt.envelopePeriod
			p.WriteNR22(nr22)
			p.WriteNR24(0x80) // Trigger

			// Clock envelope once
			p.ClockEnvelope()

			if p.envelopeVolume != tt.expectedVolume {
				t.Errorf("Volume: got %d, want %d", p.envelopeVolume, tt.expectedVolume)
			}
		})
	}
}

func TestPulseChannel_Sweep(t *testing.T) {
	p := NewPulseChannel(true) // Channel 1 with sweep

	// Set sweep: period=1, shift=1, increase
	p.WriteNR10(0x11) // 001 0 001 = period 1, increase, shift 1

	// Set frequency to 100 (small value to avoid overflow)
	p.WriteNR13(100)

	p.WriteNR12(0xF0) // Max volume

	// Trigger
	p.WriteNR14(0x80)

	// Verify sweep enabled
	if !p.sweepEnabled {
		t.Fatal("Sweep should be enabled after trigger with period=1, shift=1")
	}

	// Verify sweep shadow register was set
	if p.sweepShadow != 100 {
		t.Errorf("Sweep shadow should be 100, got %d", p.sweepShadow)
	}
}

func TestPulseChannel_DACDisable(t *testing.T) {
	p := NewPulseChannel(false)

	// Set volume to 0, no envelope (DAC off)
	p.WriteNR22(0x00)
	p.WriteNR24(0x80) // Trigger

	if p.IsEnabled() {
		t.Error("Channel should not enable when DAC is off")
	}

	if p.GetSample() != 0 {
		t.Error("Sample should be 0 when DAC is disabled")
	}
}

func TestPulseChannel_FrequencyChange(t *testing.T) {
	p := NewPulseChannel(false)

	// Set frequency
	p.WriteNR13(0xFF) // Low byte
	p.WriteNR14(0x07) // High 3 bits

	expectedFreq := uint16(0x7FF)
	if p.frequency != expectedFreq {
		t.Errorf("Frequency: got %d, want %d", p.frequency, expectedFreq)
	}
}

func TestPulseChannel_Reset(t *testing.T) {
	p := NewPulseChannel(true)

	// Set some state
	p.WriteNR10(0x7F)
	p.WriteNR11(0xFF)
	p.WriteNR12(0xFF)
	p.WriteNR13(0xFF)
	p.WriteNR14(0xFF)

	// Reset
	p.Reset()

	// Check all state is cleared
	if p.enabled {
		t.Error("Channel should be disabled after reset")
	}
	if p.dacEnabled {
		t.Error("DAC should be disabled after reset")
	}
	if p.frequency != 0 {
		t.Error("Frequency should be 0 after reset")
	}
	if p.lengthCounter != 0 {
		t.Error("Length counter should be 0 after reset")
	}
}

func TestPulseChannel_RegisterReadback(t *testing.T) {
	p := NewPulseChannel(true)

	// Test NR10 readback (only available on channel 1)
	// Bit 7 is unused and reads as 1
	p.WriteNR10(0x7F)
	if got := p.ReadNR10(); got != 0xFF {
		t.Errorf("NR10 readback: got 0x%02X, want 0xFF (bit 7 unused)", got)
	}

	// Test NR12 readback
	p.WriteNR12(0xF8)
	if got := p.ReadNR12(); got != 0xF8 {
		t.Errorf("NR12 readback: got 0x%02X, want 0xF8", got)
	}

	// Test NR11 is write-only (returns 0xFF with duty bits)
	p.WriteNR11(0xC0)
	if got := p.ReadNR11(); (got & 0xC0) != 0xC0 {
		t.Errorf("NR11 duty bits: got 0x%02X", got)
	}
}
