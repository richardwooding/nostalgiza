package apu

// PulseChannel represents a pulse wave channel (channels 1 and 2).
type PulseChannel struct {
	enabled    bool
	dacEnabled bool

	// Sweep (channel 1 only)
	hasSweep     bool
	sweepPeriod  uint8
	sweepNegate  bool
	sweepShift   uint8
	sweepTimer   uint8
	sweepEnabled bool
	sweepShadow  uint16

	// Length timer
	lengthCounter uint8
	lengthEnabled bool

	// Volume envelope
	envelopeVolume   uint8
	envelopeInitial  uint8
	envelopeIncrease bool
	envelopePeriod   uint8
	envelopeTimer    uint8

	// Frequency and duty
	frequency  uint16
	dutyCycle  uint8
	dutyPos    uint8
	phaseTimer uint16

	// Register values
	nr10, nr11, nr12, nr13, nr14 uint8
}

// Duty cycle patterns (8 steps each).
var dutyPatterns = [4][8]uint8{
	{0, 0, 0, 0, 0, 0, 0, 1}, // 12.5%
	{1, 0, 0, 0, 0, 0, 0, 1}, // 25%
	{1, 0, 0, 0, 0, 1, 1, 1}, // 50%
	{0, 1, 1, 1, 1, 1, 1, 0}, // 75%
}

// NewPulseChannel creates a new pulse channel.
func NewPulseChannel(hasSweep bool) *PulseChannel {
	return &PulseChannel{
		hasSweep: hasSweep,
	}
}

// Update advances the channel by the given number of cycles.
func (p *PulseChannel) Update(cycles uint16) {
	if !p.enabled || !p.dacEnabled {
		return
	}

	// Update phase timer
	p.phaseTimer += cycles
	freq := (2048 - p.frequency) * 4
	for p.phaseTimer >= freq {
		p.phaseTimer -= freq
		p.dutyPos = (p.dutyPos + 1) % 8
	}
}

// GetSample returns the current sample output (-1.0 to +1.0).
func (p *PulseChannel) GetSample() float32 {
	if !p.enabled || !p.dacEnabled {
		return 0.0
	}

	// Get current duty pattern bit
	bit := dutyPatterns[p.dutyCycle][p.dutyPos]

	// Convert to bipolar: 0 -> -1.0, 1 -> +1.0
	// This centers the waveform around 0 to avoid DC offset
	sample := float32(bit)*2.0 - 1.0

	// Apply volume (0-15) normalized to 0.0-1.0
	return sample * float32(p.envelopeVolume) / 15.0
}

// ClockLength clocks the length timer.
func (p *PulseChannel) ClockLength() {
	if !p.lengthEnabled || p.lengthCounter == 0 {
		return
	}

	p.lengthCounter--
	if p.lengthCounter == 0 {
		p.enabled = false
	}
}

// ClockEnvelope clocks the volume envelope.
func (p *PulseChannel) ClockEnvelope() {
	if p.envelopePeriod == 0 {
		return
	}

	// Only decrement if timer is not already 0 to prevent underflow
	if p.envelopeTimer > 0 {
		p.envelopeTimer--
	}

	if p.envelopeTimer == 0 {
		p.envelopeTimer = p.envelopePeriod

		if p.envelopeIncrease && p.envelopeVolume < 15 {
			p.envelopeVolume++
		} else if !p.envelopeIncrease && p.envelopeVolume > 0 {
			p.envelopeVolume--
		}
	}
}

// ClockSweep clocks the frequency sweep (channel 1 only).
func (p *PulseChannel) ClockSweep() {
	if !p.hasSweep || !p.sweepEnabled {
		return
	}

	// Only decrement if timer is not already 0 to prevent underflow
	if p.sweepTimer > 0 {
		p.sweepTimer--
	}

	if p.sweepTimer == 0 { //nolint:nestif // Game Boy sweep logic is inherently complex
		// Reload timer
		if p.sweepPeriod > 0 {
			p.sweepTimer = p.sweepPeriod
		} else {
			p.sweepTimer = 8
		}

		// Calculate new frequency
		if p.sweepPeriod > 0 {
			newFreq := p.calculateSweepFrequency()
			if newFreq <= 2047 && p.sweepShift > 0 {
				p.sweepShadow = newFreq
				p.frequency = newFreq

				// Overflow check
				_ = p.calculateSweepFrequency()
			}
		}
	}
}

// calculateSweepFrequency calculates the new frequency from sweep.
func (p *PulseChannel) calculateSweepFrequency() uint16 {
	delta := p.sweepShadow >> p.sweepShift
	var newFreq uint16
	if p.sweepNegate {
		newFreq = p.sweepShadow - delta
	} else {
		newFreq = p.sweepShadow + delta
	}

	// Overflow check
	if newFreq > 2047 {
		p.enabled = false
	}

	return newFreq
}

// trigger triggers the channel (restarts it).
func (p *PulseChannel) trigger() {
	p.enabled = true

	// Reload length counter if it's 0
	if p.lengthCounter == 0 {
		p.lengthCounter = 64
	}

	// Reset phase
	p.phaseTimer = 0

	// Reload envelope
	p.envelopeTimer = p.envelopePeriod
	p.envelopeVolume = p.envelopeInitial

	// Reload sweep (channel 1 only)
	if p.hasSweep {
		p.sweepShadow = p.frequency
		p.sweepTimer = p.sweepPeriod
		if p.sweepTimer == 0 {
			p.sweepTimer = 8
		}
		p.sweepEnabled = p.sweepPeriod > 0 || p.sweepShift > 0

		// Sweep calculation on trigger
		if p.sweepShift > 0 {
			_ = p.calculateSweepFrequency()
		}
	}

	// Channel is disabled if DAC is off
	if !p.dacEnabled {
		p.enabled = false
	}
}

