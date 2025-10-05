package apu

// WaveChannel represents the wave channel (channel 3).
type WaveChannel struct {
	enabled    bool
	dacEnabled bool

	// Length timer
	lengthCounter uint16
	lengthEnabled bool

	// Frequency and output
	frequency   uint16
	outputLevel uint8
	phaseTimer  uint16
	wavePos     uint8

	// Wave RAM (32 4-bit samples)
	waveRAM [16]uint8

	// Register values
	nr30, nr31, nr32, nr33, nr34 uint8
}

// NewWaveChannel creates a new wave channel.
func NewWaveChannel() *WaveChannel {
	return &WaveChannel{}
}

// Update advances the channel by the given number of cycles.
func (w *WaveChannel) Update(cycles uint16) {
	if !w.enabled || !w.dacEnabled {
		return
	}

	// Update phase timer
	w.phaseTimer += cycles
	freq := (2048 - w.frequency) * 2
	for w.phaseTimer >= freq {
		w.phaseTimer -= freq
		w.wavePos = (w.wavePos + 1) % 32
	}
}

// GetSample returns the current sample output (0.0 to 1.0).
func (w *WaveChannel) GetSample() float32 {
	if !w.enabled || !w.dacEnabled {
		return 0.0
	}

	// Get 4-bit sample from wave RAM
	byteIndex := w.wavePos / 2
	nibbleShift := (1 - (w.wavePos % 2)) * 4
	sample := (w.waveRAM[byteIndex] >> nibbleShift) & 0x0F

	// Apply output level
	switch w.outputLevel {
	case 0:
		sample = 0 // Mute
	case 1:
		// 100% (no change)
	case 2:
		sample >>= 1 // 50%
	case 3:
		sample >>= 2 // 25%
	}

	// Normalize to 0.0-1.0
	return float32(sample) / 15.0
}

// ClockLength clocks the length timer.
func (w *WaveChannel) ClockLength() {
	if !w.lengthEnabled || w.lengthCounter == 0 {
		return
	}

	w.lengthCounter--
	if w.lengthCounter == 0 {
		w.enabled = false
	}
}

// trigger triggers the channel (restarts it).
func (w *WaveChannel) trigger() {
	w.enabled = true

	// Reload length counter if it's 0
	if w.lengthCounter == 0 {
		w.lengthCounter = 256
	}

	// Reset wave position
	w.wavePos = 0
	w.phaseTimer = 0

	// Channel is disabled if DAC is off
	if !w.dacEnabled {
		w.enabled = false
	}
}

// IsEnabled returns whether the channel is enabled.
func (w *WaveChannel) IsEnabled() bool {
	return w.enabled
}

// Reset resets the channel to initial state.
func (w *WaveChannel) Reset() {
	w.enabled = false
	w.dacEnabled = false
	w.lengthCounter = 0
	w.lengthEnabled = false
	w.frequency = 0
	w.outputLevel = 0
	w.phaseTimer = 0
	w.wavePos = 0
	w.nr30 = 0
	w.nr31 = 0
	w.nr32 = 0
	w.nr33 = 0
	w.nr34 = 0

	// Clear wave RAM
	for i := range w.waveRAM {
		w.waveRAM[i] = 0
	}
}

// ReadNR30 reads NR30 (DAC enable).
func (w *WaveChannel) ReadNR30() uint8 {
	return w.nr30 | 0x7F // Lower 7 bits unused
}

// WriteNR30 writes NR30 (DAC enable).
func (w *WaveChannel) WriteNR30(value uint8) {
	w.nr30 = value
	w.dacEnabled = (value & 0x80) != 0
	if !w.dacEnabled {
		w.enabled = false
	}
}

// ReadNR31 reads NR31 (length timer).
func (w *WaveChannel) ReadNR31() uint8 {
	return 0xFF // Write-only
}

// WriteNR31 writes NR31 (length timer).
func (w *WaveChannel) WriteNR31(value uint8) {
	w.nr31 = value
	w.lengthCounter = 256 - uint16(value)
}

// ReadNR32 reads NR32 (output level).
func (w *WaveChannel) ReadNR32() uint8 {
	return w.nr32 | 0x9F // Bits 7, 4-0 unused
}

// WriteNR32 writes NR32 (output level).
func (w *WaveChannel) WriteNR32(value uint8) {
	w.nr32 = value
	w.outputLevel = (value >> 5) & 0x03
}

// ReadNR33 reads NR33 (frequency low).
func (w *WaveChannel) ReadNR33() uint8 {
	return 0xFF // Write-only
}

// WriteNR33 writes NR33 (frequency low).
func (w *WaveChannel) WriteNR33(value uint8) {
	w.nr33 = value
	w.frequency = (w.frequency & 0x0700) | uint16(value)
}

// ReadNR34 reads NR34 (frequency high & control).
func (w *WaveChannel) ReadNR34() uint8 {
	return w.nr34 | 0xBF // Bits 7, 5-3 unused
}

// WriteNR34 writes NR34 (frequency high & control).
func (w *WaveChannel) WriteNR34(value uint8) {
	w.nr34 = value
	w.frequency = (w.frequency & 0x00FF) | (uint16(value&0x07) << 8)
	w.lengthEnabled = (value & 0x40) != 0

	// Trigger
	if (value & 0x80) != 0 {
		w.trigger()
	}
}

// ReadWaveRAM reads a byte from wave RAM.
func (w *WaveChannel) ReadWaveRAM(offset uint16) uint8 {
	return w.waveRAM[offset]
}

// WriteWaveRAM writes a byte to wave RAM.
func (w *WaveChannel) WriteWaveRAM(offset uint16, value uint8) {
	w.waveRAM[offset] = value
}
