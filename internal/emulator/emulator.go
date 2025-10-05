// Package emulator provides the main emulator runner that ties together
// CPU, memory, and cartridge components.
package emulator

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/richardwooding/nostalgiza/internal/cartridge"
	"github.com/richardwooding/nostalgiza/internal/cpu"
	"github.com/richardwooding/nostalgiza/internal/memory"
)

const (
	// cyclesPerIteration is the number of cycles to execute between serial output checks.
	// At 4.19 MHz, 10,000 cycles ≈ 2.4ms.
	cyclesPerIteration = 10000

	// maxSerialBufferSize limits serial output buffer to prevent unbounded growth.
	maxSerialBufferSize = 64 * 1024 // 64 KiB

	// initialSerialBufferCapacity is the initial capacity for the serial output buffer.
	initialSerialBufferCapacity = 1024
)

var (
	// ErrTimeout indicates the operation timed out.
	ErrTimeout = errors.New("timeout waiting for serial output")
)

// Emulator represents a Game Boy emulator instance.
type Emulator struct {
	CPU    *cpu.CPU
	Memory *memory.Bus
	Cart   cartridge.Cartridge // nolint:unused // Reserved for future save state/MBC features

	// Serial output buffer for test ROMs
	serialOutput []byte
}

// New creates a new emulator instance with the given ROM data.
func New(romData []byte) (*Emulator, error) {
	// Load cartridge
	cart, err := cartridge.New(romData)
	if err != nil {
		return nil, fmt.Errorf("failed to load cartridge: %w", err)
	}

	// Create memory bus and load ROM
	mem := memory.NewBus()
	if err := mem.LoadROM(romData); err != nil {
		return nil, fmt.Errorf("failed to load ROM into memory: %w", err)
	}

	// Create CPU
	c := cpu.New(mem)

	return &Emulator{
		CPU:          c,
		Memory:       mem,
		Cart:         cart,
		serialOutput: make([]byte, 0, initialSerialBufferCapacity),
	}, nil
}

// Step executes one CPU instruction and returns the number of cycles taken.
func (e *Emulator) Step() uint8 {
	return e.CPU.Step()
}

// RunCycles runs the emulator for the specified number of cycles.
func (e *Emulator) RunCycles(cycles uint64) {
	targetCycles := e.CPU.Cycles + cycles
	for e.CPU.Cycles < targetCycles {
		e.Step()
		e.handleSerialOutput()
	}
}

// RunUntilOutput runs the emulator until serial output appears or timeout is reached.
// This is useful for test ROMs that output results via serial port.
// Returns the serial output and any error.
func (e *Emulator) RunUntilOutput(timeout time.Duration) (string, error) {
	absoluteDeadline := time.Now().Add(timeout)
	lastOutputLen := 0
	lastOutputTime := time.Now()

	// Run until we get stable output or timeout
	for {
		// Check absolute deadline to prevent infinite loops
		if time.Now().After(absoluteDeadline) {
			if len(e.serialOutput) > 0 {
				return string(e.serialOutput), nil
			}
			return "", ErrTimeout
		}

		// Execute some cycles
		e.RunCycles(cyclesPerIteration)

		// Check if we got new output - only convert to string when data changes
		if len(e.serialOutput) > lastOutputLen {
			lastOutputLen = len(e.serialOutput)
			lastOutputTime = time.Now()

			// Check if output is complete (only when new data arrives)
			// Blargg's test ROMs output "Passed" or "Failed" when complete
			output := string(e.serialOutput)
			if strings.Contains(output, "Passed") || strings.Contains(output, "Failed") {
				return output, nil
			}
		}

		// Also check for stable output (no new data for a while)
		// This handles ROMs that output continuously without completion markers
		if len(e.serialOutput) > 0 && time.Since(lastOutputTime) > timeout/10 {
			return string(e.serialOutput), nil
		}
	}
}

// handleSerialOutput checks for serial output and captures it.
// Game Boy serial transfer uses:
// - 0xFF01 (SB): Serial transfer data
// - 0xFF02 (SC): Serial transfer control.
func (e *Emulator) handleSerialOutput() {
	// Read serial control register
	sc := e.Memory.Read(0xFF02)

	// Check if transfer is requested (bit 7 set)
	if sc&0x80 != 0 {
		// Read serial data
		sb := e.Memory.Read(0xFF01)

		// Append to output buffer (with size limit to prevent unbounded growth)
		if len(e.serialOutput) < maxSerialBufferSize {
			e.serialOutput = append(e.serialOutput, sb)
		}

		// Clear transfer flag
		e.Memory.Write(0xFF02, sc&0x7F)
	}
}

// GetSerialOutput returns the accumulated serial output.
func (e *Emulator) GetSerialOutput() string {
	return string(e.serialOutput)
}

// Reset resets the emulator to initial state.
func (e *Emulator) Reset() {
	e.Memory.Reset()
	e.CPU = cpu.New(e.Memory)
	e.serialOutput = make([]byte, 0, initialSerialBufferCapacity)
}
