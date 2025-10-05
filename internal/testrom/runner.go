// Package testrom provides utilities for running and validating test ROMs.
package testrom

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/richardwooding/nostalgiza/internal/emulator"
)

// Result represents the result of running a test ROM.
type Result struct {
	Output  string
	Passed  bool
	Failed  bool
	Timeout bool
	Error   error
}

// Run executes a test ROM and returns the result.
func Run(romPath string, timeout time.Duration) *Result {
	result := &Result{}

	// Read ROM file
	// #nosec G304 - romPath is provided by the user via CLI argument
	data, err := os.ReadFile(romPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to read ROM: %w", err)
		return result
	}

	// Create emulator
	emu, err := emulator.New(data)
	if err != nil {
		result.Error = fmt.Errorf("failed to create emulator: %w", err)
		return result
	}

	// Run until output or timeout
	output, err := emu.RunUntilOutput(timeout)
	result.Output = output

	if err != nil {
		if errors.Is(err, emulator.ErrTimeout) {
			result.Timeout = true
		}
		result.Error = err
		return result
	}

	// Parse output for pass/fail
	// Check "Failed" first to avoid ambiguity if both strings are present
	result.Failed = strings.Contains(output, "Failed")
	result.Passed = strings.Contains(output, "Passed") && !result.Failed

	return result
}

// String returns a human-readable representation of the result.
func (r *Result) String() string {
	if r.Error != nil && !r.Timeout {
		return fmt.Sprintf("ERROR: %v", r.Error)
	}

	if r.Timeout {
		return "TIMEOUT"
	}

	if r.Passed {
		return "PASSED"
	}

	if r.Failed {
		return "FAILED"
	}

	return "UNKNOWN"
}

// IsSuccess returns true if the test passed.
func (r *Result) IsSuccess() bool {
	return r.Passed && !r.Failed && r.Error == nil
}
