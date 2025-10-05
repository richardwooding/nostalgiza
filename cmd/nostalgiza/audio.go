package main

import (
	"math/rand/v2"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/richardwooding/nostalgiza/internal/apu"
)

const (
	// Audio output sample rate (Hz).
	sampleRate = 48000

	// Audio buffer size in samples (not bytes).
	// Larger buffer = more latency but less chance of underrun.
	// ~0.1 seconds of audio buffering (48000 samples/sec * 0.1 sec * 2 channels).
	audioBufferSize = 9600
)

// AudioOptions configures which audio filters are enabled.
type AudioOptions struct {
	EnableLowPass  bool // Low-pass filter for anti-aliasing
	EnableHighPass bool // High-pass filter for DC offset removal
	EnableSoftClip bool // Soft clipping (vs hard clipping)
	EnableDither   bool // Triangular dithering
}

// AudioPlayer manages audio output for the emulator.
type AudioPlayer struct {
	apu          *apu.APU
	audioContext *audio.Context
	audioPlayer  *audio.Player
	sampleBuffer []float32
	options      AudioOptions

	// High-pass filter for DC offset removal (single pole)
	hpFilterLeft  float32
	hpFilterRight float32

	// Low-pass filter for anti-aliasing (single pole)
	lpFilterLeft  float32
	lpFilterRight float32
}

// NewAudioPlayer creates a new audio player.
func NewAudioPlayer(apuInstance *apu.APU, opts AudioOptions) (*AudioPlayer, error) {
	audioContext := audio.NewContext(sampleRate)

	// Create the AudioPlayer instance first
	ap := &AudioPlayer{
		apu:          apuInstance,
		audioContext: audioContext,
		sampleBuffer: make([]float32, 0, audioBufferSize),
		options:      opts,
	}

	// Create the player using the same AudioPlayer instance
	player, err := audioContext.NewPlayer(&infiniteStream{
		player: ap,
	})
	if err != nil {
		return nil, err
	}

	// Store the player reference
	ap.audioPlayer = player

	// Set a smaller buffer size for more responsive streaming
	// This forces more frequent Read() calls
	player.SetBufferSize(time.Millisecond * 20) // ~960 samples at 48kHz

	// Set volume to 0.7 to prevent clipping and reduce distortion
	player.SetVolume(0.7)

	return ap, nil
}

// Start starts audio playback.
func (ap *AudioPlayer) Start() {
	if ap.audioPlayer != nil {
		ap.audioPlayer.Play()
	}
}

// Stop stops audio playback.
func (ap *AudioPlayer) Stop() {
	if ap.audioPlayer != nil {
		ap.audioPlayer.Pause()
	}
}

// Update updates the audio player with samples from the APU.
func (ap *AudioPlayer) Update() {
	// Get samples from APU
	samples := ap.apu.GetSampleBuffer()
	if len(samples) > 0 {
		ap.sampleBuffer = append(ap.sampleBuffer, samples...)
	}

	// Limit buffer size to prevent unbounded growth
	// Allow buffer to grow to 2x target size before trimming
	maxBufferSize := audioBufferSize * 2
	if len(ap.sampleBuffer) > maxBufferSize {
		// Drop oldest samples to maintain buffer size
		excess := len(ap.sampleBuffer) - audioBufferSize
		ap.sampleBuffer = ap.sampleBuffer[excess:]
	}
}

