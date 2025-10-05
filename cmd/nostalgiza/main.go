// Package main provides the nostalgiza CLI application.
package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/richardwooding/nostalgiza/internal/cartridge"
	"github.com/richardwooding/nostalgiza/internal/emulator"
	"github.com/richardwooding/nostalgiza/internal/testrom"
)

var (
	// ErrNotImplemented indicates a feature is not yet implemented.
	ErrNotImplemented = errors.New("feature not yet implemented")

	// ErrTestFailed indicates a test ROM failed.
	ErrTestFailed = errors.New("test failed")

	// ErrInvalidScale indicates the scale factor is out of valid range.
	ErrInvalidScale = errors.New("scale must be between 1 and 10")
)

// CLI represents the command-line interface structure.
type CLI struct {
	Info InfoCmd `cmd:"" help:"Display cartridge information."`
	Run  RunCmd  `cmd:"" help:"Run a Game Boy ROM."`
	Test TestCmd `cmd:"" help:"Run a test ROM and report results."`
}

// InfoCmd displays cartridge header information.
type InfoCmd struct {
	ROM string `arg:"" type:"existingfile" help:"Path to ROM file."`
}

// Run executes the info command.
func (c *InfoCmd) Run() error {
	// Read ROM file
	data, err := os.ReadFile(c.ROM)
	if err != nil {
		return fmt.Errorf("failed to read ROM: %w", err)
	}

	// Parse cartridge
	cart, err := cartridge.New(data)
	if err != nil {
		return fmt.Errorf("failed to load cartridge: %w", err)
	}

	// Display header information
	header := cart.Header()
	fmt.Printf("ROM Information:\n")
	fmt.Printf("  Title:          %s\n", header.GetTitle())
	fmt.Printf("  Cartridge Type: %s (0x%02X)\n", cartridge.CartridgeType(header.CartridgeType), header.CartridgeType)
	fmt.Printf("  ROM Size:       %d KiB (%d banks)\n", header.GetROMSizeBytes()/1024, header.GetROMBanks())
	fmt.Printf("  RAM Size:       %d KiB (%d banks)\n", header.GetRAMSizeBytes()/1024, header.GetRAMBanks())
	fmt.Printf("  Has Battery:    %v\n", cart.HasBattery())
	fmt.Printf("  CGB Flag:       0x%02X\n", header.CGBFlag)
	fmt.Printf("  SGB Flag:       0x%02X\n", header.SGBFlag)

	return nil
}

// RunCmd runs a Game Boy ROM.
type RunCmd struct {
	ROM   string `arg:"" type:"existingfile" help:"Path to ROM file."`
	Scale int    `help:"Display scale factor (1-10)." default:"3"`

	// Audio filter flags for debugging audio quality issues
	NoLowPass  bool `help:"Disable low-pass filter (anti-aliasing)."`
	NoHighPass bool `help:"Disable high-pass filter (DC offset removal)."`
	NoSoftClip bool `help:"Disable soft clipping (use hard clipping instead)."`
	NoDither   bool `help:"Disable triangular dithering."`
}

// Run executes the run command.
func (c *RunCmd) Run() error {
	// Validate scale factor
	if c.Scale < 1 || c.Scale > 10 {
		return fmt.Errorf("%w: got %d", ErrInvalidScale, c.Scale)
	}

	// Read ROM file
	data, err := os.ReadFile(c.ROM)
	if err != nil {
		return fmt.Errorf("failed to read ROM: %w", err)
	}

	// Create emulator instance
	emu, err := emulator.New(data)
	if err != nil {
		return fmt.Errorf("failed to create emulator: %w", err)
	}

	// Create display with audio filter options
	display := NewDisplay(emu, AudioOptions{
		EnableLowPass:  !c.NoLowPass,
		EnableHighPass: !c.NoHighPass,
		EnableSoftClip: !c.NoSoftClip,
		EnableDither:   !c.NoDither,
	})

	// Configure Ebiten window
	ebiten.SetWindowTitle("NostalgiZA - Game Boy Emulator")
	ebiten.SetWindowSize(160*c.Scale, 144*c.Scale)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60) // Set to 60 ticks per second (matching Game Boy ~59.73 Hz)

	// Run the emulator
	if err := ebiten.RunGame(display); err != nil {
		return fmt.Errorf("emulator error: %w", err)
	}

	return nil
}

// TestCmd runs a test ROM and reports results.
type TestCmd struct {
	ROM     string `arg:"" type:"existingfile" help:"Path to test ROM file."`
	Timeout int    `default:"30" help:"Timeout in seconds."`
	Verbose bool   `short:"v" help:"Show detailed output."`
}

// Run executes the test command.
func (c *TestCmd) Run() error {
	fmt.Printf("Running test ROM: %s\n", c.ROM)

	// Run the test ROM
	timeout := time.Duration(c.Timeout) * time.Second
	result := testrom.Run(c.ROM, timeout)

	// Display results
	fmt.Printf("Result: %s\n", result.String())

	if c.Verbose || !result.IsSuccess() {
		fmt.Printf("\nOutput:\n%s\n", result.Output)
	}

	if !result.IsSuccess() {
		return ErrTestFailed
	}

	return nil
}

func main() {
	cli := &CLI{}
	ctx := kong.Parse(cli,
		kong.Name("nostalgiza"),
		kong.Description("A Game Boy (DMG) emulator written in Go."),
		kong.UsageOnError(),
	)

	err := ctx.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