// IsEnabled returns whether the channel is enabled.
func (p *PulseChannel) IsEnabled() bool {
	return p.enabled
}

// Reset resets the channel to initial state.
func (p *PulseChannel) Reset() {
	p.enabled = false
	p.dacEnabled = false
	p.lengthCounter = 0
	p.lengthEnabled = false
	p.envelopeVolume = 0
	p.envelopeInitial = 0
	p.envelopeIncrease = false
	p.envelopePeriod = 0
	p.envelopeTimer = 0
	p.frequency = 0
	p.dutyCycle = 0
	p.dutyPos = 0
	p.phaseTimer = 0
	p.nr10 = 0
	p.nr11 = 0
	p.nr12 = 0
	p.nr13 = 0
	p.nr14 = 0

	if p.hasSweep {
		p.sweepPeriod = 0
		p.sweepNegate = false
		p.sweepShift = 0
		p.sweepTimer = 0
		p.sweepEnabled = false
		p.sweepShadow = 0
	}
}

// Register read/write methods for Channel 1.

// ReadNR10 reads NR10 (sweep).
func (p *PulseChannel) ReadNR10() uint8 {
	return p.nr10 | 0x80 // Bit 7 unused, reads as 1
}

// WriteNR10 writes NR10 (sweep).
func (p *PulseChannel) WriteNR10(value uint8) {
	p.nr10 = value
	p.sweepPeriod = (value >> 4) & 0x07
	p.sweepNegate = (value & 0x08) != 0
	p.sweepShift = value & 0x07
}

// ReadNR11 reads NR11/NR21 (length timer & duty).
func (p *PulseChannel) ReadNR11() uint8 {
	return p.nr11 | 0x3F // Lower 6 bits write-only
}

// WriteNR11 writes NR11/NR21 (length timer & duty).
func (p *PulseChannel) WriteNR11(value uint8) {
	p.nr11 = value
	p.dutyCycle = (value >> 6) & 0x03
	p.lengthCounter = 64 - (value & 0x3F)
}

// ReadNR12 reads NR12/NR22 (volume envelope).
func (p *PulseChannel) ReadNR12() uint8 {
	return p.nr12
}

// WriteNR12 writes NR12/NR22 (volume envelope).
func (p *PulseChannel) WriteNR12(value uint8) {
	p.nr12 = value
	p.envelopeInitial = (value >> 4) & 0x0F
	p.envelopeIncrease = (value & 0x08) != 0
	p.envelopePeriod = value & 0x07

	// DAC is enabled if top 5 bits are non-zero
	p.dacEnabled = (value & 0xF8) != 0
	if !p.dacEnabled {
		p.enabled = false
	}
}

// ReadNR13 reads NR13/NR23 (frequency low).
func (p *PulseChannel) ReadNR13() uint8 {
	return 0xFF // Write-only
}

// WriteNR13 writes NR13/NR23 (frequency low).
func (p *PulseChannel) WriteNR13(value uint8) {
	p.nr13 = value
	p.frequency = (p.frequency & 0x0700) | uint16(value)
}

// ReadNR14 reads NR14/NR24 (frequency high & control).
func (p *PulseChannel) ReadNR14() uint8 {
	return p.nr14 | 0xBF // Bits 7, 5-3 unused
}

// WriteNR14 writes NR14/NR24 (frequency high & control).
func (p *PulseChannel) WriteNR14(value uint8) {
	p.nr14 = value
	p.frequency = (p.frequency & 0x00FF) | (uint16(value&0x07) << 8)
	p.lengthEnabled = (value & 0x40) != 0

	// Trigger
	if (value & 0x80) != 0 {
		p.trigger()
	}
}

// Channel 2 uses the same register methods but with different names.

// ReadNR21 reads NR21 (channel 2 length timer & duty).
func (p *PulseChannel) ReadNR21() uint8 {
	return p.ReadNR11()
}

// WriteNR21 writes NR21 (channel 2 length timer & duty).
func (p *PulseChannel) WriteNR21(value uint8) {
	p.WriteNR11(value)
}

// ReadNR22 reads NR22 (channel 2 volume envelope).
func (p *PulseChannel) ReadNR22() uint8 {
	return p.ReadNR12()
}

// WriteNR22 writes NR22 (channel 2 volume envelope).
func (p *PulseChannel) WriteNR22(value uint8) {
	p.WriteNR12(value)
}

// ReadNR23 reads NR23 (channel 2 frequency low).
func (p *PulseChannel) ReadNR23() uint8 {
	return p.ReadNR13()
}

// WriteNR23 writes NR23 (channel 2 frequency low).
func (p *PulseChannel) WriteNR23(value uint8) {
	p.WriteNR13(value)
}

// ReadNR24 reads NR24 (channel 2 frequency high & control).
func (p *PulseChannel) ReadNR24() uint8 {
	return p.ReadNR14()
}

// WriteNR24 writes NR24 (channel 2 frequency high & control).
func (p *PulseChannel) WriteNR24(value uint8) {
	p.WriteNR14(value)
}