// Read reads audio samples for playback (implements io.Reader).
//
//nolint:gocognit // Complexity from optional filter flags for debugging
func (ap *AudioPlayer) Read(buf []byte) (int, error) {
	// Convert buffer to samples (2 bytes per sample, stereo)
	numSamples := len(buf) / 4 // 4 bytes per stereo sample (2 channels × 2 bytes)

	// Determine how many samples we can actually provide
	availableSamples := len(ap.sampleBuffer) / 2 // stereo pairs
	samplesToWrite := numSamples
	if availableSamples < numSamples {
		samplesToWrite = availableSamples
	}

	// Convert float32 samples to int16 for audio output with optional filtering
	const hpFilterFactor = 0.9999 // High-pass filter coefficient (removes DC offset)
	const lpFilterFactor = 0.90   // Low-pass filter coefficient (removes aliasing/harshness)

	for i := 0; i < samplesToWrite; i++ {
		// Left channel
		leftRaw := ap.sampleBuffer[i*2]
		left := leftRaw

		// Apply low-pass filter (if enabled)
		if ap.options.EnableLowPass {
			ap.lpFilterLeft = ap.lpFilterLeft*lpFilterFactor + leftRaw*(1.0-lpFilterFactor)
			left = ap.lpFilterLeft
		}

		// Apply high-pass filter (if enabled)
		if ap.options.EnableHighPass {
			leftHP := left - ap.hpFilterLeft
			ap.hpFilterLeft = ap.hpFilterLeft*hpFilterFactor + left*(1.0-hpFilterFactor)
			left = leftHP
		}

		// Apply soft or hard clipping
		if ap.options.EnableSoftClip { //nolint:nestif // Optional filter for debugging
			// Soft clipping using tanh-like approximation (smoother than hard clipping)
			if left > 0.9 {
				left = 0.9 + (left-0.9)*0.1
			} else if left < -0.9 {
				left = -0.9 + (left+0.9)*0.1
			}
		} else {
			// Hard clipping
			if left > 1.0 {
				left = 1.0
			} else if left < -1.0 {
				left = -1.0
			}
		}

		// Apply triangular dithering (if enabled)
		if ap.options.EnableDither {
			dither := (rand.Float32() + rand.Float32() - 1.0) / 32768.0 //nolint:gosec // Weak random is fine for audio dithering
			left += dither
		}

		leftInt16 := int16(left * 32767.0)
		buf[i*4] = byte(leftInt16)
		buf[i*4+1] = byte(leftInt16 >> 8)

		// Right channel
		rightRaw := ap.sampleBuffer[i*2+1]
		right := rightRaw

		// Apply low-pass filter (if enabled)
		if ap.options.EnableLowPass {
			ap.lpFilterRight = ap.lpFilterRight*lpFilterFactor + rightRaw*(1.0-lpFilterFactor)
			right = ap.lpFilterRight
		}

		// Apply high-pass filter (if enabled)
		if ap.options.EnableHighPass {
			rightHP := right - ap.hpFilterRight
			ap.hpFilterRight = ap.hpFilterRight*hpFilterFactor + right*(1.0-hpFilterFactor)
			right = rightHP
		}

		// Apply soft or hard clipping
		if ap.options.EnableSoftClip { //nolint:nestif // Optional filter for debugging
			// Soft clipping
			if right > 0.9 {
				right = 0.9 + (right-0.9)*0.1
			} else if right < -0.9 {
				right = -0.9 + (right+0.9)*0.1
			}
		} else {
			// Hard clipping
			if right > 1.0 {
				right = 1.0
			} else if right < -1.0 {
				right = -1.0
			}
		}

		// Apply triangular dithering (if enabled)
		if ap.options.EnableDither {
			dither := (rand.Float32() + rand.Float32() - 1.0) / 32768.0 //nolint:gosec // Weak random is fine for audio dithering
			right += dither
		}

		rightInt16 := int16(right * 32767.0)
		buf[i*4+2] = byte(rightInt16)
		buf[i*4+3] = byte(rightInt16 >> 8)
	}

	// Pad remaining samples with silence
	for i := samplesToWrite; i < numSamples; i++ {
		buf[i*4] = 0
		buf[i*4+1] = 0
		buf[i*4+2] = 0
		buf[i*4+3] = 0
	}

	// Remove consumed samples
	if samplesToWrite > 0 {
		ap.sampleBuffer = ap.sampleBuffer[samplesToWrite*2:]
	}

	return len(buf), nil
}

// infiniteStream wraps AudioPlayer to implement an infinite audio stream.
type infiniteStream struct {
	player *AudioPlayer
}

// Read implements io.Reader for infinite audio streaming.
func (s *infiniteStream) Read(buf []byte) (int, error) {
	return s.player.Read(buf)
}
