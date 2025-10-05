// Package apu implements the Game Boy Audio Processing Unit.
//
// The APU generates sound through 4 independent channels mixed into stereo output:
//   - Channel 1: Pulse wave with frequency sweep
//   - Channel 2: Pulse wave
//   - Channel 3: Programmable wave pattern
//   - Channel 4: Noise
//
// The APU runs synchronously with the CPU at 4.194304 MHz and has a frame
// sequencer that clocks various subsystems at 512 Hz.
package apu

// APU represents the Game Boy Audio Processing Unit.
type APU struct {
	enabled bool // NR52 bit 7: Audio master enable

	// Frame sequencer (512 Hz, every 8192 CPU cycles)
	frameStep    uint8  // Current step (0-7)
	frameCounter uint16 // Cycles until next step

	// Sound channels
	channel1 *PulseChannel // Pulse with sweep
	channel2 *PulseChannel // Pulse without sweep
	channel3 *WaveChannel  // Programmable wave
	channel4 *NoiseChannel // Noise

	// Master control (NR50)
	leftVolume  uint8 // Left speaker volume (0-7)
	rightVolume uint8 // Right speaker volume (0-7)
	vinLeft     bool  // VIN to left speaker
	vinRight    bool  // VIN to right speaker

	// Sound panning (NR51)
	panning uint8 // Channel panning bits

	// Audio output
	sampleBuffer []float32 // Stereo samples (L, R, L, R, ...)
}

// New creates a new APU instance.
func New() *APU {
	return &APU{
		channel1:     NewPulseChannel(true),  // With sweep
		channel2:     NewPulseChannel(false), // Without sweep
		channel3:     NewWaveChannel(),
		channel4:     NewNoiseChannel(),
		sampleBuffer: make([]float32, 0, 4096), // Initial capacity
	}
}

// Update advances the APU by the given number of CPU cycles.
func (a *APU) Update(cycles uint16) {
	if !a.enabled {
		return
	}

	// Update frame sequencer
	a.frameCounter += cycles
	for a.frameCounter >= 8192 {
		a.frameCounter -= 8192
		a.clockFrameSequencer()
	}

	// Update each channel
	a.channel1.Update(cycles)
	a.channel2.Update(cycles)
	a.channel3.Update(cycles)
	a.channel4.Update(cycles)

	// Generate audio samples
	a.generateSamples(cycles)
}

// clockFrameSequencer advances the frame sequencer by one step.
func (a *APU) clockFrameSequencer() {
	// Length counter (256 Hz) - steps 0, 2, 4, 6
	if a.frameStep%2 == 0 {
		a.channel1.ClockLength()
		a.channel2.ClockLength()
		a.channel3.ClockLength()
		a.channel4.ClockLength()
	}

	// Sweep (128 Hz) - steps 2, 6 (channel 1 only)
	if a.frameStep == 2 || a.frameStep == 6 {
		a.channel1.ClockSweep()
	}

	// Volume envelope (64 Hz) - step 7
	if a.frameStep == 7 {
		a.channel1.ClockEnvelope()
		a.channel2.ClockEnvelope()
		a.channel4.ClockEnvelope()
	}

	a.frameStep = (a.frameStep + 1) % 8
}

// generateSamples generates audio samples for the given number of cycles.
// The Game Boy CPU runs at 4.194304 MHz, but we output at 48 kHz.
// We need to generate the correct number of samples based on elapsed cycles.
func (a *APU) generateSamples(cycles uint16) {
	// Calculate how many samples we need for this many CPU cycles
	// Sample rate: 48000 Hz
	// CPU clock: 4194304 Hz
	// Samples needed = cycles * 48000 / 4194304
	const sampleRate = 48000.0
	const cpuClock = 4194304.0
	samplesNeeded := int(float64(cycles) * sampleRate / cpuClock)

	// Generate the required number of samples
	for i := 0; i < samplesNeeded; i++ {
		// Get sample from each channel (0.0 to 1.0)
		sample1 := a.channel1.GetSample()
		sample2 := a.channel2.GetSample()
		sample3 := a.channel3.GetSample()
		sample4 := a.channel4.GetSample()

		// Mix channels for left and right outputs
		var left, right float32

		// Channel 1 panning
		if a.panning&0x10 != 0 {
			left += sample1
		}
		if a.panning&0x01 != 0 {
			right += sample1
		}

		// Channel 2 panning
		if a.panning&0x20 != 0 {
			left += sample2
		}
		if a.panning&0x02 != 0 {
			right += sample2
		}

		// Channel 3 panning
		if a.panning&0x40 != 0 {
			left += sample3
		}
		if a.panning&0x04 != 0 {
			right += sample3
		}

		// Channel 4 panning
		if a.panning&0x80 != 0 {
			left += sample4
		}
		if a.panning&0x08 != 0 {
			right += sample4
		}

		// Apply master volume (0-7)
		left *= float32(a.leftVolume) / 7.0
		right *= float32(a.rightVolume) / 7.0

		// Normalize (4 channels max)
		left /= 4.0
		right /= 4.0

		// Add to output buffer (stereo interleaved)
		a.sampleBuffer = append(a.sampleBuffer, left, right)
	}
}

