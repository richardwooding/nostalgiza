// Package emulator provides the main emulator runner that ties together
// CPU, memory, and cartridge components.
package emulator

import (
	"errors"
	"fmt"
	"time"

	"github.com/richardwooding/nostalgiza/internal/cartridge"
	"github.com/richardwooding/nostalgiza/internal/cpu"
	"github.com/richardwooding/nostalgiza/internal/memory"
)

var (
	// ErrTimeout indicates the operation timed out.
	ErrTimeout = errors.New("timeout waiting for serial output")
)

// Emulator represents a Game Boy emulator instance.
type Emulator struct {
	CPU    *cpu.CPU
	Memory *memory.Bus
	Cart   cartridge.Cartridge

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
		serialOutput: make([]byte, 0, 1024),
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
	startTime := time.Now()
	lastOutputLen := 0

	// Run until we get stable output or timeout
	for {
		// Check timeout
		if time.Since(startTime) > timeout {
			if len(e.serialOutput) > 0 {
				return string(e.serialOutput), nil
			}
			return "", ErrTimeout
		}

		// Execute some cycles
		e.RunCycles(10000) // Run ~10,000 cycles at a time

		// Check if we got new output
		if len(e.serialOutput) > lastOutputLen {
			lastOutputLen = len(e.serialOutput)
			startTime = time.Now() // Reset timeout on new output
		}

		// Check if output is complete
		// Blargg's test ROMs output "Passed" or "Failed" when complete
		if len(e.serialOutput) > 0 {
			output := string(e.serialOutput)
			if containsAny(output, []string{"Passed", "Failed"}) {
				return output, nil
			}
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

		// Append to output buffer
		e.serialOutput = append(e.serialOutput, sb)

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
	e.CPU = cpu.New(e.Memory)
	e.serialOutput = make([]byte, 0, 1024)
}

// containsAny checks if the string contains any of the substrings.
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
