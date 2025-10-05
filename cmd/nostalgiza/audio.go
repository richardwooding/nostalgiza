package main

import (
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/richardwooding/nostalgiza/internal/apu"
)

const (
	// Audio output sample rate (Hz).
	sampleRate = 48000

	// Audio buffer size in bytes.
	// Larger buffer = more latency but less chance of underrun.
	audioBufferSize = 4096
)

// AudioPlayer manages audio output for the emulator.
type AudioPlayer struct {
	apu           *apu.APU
	audioContext  *audio.Context
	audioPlayer   *audio.Player
	sampleBuffer  []float32
	resampleRatio float64
}

// NewAudioPlayer creates a new audio player.
func NewAudioPlayer(apuInstance *apu.APU) (*AudioPlayer, error) {
	audioContext := audio.NewContext(sampleRate)

	player, err := audioContext.NewPlayer(&infiniteStream{
		player: &AudioPlayer{
			apu:           apuInstance,
			audioContext:  audioContext,
			sampleBuffer:  make([]float32, 0, audioBufferSize),
			resampleRatio: float64(sampleRate) / 4194304.0, // GB CPU frequency
		},
	})
	if err != nil {
		return nil, err
	}

	ap := &AudioPlayer{
		apu:           apuInstance,
		audioContext:  audioContext,
		audioPlayer:   player,
		sampleBuffer:  make([]float32, 0, audioBufferSize),
		resampleRatio: float64(sampleRate) / 4194304.0,
	}

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
	maxBufferSize := audioBufferSize * 4
	if len(ap.sampleBuffer) > maxBufferSize {
		// Keep only the most recent samples
		ap.sampleBuffer = ap.sampleBuffer[len(ap.sampleBuffer)-maxBufferSize:]
	}
}

// Read reads audio samples for playback (implements io.Reader).
func (ap *AudioPlayer) Read(buf []byte) (int, error) {
	// Convert buffer to samples (2 bytes per sample, stereo)
	numSamples := len(buf) / 4 // 4 bytes per stereo sample (2 channels × 2 bytes)

	if len(ap.sampleBuffer) < numSamples*2 {
		// Not enough samples, return silence
		for i := range buf {
			buf[i] = 0
		}
		return len(buf), nil
	}

	// Convert float32 samples to int16 for audio output
	for i := 0; i < numSamples; i++ {
		// Left channel
		left := ap.sampleBuffer[i*2]
		leftInt16 := int16(left * 32767.0)
		buf[i*4] = byte(leftInt16)
		buf[i*4+1] = byte(leftInt16 >> 8)

		// Right channel
		right := ap.sampleBuffer[i*2+1]
		rightInt16 := int16(right * 32767.0)
		buf[i*4+2] = byte(rightInt16)
		buf[i*4+3] = byte(rightInt16 >> 8)
	}

	// Remove consumed samples
	ap.sampleBuffer = ap.sampleBuffer[numSamples*2:]

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