// Read reads an APU register.
func (a *APU) Read(addr uint16) uint8 {
	// When APU is disabled, all registers read as 0 except NR52
	if !a.enabled && addr != 0xFF26 {
		return 0x00
	}

	switch addr {
	// Channel 1 - Pulse with sweep
	case 0xFF10:
		return a.channel1.ReadNR10()
	case 0xFF11:
		return a.channel1.ReadNR11()
	case 0xFF12:
		return a.channel1.ReadNR12()
	case 0xFF13:
		return a.channel1.ReadNR13()
	case 0xFF14:
		return a.channel1.ReadNR14()

	case 0xFF15: // NR15 - Unused register
		return 0xFF // Unused register reads as 0xFF

	// Channel 2 - Pulse
	case 0xFF16:
		return a.channel2.ReadNR21()
	case 0xFF17:
		return a.channel2.ReadNR22()
	case 0xFF18:
		return a.channel2.ReadNR23()
	case 0xFF19:
		return a.channel2.ReadNR24()

	// Channel 3 - Wave
	case 0xFF1A:
		return a.channel3.ReadNR30()
	case 0xFF1B:
		return a.channel3.ReadNR31()
	case 0xFF1C:
		return a.channel3.ReadNR32()
	case 0xFF1D:
		return a.channel3.ReadNR33()
	case 0xFF1E:
		return a.channel3.ReadNR34()

	// Wave RAM
	case 0xFF30, 0xFF31, 0xFF32, 0xFF33, 0xFF34, 0xFF35, 0xFF36, 0xFF37,
		0xFF38, 0xFF39, 0xFF3A, 0xFF3B, 0xFF3C, 0xFF3D, 0xFF3E, 0xFF3F:
		return a.channel3.ReadWaveRAM(addr - 0xFF30)

	// Channel 4 - Noise
	case 0xFF20:
		return a.channel4.ReadNR41()
	case 0xFF21:
		return a.channel4.ReadNR42()
	case 0xFF22:
		return a.channel4.ReadNR43()
	case 0xFF23:
		return a.channel4.ReadNR44()

	// Master control
	case 0xFF24: // NR50
		return a.readNR50()
	case 0xFF25: // NR51
		return a.panning
	case 0xFF26: // NR52
		return a.readNR52()

	default:
		return 0xFF
	}
}

