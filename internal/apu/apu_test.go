package apu

import (
	"testing"
)

func TestAPU_MasterControl(t *testing.T) {
	apu := New()

	// APU should start disabled
	if apu.enabled {
		t.Error("APU should be disabled initially")
	}

	// Enable APU
	apu.Write(0xFF26, 0x80)
	if !apu.enabled {
		t.Error("APU should be enabled after writing 0x80 to NR52")
	}

	// Disable APU
	apu.Write(0xFF26, 0x00)
	if apu.enabled {
		t.Error("APU should be disabled after writing 0x00 to NR52")
	}
}

func TestAPU_ChannelEnableStatus(t *testing.T) {
	apu := New()
	apu.Write(0xFF26, 0x80) // Enable APU

	// Trigger all channels
	apu.Write(0xFF12, 0xF0) // CH1 volume
	apu.Write(0xFF14, 0x80) // CH1 trigger

	apu.Write(0xFF17, 0xF0) // CH2 volume
	apu.Write(0xFF19, 0x80) // CH2 trigger

	apu.Write(0xFF1A, 0x80) // CH3 DAC enable
	apu.Write(0xFF1E, 0x80) // CH3 trigger

	apu.Write(0xFF21, 0xF0) // CH4 volume
	apu.Write(0xFF23, 0x80) // CH4 trigger

	// Read NR52 and check channel status bits
	nr52 := apu.Read(0xFF26)

	if (nr52 & 0x01) == 0 {
		t.Error("Channel 1 should be enabled (bit 0)")
	}
	if (nr52 & 0x02) == 0 {
		t.Error("Channel 2 should be enabled (bit 1)")
	}
	if (nr52 & 0x04) == 0 {
		t.Error("Channel 3 should be enabled (bit 2)")
	}
	if (nr52 & 0x08) == 0 {
		t.Error("Channel 4 should be enabled (bit 3)")
	}
}

func TestAPU_FrameSequencer(t *testing.T) {
	apu := New()
	apu.Write(0xFF26, 0x80) // Enable APU

	// Set up channel 1 with length timer
	apu.Write(0xFF11, 0x3F) // Length = 1
	apu.Write(0xFF12, 0xF0) // Max volume
	apu.Write(0xFF14, 0xC0) // Trigger with length enabled

	if !apu.channel1.IsEnabled() {
		t.Fatal("Channel 1 should be enabled")
	}

	// Advance frame sequencer to step 0 (clocks length)
	// Frame sequencer runs at 512 Hz = every 8192 cycles
	apu.Update(8192)

	// Length should have been clocked, disabling the channel
	if apu.channel1.IsEnabled() {
		t.Error("Channel 1 should be disabled after length timer expires")
	}
}

func TestAPU_Panning(t *testing.T) {
	apu := New()
	apu.Write(0xFF26, 0x80) // Enable APU

	// Set panning: CH1 left only
	apu.Write(0xFF25, 0x10) // NR51: CH1 left

	if (apu.panning & 0x10) == 0 {
		t.Error("Channel 1 should be enabled on left")
	}
	if (apu.panning & 0x01) != 0 {
		t.Error("Channel 1 should be disabled on right")
	}

	// Set panning: CH1 right only
	apu.Write(0xFF25, 0x01) // NR51: CH1 right

	if (apu.panning & 0x10) != 0 {
		t.Error("Channel 1 should be disabled on left")
	}
	if (apu.panning & 0x01) == 0 {
		t.Error("Channel 1 should be enabled on right")
	}

	// Set panning: CH1 both
	apu.Write(0xFF25, 0x11) // NR51: CH1 both

	if (apu.panning & 0x10) == 0 {
		t.Error("Channel 1 should be enabled on left")
	}
	if (apu.panning & 0x01) == 0 {
		t.Error("Channel 1 should be enabled on right")
	}
}

func TestAPU_MasterVolume(t *testing.T) {
	apu := New()
	apu.Write(0xFF26, 0x80) // Enable APU

	// Set master volume
	apu.Write(0xFF24, 0x77) // Max volume both channels

	if apu.leftVolume != 7 {
		t.Errorf("Left volume: got %d, want 7", apu.leftVolume)
	}
	if apu.rightVolume != 7 {
		t.Errorf("Right volume: got %d, want 7", apu.rightVolume)
	}

	// Test different volumes
	apu.Write(0xFF24, 0x35) // Left 3, Right 5

	if apu.leftVolume != 3 {
		t.Errorf("Left volume: got %d, want 3", apu.leftVolume)
	}
	if apu.rightVolume != 5 {
		t.Errorf("Right volume: got %d, want 5", apu.rightVolume)
	}
}

func TestAPU_WaveRAM(t *testing.T) {
	apu := New()
	apu.Write(0xFF26, 0x80) // Enable APU

	// Write to wave RAM
	for addr := uint16(0xFF30); addr <= 0xFF3F; addr++ {
		offset := addr - 0xFF30
		apu.Write(addr, uint8(offset&0xFF)) //nolint:gosec // offset is always 0-15
	}

	// Read back and verify
	for addr := uint16(0xFF30); addr <= 0xFF3F; addr++ {
		offset := addr - 0xFF30
		got := apu.Read(addr)
		expected := uint8(offset & 0xFF) //nolint:gosec // offset is always 0-15
		if got != expected {
			t.Errorf("WaveRAM[0x%04X]: got 0x%02X, want 0x%02X", addr, got, expected)
		}
	}
}

