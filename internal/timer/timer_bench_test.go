package timer

import (
	"testing"
)

// Benchmark tests to measure timer performance

func BenchmarkTimer_Disabled(b *testing.B) {
	timer := New(nil)
	timer.Write(TAC, 0x00) // Disabled

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer.Update(100)
	}
}

func BenchmarkTimer_HighFrequency(b *testing.B) {
	timer := New(nil)
	timer.Write(TAC, 0x05) // 262144 Hz (worst case - most frequent increments)
	timer.Write(TIMA, 0x00)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer.Update(100)
	}
}

func BenchmarkTimer_LowFrequency(b *testing.B) {
	timer := New(nil)
	timer.Write(TAC, 0x04) // 4096 Hz (best case - least frequent increments)
	timer.Write(TIMA, 0x00)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer.Update(100)
	}
}

func BenchmarkTimer_DIVIncrement(b *testing.B) {
	timer := New(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer.Update(256) // One DIV increment
	}
}

func BenchmarkTimer_DIVReset(b *testing.B) {
	timer := New(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer.Write(DIV, 0x00)
	}
}

func BenchmarkTimer_TACChange(b *testing.B) {
	timer := New(nil)
	frequencies := []uint8{0x04, 0x05, 0x06, 0x07}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer.Write(TAC, frequencies[i%len(frequencies)])
	}
}

func BenchmarkTimer_ReadWrite(b *testing.B) {
	timer := New(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer.Write(TIMA, 0x42)
		_ = timer.Read(TIMA)
	}
}

func BenchmarkTimer_OverflowHandling(b *testing.B) {
	interruptCount := 0
	timer := New(func() { interruptCount++ })
	timer.Write(TAC, 0x05) // 262144 Hz
	timer.Write(TMA, 0x00)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer.Write(TIMA, 0xFF)
		timer.divCounter = 0
		timer.Update(16) // Trigger overflow
	}
}

func BenchmarkTimer_MixedOperations(b *testing.B) {
	timer := New(nil)
	timer.Write(TAC, 0x05)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer.Update(50)
		_ = timer.Read(DIV)
		_ = timer.Read(TIMA)
		timer.Write(TIMA, uint8(i%256))
	}
}
