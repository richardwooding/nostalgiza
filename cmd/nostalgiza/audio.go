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

// AudioPlayer manages audio output for the emulator.
type AudioPlayer struct {
	apu           *apu.APU
	audioContext  *audio.Context
	audioPlayer   *audio.Player
	sampleBuffer  []float32
	resampleRatio float64

	// High-pass filter for DC offset removal (single pole)
	hpFilterLeft  float32
	hpFilterRight float32

	// Low-pass filter for anti-aliasing (single pole)
	lpFilterLeft  float32
	lpFilterRight float32
}

// NewAudioPlayer creates a new audio player.
func NewAudioPlayer(apuInstance *apu.APU) (*AudioPlayer, error) {
	audioContext := audio.NewContext(sampleRate)

	// Create the AudioPlayer instance first
	ap := &AudioPlayer{
		apu:           apuInstance,
		audioContext:  audioContext,
		sampleBuffer:  make([]float32, 0, audioBufferSize),
		resampleRatio: float64(sampleRate) / 4194304.0,
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
func (ap *AudioPlayer) Read(buf []byte) (int, error) {
	// Convert buffer to samples (2 bytes per sample, stereo)
	numSamples := len(buf) / 4 // 4 bytes per stereo sample (2 channels × 2 bytes)

	// Determine how many samples we can actually provide
	availableSamples := len(ap.sampleBuffer) / 2 // stereo pairs
	samplesToWrite := numSamples
	if availableSamples < numSamples {
		samplesToWrite = availableSamples
	}

	// Convert float32 samples to int16 for audio output with filtering
	const hpFilterFactor = 0.9999 // High-pass filter coefficient (removes DC offset)
	const lpFilterFactor = 0.90   // Low-pass filter coefficient (removes aliasing/harshness)

	for i := 0; i < samplesToWrite; i++ {
		// Left channel
		leftRaw := ap.sampleBuffer[i*2]

		// Apply low-pass filter first (smooths harsh transitions)
		ap.lpFilterLeft = ap.lpFilterLeft*lpFilterFactor + leftRaw*(1.0-lpFilterFactor)

		// Then apply high-pass filter (removes DC offset)
		left := ap.lpFilterLeft - ap.hpFilterLeft
		ap.hpFilterLeft = ap.hpFilterLeft*hpFilterFactor + ap.lpFilterLeft*(1.0-hpFilterFactor)

		// Soft clipping using tanh-like approximation (smoother than hard clipping)
		// This prevents harsh distortion from clipping
		if left > 0.9 {
			left = 0.9 + (left-0.9)*0.1
		} else if left < -0.9 {
			left = -0.9 + (left+0.9)*0.1
		}

		// Apply triangular dithering to reduce quantization noise
		dither := (rand.Float32() + rand.Float32() - 1.0) / 32768.0 //nolint:gosec // Weak random is fine for audio dithering
		leftInt16 := int16((left + dither) * 32767.0)
		buf[i*4] = byte(leftInt16)
		buf[i*4+1] = byte(leftInt16 >> 8)

		// Right channel
		rightRaw := ap.sampleBuffer[i*2+1]

		// Apply low-pass filter first (smooths harsh transitions)
		ap.lpFilterRight = ap.lpFilterRight*lpFilterFactor + rightRaw*(1.0-lpFilterFactor)

		// Then apply high-pass filter (removes DC offset)
		right := ap.lpFilterRight - ap.hpFilterRight
		ap.hpFilterRight = ap.hpFilterRight*hpFilterFactor + ap.lpFilterRight*(1.0-hpFilterFactor)

		// Soft clipping using tanh-like approximation (smoother than hard clipping)
		if right > 0.9 {
			right = 0.9 + (right-0.9)*0.1
		} else if right < -0.9 {
			right = -0.9 + (right+0.9)*0.1
		}

		// Apply triangular dithering to reduce quantization noise
		dither = (rand.Float32() + rand.Float32() - 1.0) / 32768.0 //nolint:gosec // Weak random is fine for audio dithering
		rightInt16 := int16((right + dither) * 32767.0)
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
