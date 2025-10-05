# Audio System (APU)

## Overview
The Game Boy's Audio Processing Unit (APU) generates sound through 4 independent channels mixed into stereo output. The APU runs synchronously with the CPU and PPU.

## Sound Channels

The Game Boy has 4 sound channels:

| Channel | Type | Description |
|---------|------|-------------|
| 1 | Pulse with Sweep | Square wave with frequency sweep |
| 2 | Pulse | Square wave without sweep |
| 3 | Wave | Programmable 4-bit wave pattern |
| 4 | Noise | Pseudo-random noise |

## Audio Registers

### Master Control

| Address | Name | Description |
|---------|------|-------------|
| FF24 | NR50 | Master volume & VIN panning |
| FF25 | NR51 | Sound panning |
| FF26 | NR52 | Audio master control |

### Channel 1 - Pulse with Sweep

| Address | Name | Description |
|---------|------|-------------|
| FF10 | NR10 | Sweep control |
| FF11 | NR11 | Length timer & duty cycle |
| FF12 | NR12 | Volume envelope |
| FF13 | NR13 | Frequency low |
| FF14 | NR14 | Frequency high & control |

### Channel 2 - Pulse

| Address | Name | Description |
|---------|------|-------------|
| FF16 | NR21 | Length timer & duty cycle |
| FF17 | NR22 | Volume envelope |
| FF18 | NR23 | Frequency low |
| FF19 | NR24 | Frequency high & control |

### Channel 3 - Wave

| Address | Name | Description |
|---------|------|-------------|
| FF1A | NR30 | DAC enable |
| FF1B | NR31 | Length timer |
| FF1C | NR32 | Output level |
| FF1D | NR33 | Frequency low |
| FF1E | NR34 | Frequency high & control |
| FF30-FF3F | Wave RAM | 32 4-bit samples |

### Channel 4 - Noise

| Address | Name | Description |
|---------|------|-------------|
| FF20 | NR41 | Length timer |
| FF21 | NR42 | Volume envelope |
| FF22 | NR43 | Frequency & randomness |
| FF23 | NR44 | Control |

## NR52 - Audio Master Control ($FF26)

| Bit | Name | Description |
|-----|------|-------------|
| 7 | Power | All sound on/off (1=On, 0=Off) |
| 6-4 | - | Unused |
| 3 | Channel 4 | Channel 4 status (read-only) |
| 2 | Channel 3 | Channel 3 status (read-only) |
| 1 | Channel 2 | Channel 2 status (read-only) |
| 0 | Channel 1 | Channel 1 status (read-only) |

**Important**: When bit 7 is cleared, all audio registers are cleared and cannot be written (except NR52 itself).

## NR51 - Sound Panning ($FF25)

| Bit | Description |
|-----|-------------|
| 7 | Channel 4 → Left |
| 6 | Channel 3 → Left |
| 5 | Channel 2 → Left |
| 4 | Channel 1 → Left |
| 3 | Channel 4 → Right |
| 2 | Channel 3 → Right |
| 1 | Channel 2 → Right |
| 0 | Channel 1 → Right |

Each channel can be independently enabled for left and/or right speaker.

## NR50 - Master Volume ($FF24)

| Bit | Description |
|-----|-------------|
| 7 | VIN → Left enable |
| 6-4 | Left volume (0-7) |
| 3 | VIN → Right enable |
| 2-0 | Right volume (0-7) |

VIN is an external audio input (cartridge), rarely used.

## Channel 1/2 - Pulse Waves

### Duty Cycle (NR11/NR21 bits 7-6)

| Value | Duty | Waveform |
|-------|------|----------|
| 00 | 12.5% | \_\_\_\_\_\_\_ |
| 01 | 25% | \_\_\_\_\_\_\_\_ |
| 10 | 50% | \_\_\_\_\_\_\_\_ |
| 11 | 75% | \_\_\_\_\_\_\_\_ |

### Length Timer (NR11/NR21 bits 5-0)
- 6-bit value (0-63)
- Channel stops after (64 - length) / 64 seconds
- Only active if triggered with length enable

