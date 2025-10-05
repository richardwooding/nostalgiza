package apu

import (
	"testing"
)

func TestWaveChannel_DACEnable(t *testing.T) {
	w := NewWaveChannel()

	// DAC disabled by default
	if w.dacEnabled {
		t.Error("DAC should be disabled by default")
	}

	// Enable DAC
	w.WriteNR30(0x80)
	if !w.dacEnabled {
		t.Error("DAC should be enabled after writing 0x80 to NR30")
	}

	// Disable DAC
	w.WriteNR30(0x00)
	if w.dacEnabled {
		t.Error("DAC should be disabled after writing 0x00 to NR30")
	}
}

func TestWaveChannel_LengthTimer(t *testing.T) {
	w := NewWaveChannel()

	// Set length to 1 (255 internal counter)
	w.WriteNR31(0xFF)
	w.WriteNR30(0x80) // Enable DAC
	w.WriteNR34(0xC0) // Trigger with length enabled

	if !w.IsEnabled() {
		t.Fatal("Channel should be enabled after trigger")
	}

	// Clock length timer once
	w.ClockLength()

	if w.IsEnabled() {
		t.Error("Channel should be disabled after length expires")
	}
}

func TestWaveChannel_OutputLevel(t *testing.T) {
	w := NewWaveChannel()

	// Write a known pattern to wave RAM (all 0xF)
	for i := uint16(0); i < 16; i++ {
		w.WriteWaveRAM(i, 0xFF)
	}

	tests := []struct {
		name        string
		outputLevel uint8
		expected    float32
	}{
		{"Mute (0%)", 0, -1.0},      // Sample 0xF >> inf = 0, bipolar: 0/7.5 - 1.0 = -1.0
		{"100%", 1, 1.0},            // Sample 0xF (no shift) = 15, bipolar: 15/7.5 - 1.0 = 1.0
		{"50%", 2, -0.066667},       // Sample 0xF >> 1 = 7, bipolar: 7/7.5 - 1.0 = -0.066667
		{"25%", 3, -0.6},            // Sample 0xF >> 2 = 3, bipolar: 3/7.5 - 1.0 = -0.6
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w.WriteNR30(0x80)                // Enable DAC
			w.WriteNR32(tt.outputLevel << 5) // Set output level
			w.WriteNR34(0x80)                // Trigger

			sample := w.GetSample()

			// Allow small floating point error
			if abs(sample-tt.expected) > 0.01 {
				t.Errorf("Output level %d: got %f, want %f", tt.outputLevel, sample, tt.expected)
			}
		})
	}
}

func TestWaveChannel_WaveRAM(t *testing.T) {
	w := NewWaveChannel()

	// Write pattern to wave RAM
	for i := uint16(0); i < 16; i++ {
		w.WriteWaveRAM(i, uint8(i))
	}

	// Read back and verify
	for i := uint16(0); i < 16; i++ {
		got := w.ReadWaveRAM(i)
		want := uint8(i)
		if got != want {
			t.Errorf("WaveRAM[%d]: got 0x%02X, want 0x%02X", i, got, want)
		}
	}
}

func TestWaveChannel_FrequencyChange(t *testing.T) {
	w := NewWaveChannel()

	// Set frequency
	w.WriteNR33(0xFF) // Low byte
	w.WriteNR34(0x07) // High 3 bits

	expectedFreq := uint16(0x7FF)
	if w.frequency != expectedFreq {
		t.Errorf("Frequency: got %d, want %d", w.frequency, expectedFreq)
	}
}

func TestWaveChannel_Trigger(t *testing.T) {
	w := NewWaveChannel()

	w.WriteNR30(0x80) // Enable DAC
	w.WriteNR34(0x80) // Trigger

	if !w.IsEnabled() {
		t.Error("Channel should be enabled after trigger")
	}

	if w.wavePos != 0 {
		t.Error("Wave position should be reset to 0 after trigger")
	}

	if w.phaseTimer != 0 {
		t.Error("Phase timer should be reset to 0 after trigger")
	}
}

func TestWaveChannel_DACDisableClearsEnabled(t *testing.T) {
	w := NewWaveChannel()

	// Enable and trigger
	w.WriteNR30(0x80)
	w.WriteNR34(0x80)

	if !w.IsEnabled() {
		t.Fatal("Channel should be enabled")
	}

	// Disable DAC
	w.WriteNR30(0x00)

	if w.IsEnabled() {
		t.Error("Channel should be disabled when DAC is turned off")
	}
}

func TestWaveChannel_SampleProgression(t *testing.T) {
	w := NewWaveChannel()

	// Write simple pattern: first byte 0x01, rest 0x00
	w.WriteWaveRAM(0, 0x01)
	for i := uint16(1); i < 16; i++ {
		w.WriteWaveRAM(i, 0x00)
	}

	w.WriteNR30(0x80) // Enable DAC
	w.WriteNR32(0x20) // 100% output level
	w.WriteNR33(0x00) // Frequency = 0
	w.WriteNR34(0x80) // Trigger

	// First sample should be from high nibble of first byte (0x0)
	// Bipolar: 0/7.5 - 1.0 = -1.0
	sample1 := w.GetSample()
	if sample1 != -1.0 {
		t.Errorf("First sample: got %f, want -1.0", sample1)
	}

	// Advance wave position
	w.wavePos = 1
	// Second sample should be from low nibble of first byte (0x1)
	// Bipolar: 1/7.5 - 1.0 = -0.866667
	sample2 := w.GetSample()
	expected := float32(1)/7.5 - 1.0
	if abs(sample2-expected) > 0.01 {
		t.Errorf("Second sample: got %f, want %f", sample2, expected)
	}
}

func TestWaveChannel_Reset(t *testing.T) {
	w := NewWaveChannel()

	// Set some state
	w.WriteNR30(0xFF)
	w.WriteNR31(0xFF)
	w.WriteNR32(0xFF)
	w.WriteNR33(0xFF)
	w.WriteNR34(0xFF)
	for i := uint16(0); i < 16; i++ {
		w.WriteWaveRAM(i, 0xFF)
	}

	// Reset
	w.Reset()

	// Check state is cleared
	if w.enabled {
		t.Error("Channel should be disabled after reset")
	}
	if w.dacEnabled {
		t.Error("DAC should be disabled after reset")
	}
	if w.frequency != 0 {
		t.Error("Frequency should be 0 after reset")
	}
	for i := uint16(0); i < 16; i++ {
		if w.waveRAM[i] != 0 {
			t.Errorf("WaveRAM[%d] should be 0 after reset, got 0x%02X", i, w.waveRAM[i])
		}
	}
}

// Helper function for floating point comparison.
func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