// Write writes to an APU register.
func (a *APU) Write(addr uint16, value uint8) {
	// Special case: NR52 can always be written
	if addr == 0xFF26 {
		a.writeNR52(value)
		return
	}

	// When APU is disabled, all other registers are read-only
	if !a.enabled {
		return
	}

	switch addr {
	// Channel 1 - Pulse with sweep
	case 0xFF10:
		a.channel1.WriteNR10(value)
	case 0xFF11:
		a.channel1.WriteNR11(value)
	case 0xFF12:
		a.channel1.WriteNR12(value)
	case 0xFF13:
		a.channel1.WriteNR13(value)
	case 0xFF14:
		a.channel1.WriteNR14(value)

	case 0xFF15: // NR15 - Unused register
		// Unused register - writes are ignored

	// Channel 2 - Pulse
	case 0xFF16:
		a.channel2.WriteNR21(value)
	case 0xFF17:
		a.channel2.WriteNR22(value)
	case 0xFF18:
		a.channel2.WriteNR23(value)
	case 0xFF19:
		a.channel2.WriteNR24(value)

	// Channel 3 - Wave
	case 0xFF1A:
		a.channel3.WriteNR30(value)
	case 0xFF1B:
		a.channel3.WriteNR31(value)
	case 0xFF1C:
		a.channel3.WriteNR32(value)
	case 0xFF1D:
		a.channel3.WriteNR33(value)
	case 0xFF1E:
		a.channel3.WriteNR34(value)

	// Wave RAM
	case 0xFF30, 0xFF31, 0xFF32, 0xFF33, 0xFF34, 0xFF35, 0xFF36, 0xFF37,
		0xFF38, 0xFF39, 0xFF3A, 0xFF3B, 0xFF3C, 0xFF3D, 0xFF3E, 0xFF3F:
		a.channel3.WriteWaveRAM(addr-0xFF30, value)

	// Channel 4 - Noise
	case 0xFF20:
		a.channel4.WriteNR41(value)
	case 0xFF21:
		a.channel4.WriteNR42(value)
	case 0xFF22:
		a.channel4.WriteNR43(value)
	case 0xFF23:
		a.channel4.WriteNR44(value)

	// Master control
	case 0xFF24: // NR50
		a.writeNR50(value)
	case 0xFF25: // NR51
		a.panning = value
	}
}

// readNR50 reads the NR50 register (master volume).
func (a *APU) readNR50() uint8 {
	var value uint8
	if a.vinLeft {
		value |= 0x80
	}
	value |= (a.leftVolume & 0x07) << 4
	if a.vinRight {
		value |= 0x08
	}
	value |= a.rightVolume & 0x07
	return value
}

// writeNR50 writes to the NR50 register (master volume).
func (a *APU) writeNR50(value uint8) {
	a.vinLeft = (value & 0x80) != 0
	a.leftVolume = (value >> 4) & 0x07
	a.vinRight = (value & 0x08) != 0
	a.rightVolume = value & 0x07
}

// readNR52 reads the NR52 register (audio master control).
func (a *APU) readNR52() uint8 {
	var value uint8
	if a.enabled {
		value |= 0x80
	}
	// Bits 3-0: Channel enable status (read-only)
	if a.channel1.IsEnabled() {
		value |= 0x01
	}
	if a.channel2.IsEnabled() {
		value |= 0x02
	}
	if a.channel3.IsEnabled() {
		value |= 0x04
	}
	if a.channel4.IsEnabled() {
		value |= 0x08
	}
	// Bits 6-4 are unused, read as 1
	value |= 0x70
	return value
}

// writeNR52 writes to the NR52 register (audio master control).
func (a *APU) writeNR52(value uint8) {
	wasEnabled := a.enabled
	a.enabled = (value & 0x80) != 0

	// If APU is being disabled, clear all registers
	if wasEnabled && !a.enabled {
		a.reset()
	}

	// If APU is being enabled, reset frame sequencer
	if !wasEnabled && a.enabled {
		a.frameStep = 0
		a.frameCounter = 0
	}
}

// reset clears all APU registers (called when APU is disabled).
func (a *APU) reset() {
	a.channel1.Reset()
	a.channel2.Reset()
	a.channel3.Reset()
	a.channel4.Reset()
	a.leftVolume = 0
	a.rightVolume = 0
	a.vinLeft = false
	a.vinRight = false
	a.panning = 0
	a.frameStep = 0
	a.frameCounter = 0
}

// GetSampleBuffer returns the current audio sample buffer and clears it.
func (a *APU) GetSampleBuffer() []float32 {
	// Warn if buffer is growing too large (indicates Update() isn't being called regularly)
	const maxBufferSize = 48000 * 2 // 1 second of stereo samples at 48kHz
	if len(a.sampleBuffer) > maxBufferSize {
		// Truncate buffer to prevent unbounded growth
		a.sampleBuffer = a.sampleBuffer[len(a.sampleBuffer)-maxBufferSize:]
	}

	samples := a.sampleBuffer
	a.sampleBuffer = make([]float32, 0, 4096)
	return samples
}

// Reset resets the APU to initial state.
func (a *APU) Reset() {
	a.enabled = false
	a.reset()
}