### Volume Envelope (NR12/NR22)

| Bit | Description |
|-----|-------------|
| 7-4 | Initial volume (0-15) |
| 3 | Direction (0=Decrease, 1=Increase) |
| 2-0 | Sweep pace (0=disabled, 1-7=steps) |

Envelope changes volume over time:
- Each step takes: pace / 64 seconds
- Volume increases or decreases by 1 each step
- Stops at 0 or 15

### Frequency (NR13-14/NR23-24)
- 11-bit value (0-2047)
- Actual frequency: 131072 / (2048 - frequency) Hz
- NR13/23: Lower 8 bits
- NR14/24 bits 2-0: Upper 3 bits

### Control (NR14/NR24)

| Bit | Description |
|-----|-------------|
| 7 | Trigger (1=Restart channel) |
| 6 | Length enable (1=Use length timer) |
| 5-3 | Unused |
| 2-0 | Frequency high bits |

### Sweep (Channel 1 only - NR10)

| Bit | Description |
|-----|-------------|
| 6-4 | Sweep time (0=disabled, 1-7) |
| 3 | Direction (0=Increase, 1=Decrease) |
| 2-0 | Shift (0-7) |

Frequency sweep calculation:
```
new_freq = freq ± (freq >> shift)
```

Sweep changes frequency periodically:
- Period: sweep_time / 128 seconds
- Continues until overflow or channel stops

## Channel 3 - Wave

### DAC Enable (NR30)
- Bit 7: DAC on/off (1=On, 0=Off)
- Channel must be enabled to produce sound

### Wave RAM ($FF30-$FF3F)
- 32 4-bit samples (16 bytes, 2 samples per byte)
- Each byte: high nibble = first sample, low nibble = second sample
- Played sequentially in a loop

### Output Level (NR32 bits 6-5)

| Value | Volume |
|-------|--------|
| 00 | Mute (0%) |
| 01 | 100% |
| 10 | 50% |
| 11 | 25% |

### Frequency (NR33-34)
- 11-bit value
- Sample rate: 2097152 / (2048 - frequency) Hz
- Each sample plays for (2048 - frequency) / 2 cycles

## Channel 4 - Noise

### Frequency & Randomness (NR43)

| Bit | Description |
|-----|-------------|
| 7-4 | Clock shift (s) |
| 3 | LFSR width (0=15bit, 1=7bit) |
| 2-0 | Clock divider (r) |

Frequency calculation:
```
frequency = 524288 / r / 2^(s+1)

where r = divisor[NR43 & 7]
divisor = [8, 16, 32, 48, 64, 80, 96, 112]
```

LFSR width:
- 15-bit: Smooth white noise
- 7-bit: Metallic/harsh noise

### Linear Feedback Shift Register (LFSR)
Generates pseudo-random bit pattern:
1. XOR bits 0 and 1
2. Shift register right
3. Place XOR result in bit 14 (and bit 6 if 7-bit mode)
4. Output bit 0

## APU Timing

### Frame Sequencer
The APU has an internal frame sequencer running at 512 Hz:

| Step | Length | Sweep | Envelope |
|------|--------|-------|----------|
| 0 | Clock | - | - |
| 1 | - | - | - |
| 2 | Clock | Clock | - |
| 3 | - | - | - |
| 4 | Clock | - | - |
| 5 | - | - | - |
| 6 | Clock | Clock | - |
| 7 | - | - | Clock |

- **512 Hz**: Frame sequencer step (every 8192 cycles)
- **256 Hz**: Length timer (steps 0, 2, 4, 6)
- **128 Hz**: Sweep (steps 2, 6) - Channel 1 only
- **64 Hz**: Volume envelope (step 7)

## Implementation Strategy

### APU State
```go
type APU struct {
    enabled bool

    // Frame sequencer
    frameStep int
    frameCounter int

    // Channels
    channel1 PulseChannel  // With sweep
    channel2 PulseChannel  // Without sweep
    channel3 WaveChannel
    channel4 NoiseChannel

    // Master control
    leftVolume  uint8
    rightVolume uint8
    panning     uint8

    // Audio output buffer
    sampleRate int
    buffer []float32
}
```