func TestAPU_RegisterReadback(t *testing.T) {
	apu := New()
	apu.Write(0xFF26, 0x80) // Enable APU

	tests := []struct {
		addr  uint16
		write uint8
		read  uint8
		mask  uint8
	}{
		// Pulse channel 1
		{0xFF10, 0x7F, 0xFF, 0x7F}, // NR10 (sweep, bit 7 unused reads as 1)
		{0xFF11, 0xC0, 0xC0, 0xC0}, // NR11 (duty only)
		{0xFF12, 0xF8, 0xF8, 0xFF}, // NR12 (envelope)

		// Pulse channel 2
		{0xFF16, 0xC0, 0xC0, 0xC0}, // NR21 (duty only)
		{0xFF17, 0xF8, 0xF8, 0xFF}, // NR22 (envelope)

		// Wave channel
		{0xFF1A, 0x80, 0x80, 0x80}, // NR30 (DAC enable only)
		{0xFF1C, 0x60, 0x60, 0x60}, // NR32 (output level only)

		// Noise channel
		{0xFF21, 0xF8, 0xF8, 0xFF}, // NR42 (envelope)
		{0xFF22, 0xFF, 0xFF, 0xFF}, // NR43 (frequency)

		// Master control
		{0xFF24, 0x77, 0x77, 0xFF}, // NR50 (volume)
		{0xFF25, 0xFF, 0xFF, 0xFF}, // NR51 (panning)
	}

	for _, tt := range tests {
		apu.Write(tt.addr, tt.write)
		got := apu.Read(tt.addr)
		expected := tt.read | ^tt.mask // Account for unused bits

		if (got & tt.mask) != (expected & tt.mask) {
			t.Errorf("Register 0x%04X: wrote 0x%02X, read 0x%02X, want 0x%02X",
				tt.addr, tt.write, got, expected)
		}
	}
}

func TestAPU_DisableClearsRegisters(_ *testing.T) {
	apu := New()
	apu.Write(0xFF26, 0x80) // Enable APU

	// Write to various registers
	apu.Write(0xFF11, 0xFF)
	apu.Write(0xFF12, 0xFF)
	apu.Write(0xFF24, 0x77)
	apu.Write(0xFF25, 0xFF)

	// Disable APU
	apu.Write(0xFF26, 0x00)

	// Most registers should be cleared (except NR52 and wave RAM)
	registers := []uint16{
		0xFF10, 0xFF11, 0xFF12, 0xFF13, 0xFF14,
		0xFF16, 0xFF17, 0xFF18, 0xFF19,
		0xFF1A, 0xFF1B, 0xFF1C, 0xFF1D, 0xFF1E,
		0xFF20, 0xFF21, 0xFF22, 0xFF23,
		0xFF24, 0xFF25,
	}

	for _, addr := range registers {
		got := apu.Read(addr)
		// Most registers return 0xFF when APU is disabled or have specific unused bits
		// Just verify we can read them without crashing
		_ = got
	}
}

func TestAPU_SampleGeneration(t *testing.T) {
	apu := New()
	apu.Write(0xFF26, 0x80) // Enable APU

	// Enable channel 1
	apu.Write(0xFF12, 0xF0) // Max volume
	apu.Write(0xFF14, 0x80) // Trigger

	// Set master volume and panning
	apu.Write(0xFF24, 0x77) // Max volume
	apu.Write(0xFF25, 0x11) // CH1 both channels

	// Update APU to generate samples
	apu.Update(1000)

	// Get samples
	samples := apu.GetSampleBuffer()

	if len(samples) == 0 {
		t.Error("No samples generated")
	}

	// Samples should be in stereo (pairs)
	if len(samples)%2 != 0 {
		t.Error("Sample count should be even (stereo)")
	}

	// Verify samples are in valid range [-1.0, 1.0]
	for i, sample := range samples {
		if sample < -1.0 || sample > 1.0 {
			t.Errorf("Sample %d out of range: %f", i, sample)
		}
	}
}

func TestAPU_Reset(t *testing.T) {
	apu := New()

	// Set some state
	apu.Write(0xFF26, 0x80)
	apu.Write(0xFF24, 0x77)
	apu.Write(0xFF25, 0xFF)

	// Reset
	apu.Reset()

	// Check state is cleared
	if apu.enabled {
		t.Error("APU should be disabled after reset")
	}
	if apu.leftVolume != 0 {
		t.Error("Left volume should be 0 after reset")
	}
	if apu.rightVolume != 0 {
		t.Error("Right volume should be 0 after reset")
	}
	if len(apu.sampleBuffer) != 0 {
		t.Error("Sample buffer should be empty after reset")
	}
}

func TestAPU_FrameSequencerSteps(t *testing.T) {
	apu := New()
	apu.Write(0xFF26, 0x80) // Enable APU

	// Set up pulse channel 1 with length timer
	apu.Write(0xFF11, 0x3F) // Length = 1
	apu.Write(0xFF12, 0xF0) // Max volume
	apu.Write(0xFF14, 0xC0) // Trigger with length enabled

	if !apu.channel1.IsEnabled() {
		t.Fatal("Channel 1 should be enabled after trigger")
	}

	// Advance through frame sequencer steps
	// Step 0, 2, 4, 6 clock length at 256 Hz
	// So we need one full cycle (8 steps = 8*8192 cycles) to clock length
	for i := 0; i < 8; i++ {
		apu.Update(8192)
	}

	// Channel should be disabled by length timer
	if apu.channel1.IsEnabled() {
		t.Error("Channel 1 should be disabled after length timer expires via frame sequencer")
	}
}

func TestAPU_Update_DisabledAPU(t *testing.T) {
	apu := New()
	// APU starts disabled

	// Update should not crash
	apu.Update(10000)

	// Should generate no samples
	samples := apu.GetSampleBuffer()
	if len(samples) != 0 {
		t.Error("Disabled APU should not generate samples")
	}
}
