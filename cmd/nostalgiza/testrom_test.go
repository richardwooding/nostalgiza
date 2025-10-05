package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/richardwooding/nostalgiza/internal/testrom"
)

// testROMPath returns the path to a test ROM, or skips the test if not found.
func testROMPath(t *testing.T, relPath string) string {
	t.Helper()

	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping test ROM integration test in short mode")
	}

	path := filepath.Join("../../testdata/blargg", relPath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("Test ROM not found: %s\nDownload from: https://github.com/retrio/gb-test-roms\nSee: testdata/blargg/README.md", path)
	}

	return path
}

// TestBlarggCPUInstrs tests Blargg's CPU instruction test ROMs.
func TestBlarggCPUInstrs(t *testing.T) {
	tests := []struct {
		name       string
		rom        string
		skip       bool
		skipReason string
	}{
		{"01-special", "cpu_instrs/individual/01-special.gb", false, ""},
		{"02-interrupts", "cpu_instrs/individual/02-interrupts.gb", true, "requires interrupt support (Phase 4)"},
		{"03-op sp,hl", "cpu_instrs/individual/03-op sp,hl.gb", false, ""},
		{"04-op r,imm", "cpu_instrs/individual/04-op r,imm.gb", false, ""},
		{"05-op rp", "cpu_instrs/individual/05-op rp.gb", false, ""},
		{"06-ld r,r", "cpu_instrs/individual/06-ld r,r.gb", false, ""},
		{"07-jr,jp,call,ret,rst", "cpu_instrs/individual/07-jr,jp,call,ret,rst.gb", false, ""},
		{"08-misc instrs", "cpu_instrs/individual/08-misc instrs.gb", false, ""},
		{"09-op r,r", "cpu_instrs/individual/09-op r,r.gb", false, ""},
		{"10-bit ops", "cpu_instrs/individual/10-bit ops.gb", false, ""},
		{"11-op a,(hl)", "cpu_instrs/individual/11-op a,(hl).gb", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skipf("Skipping: %s", tt.skipReason)
			}

			romPath := testROMPath(t, tt.rom)

			// Run test ROM with 30 second timeout
			result := testrom.Run(romPath, 30*time.Second)

			// Check for errors
			if result.Error != nil && !result.Timeout {
				t.Fatalf("Error running test ROM: %v", result.Error)
			}

			// Check for timeout
			if result.Timeout {
				t.Errorf("Test timed out\nOutput:\n%s", result.Output)
				return
			}

			// Check if test passed
			if !result.Passed {
				t.Errorf("Test failed\nOutput:\n%s", result.Output)
			}
		})
	}
}

// TestBlarggHaltBug tests the HALT instruction bug.
func TestBlarggHaltBug(t *testing.T) {
	romPath := testROMPath(t, "halt_bug.gb")

	result := testrom.Run(romPath, 30*time.Second)

	if result.Error != nil && !result.Timeout {
		t.Fatalf("Error running test ROM: %v", result.Error)
	}

	if result.Timeout {
		t.Errorf("Test timed out\nOutput:\n%s", result.Output)
		return
	}

	if !result.Passed {
		t.Errorf("Test failed\nOutput:\n%s", result.Output)
	}
}