### Update Cycle
```go
func (apu *APU) Update(cycles int) {
    if !apu.enabled {
        return
    }

    // Update frame sequencer
    apu.frameCounter += cycles
    if apu.frameCounter >= 8192 {
        apu.frameCounter -= 8192
        apu.clockFrameSequencer()
    }

    // Update each channel
    apu.channel1.Update(cycles)
    apu.channel2.Update(cycles)
    apu.channel3.Update(cycles)
    apu.channel4.Update(cycles)

    // Generate audio samples
    apu.generateSamples(cycles)
}
```

### Sample Generation
```go
func (apu *APU) generateSamples(cycles int) {
    // Get sample from each channel
    sample1 := apu.channel1.GetSample()
    sample2 := apu.channel2.GetSample()
    sample3 := apu.channel3.GetSample()
    sample4 := apu.channel4.GetSample()

    // Mix channels for left/right
    left := 0.0
    right := 0.0

    if apu.panning & 0x10 != 0 { left += sample1 }
    if apu.panning & 0x01 != 0 { right += sample1 }
    if apu.panning & 0x20 != 0 { left += sample2 }
    if apu.panning & 0x02 != 0 { right += sample2 }
    // ... etc for channels 3 and 4

    // Apply master volume
    left *= float64(apu.leftVolume) / 7.0
    right *= float64(apu.rightVolume) / 7.0

    // Add to output buffer
    apu.buffer = append(apu.buffer, float32(left), float32(right))
}
```

## Audio Output

### Sample Rate
Common choices:
- **44100 Hz**: CD quality
- **48000 Hz**: DAC standard
- **22050 Hz**: Lower quality, better performance

### Downsampling
Game Boy generates samples at ~1MHz, needs downsampling:
1. Accumulate samples over time window
2. Average or use low-pass filter
3. Output at target sample rate

### Buffer Management
- Use ring buffer for audio samples
- Audio callback reads from buffer
- APU writes to buffer
- Handle buffer underrun/overrun

## Testing

### Test Cases
1. Each channel independently
2. Channel enable/disable
3. Panning (left/right)
4. Master volume
5. Length timers
6. Envelopes
7. Frequency changes
8. Wave RAM playback
9. Noise generation
10. APU power on/off

### Test ROMs
- **Blargg's sound tests**: dmg_sound test ROMs
- Listen for correct pitch, volume, timing

## Implementation Priority

### Minimal Implementation
1. Channel enable/disable flags
2. Dummy sound generation (silence)
3. Register reads/writes

### Basic Audio
1. Channel 1/2 square waves
2. Simple frequency generation
3. Basic mixing and panning

### Complete Audio
1. Volume envelopes
2. Length timers
3. Frequency sweep (channel 1)
4. Wave channel (channel 3)
5. Noise channel (channel 4)
6. Frame sequencer

## Common Pitfalls

- Not clearing registers when APU is disabled
- Incorrect channel enable/disable logic
- Missing frame sequencer timing
- Wrong frequency calculations
- Incorrect LFSR implementation for noise
- Not implementing length timer correctly
- Missing envelope updates
- Incorrect wave RAM access during playback
- Wrong duty cycle patterns
- Not handling trigger bit correctly
- Buffer underrun/overrun issues

## Performance Optimization

### Sample Generation
- Generate samples at lower rate
- Use lookup tables for waveforms
- Optimize mixing
- Only update active channels

### Skip Rendering
If audio disabled or muted:
- Still update registers
- Skip sample generation
- Maintain timing

## References
- Pan Docs Audio: https://gbdev.io/pandocs/Audio.html
- Pan Docs Audio Details: https://gbdev.io/pandocs/Audio_details.html
- Game Boy Sound Hardware: https://gbdev.gg8.se/wiki/articles/Gameboy_sound_hardware
