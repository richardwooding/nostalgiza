package apu

// NoiseChannel represents the noise channel (channel 4).
type NoiseChannel struct {
	enabled    bool
	dacEnabled bool

	// Length timer
	lengthCounter uint8
	lengthEnabled bool

	// Volume envelope
	envelopeVolume   uint8
	envelopeInitial  uint8
	envelopeIncrease bool
	envelopePeriod   uint8
	envelopeTimer    uint8

	// LFSR (Linear Feedback Shift Register)
	lfsr      uint16
	lfsrWidth bool // false = 15-bit, true = 7-bit

	// Frequency
	clockShift  uint8
	divisorCode uint8
	phaseTimer  uint16

	// Register values
	nr41, nr42, nr43, nr44 uint8
}

// Clock divisors for noise frequency.
var noiseDivisors = [8]uint16{8, 16, 32, 48, 64, 80, 96, 112}

// NewNoiseChannel creates a new noise channel.
func NewNoiseChannel() *NoiseChannel {
	return &NoiseChannel{
		lfsr: 0x7FFF, // Initialize to all 1s
	}
}

// Update advances the channel by the given number of cycles.
func (n *NoiseChannel) Update(cycles uint16) {
	if !n.enabled || !n.dacEnabled {
		return
	}

	// Update phase timer
	n.phaseTimer += cycles
	freq := noiseDivisors[n.divisorCode] << n.clockShift
	for n.phaseTimer >= freq {
		n.phaseTimer -= freq
		n.clockLFSR()
	}
}

// clockLFSR advances the Linear Feedback Shift Register.
func (n *NoiseChannel) clockLFSR() {
	// XOR bits 0 and 1
	xorResult := (n.lfsr & 0x01) ^ ((n.lfsr >> 1) & 0x01)

	// Shift right
	n.lfsr >>= 1

	// Place XOR result in bit 14
	n.lfsr |= xorResult << 14

	// If 7-bit mode, also place in bit 6
	if n.lfsrWidth {
		n.lfsr &= ^uint16(0x40) // Clear bit 6
		n.lfsr |= xorResult << 6
	}
}

// GetSample returns the current sample output (0.0 to 1.0).
func (n *NoiseChannel) GetSample() float32 {
	if !n.enabled || !n.dacEnabled {
		return 0.0
	}

	// Output is inverted bit 0 of LFSR
	bit := (^n.lfsr) & 0x01
	if bit == 0 {
		return 0.0
	}

	// Return volume (0-15) normalized to 0.0-1.0
	return float32(n.envelopeVolume) / 15.0
}

// ClockLength clocks the length timer.
func (n *NoiseChannel) ClockLength() {
	if !n.lengthEnabled || n.lengthCounter == 0 {
		return
	}

	n.lengthCounter--
	if n.lengthCounter == 0 {
		n.enabled = false
	}
}

// ClockEnvelope clocks the volume envelope.
func (n *NoiseChannel) ClockEnvelope() {
	if n.envelopePeriod == 0 {
		return
	}

	// Only decrement if timer is not already 0 to prevent underflow
	if n.envelopeTimer > 0 {
		n.envelopeTimer--
	}

	if n.envelopeTimer == 0 {
		n.envelopeTimer = n.envelopePeriod

		if n.envelopeIncrease && n.envelopeVolume < 15 {
			n.envelopeVolume++
		} else if !n.envelopeIncrease && n.envelopeVolume > 0 {
			n.envelopeVolume--
		}
	}
}

// trigger triggers the channel (restarts it).
func (n *NoiseChannel) trigger() {
	n.enabled = true

	// Reload length counter if it's 0
	if n.lengthCounter == 0 {
		n.lengthCounter = 64
	}

	// Reset phase
	n.phaseTimer = 0

	// Reload envelope
	n.envelopeTimer = n.envelopePeriod
	n.envelopeVolume = n.envelopeInitial

	// Reset LFSR
	n.lfsr = 0x7FFF

	// Channel is disabled if DAC is off
	if !n.dacEnabled {
		n.enabled = false
	}
}

// IsEnabled returns whether the channel is enabled.
func (n *NoiseChannel) IsEnabled() bool {
	return n.enabled
}

// Reset resets the channel to initial state.
func (n *NoiseChannel) Reset() {
	n.enabled = false
	n.dacEnabled = false
	n.lengthCounter = 0
	n.lengthEnabled = false
	n.envelopeVolume = 0
	n.envelopeInitial = 0
	n.envelopeIncrease = false
	n.envelopePeriod = 0
	n.envelopeTimer = 0
	n.lfsr = 0x7FFF
	n.lfsrWidth = false
	n.clockShift = 0
	n.divisorCode = 0
	n.phaseTimer = 0
	n.nr41 = 0
	n.nr42 = 0
	n.nr43 = 0
	n.nr44 = 0
}

// ReadNR41 reads NR41 (length timer).
func (n *NoiseChannel) ReadNR41() uint8 {
	return 0xFF // Write-only
}

// WriteNR41 writes NR41 (length timer).
func (n *NoiseChannel) WriteNR41(value uint8) {
	n.nr41 = value
	n.lengthCounter = 64 - (value & 0x3F)
}

// ReadNR42 reads NR42 (volume envelope).
func (n *NoiseChannel) ReadNR42() uint8 {
	return n.nr42
}

// WriteNR42 writes NR42 (volume envelope).
func (n *NoiseChannel) WriteNR42(value uint8) {
	n.nr42 = value
	n.envelopeInitial = (value >> 4) & 0x0F
	n.envelopeIncrease = (value & 0x08) != 0
	n.envelopePeriod = value & 0x07

	// DAC is enabled if top 5 bits are non-zero
	n.dacEnabled = (value & 0xF8) != 0
	if !n.dacEnabled {
		n.enabled = false
	}
}

// ReadNR43 reads NR43 (frequency & randomness).
func (n *NoiseChannel) ReadNR43() uint8 {
	return n.nr43
}

// WriteNR43 writes NR43 (frequency & randomness).
func (n *NoiseChannel) WriteNR43(value uint8) {
	n.nr43 = value
	n.clockShift = (value >> 4) & 0x0F
	n.lfsrWidth = (value & 0x08) != 0
	n.divisorCode = value & 0x07
}

// ReadNR44 reads NR44 (control).
func (n *NoiseChannel) ReadNR44() uint8 {
	return n.nr44 | 0xBF // Bits 7, 5-0 unused
}

// WriteNR44 writes NR44 (control).
func (n *NoiseChannel) WriteNR44(value uint8) {
	n.nr44 = value
	n.lengthEnabled = (value & 0x40) != 0

	// Trigger
	if (value & 0x80) != 0 {
		n.trigger()
	}
}
