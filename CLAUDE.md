# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

NostalgiZA is a Game Boy (DMG) emulator written in Go.

- **Language**: Go 1.25.x (intentionally using latest Go version features)
- **Module**: github.com/richardwooding/nostalgiza
- **License**: MIT
- **Target Platform**: Game Boy (DMG - original monochrome model)

**Note**: This project uses Go 1.25.x intentionally to leverage the latest language features and improvements. Ensure you have Go 1.25.0 or later installed.

## Development Commands

```bash
# Build the project
go build ./...

# Build the nostalgiza CLI
go build ./cmd/nostalgiza

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run integration tests (requires Blargg's test ROMs)
go test ./cmd/nostalgiza/...

# Skip integration tests (short mode)
go test ./cmd/nostalgiza/... -short

# Run a specific test
go test -run TestName ./path/to/package

# Format code
go fmt ./...

# Run linter
golangci-lint run

# Verify linter configuration
golangci-lint config verify

# Tidy dependencies
go mod tidy
```

## CLI Usage

```bash
# Display cartridge information
./nostalgiza info <rom-file>

# Run a Game Boy ROM with graphics (Ebiten window)
./nostalgiza run <rom-file>

# Run a test ROM and report results
./nostalgiza test <test-rom> [--timeout 30] [-v]
```

## Graphics Library

The project uses **Ebiten (Ebitengine)** v2.8.7 for graphics rendering and window management:

- **Package**: `github.com/hajimehoshi/ebiten/v2`
- **Website**: https://ebitengine.org/
- **License**: Apache 2.0
- **Platform Support**: Cross-platform (macOS, Linux, Windows, mobile, web)
- **Features**: Pure Go (no CGO required on Windows), actively maintained, includes audio support

The PPU (Picture Processing Unit) renders to a framebuffer which is then displayed using Ebiten's game interface.

## Code Quality Tools

### golangci-lint v2.5.0

The project uses golangci-lint v2.5.0 for comprehensive code quality checks.

**Installation:**
```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0
```

**Configuration:**
- File: `.golangci.yml` (project root)
- Version: 2
- Validated against: https://golangci-lint.run/jsonschema/golangci.jsonschema.json

**Enabled Linters:**
- **Code Quality**: revive, govet, staticcheck, gosimple, unused
- **Error Handling**: errcheck, errorlint, err113
- **Performance**: prealloc, gocritic
- **Code Style**: gofmt, goimports, gci, whitespace
- **Bug Detection**: gosec, bodyclose, nilerr, nilnil
- **Complexity**: gocyclo, gocognit, nestif
- **Documentation**: godot, misspell
- **Testing**: testifylint, thelper
- **Miscellaneous**: ineffassign, unconvert, unparam, wastedassign

**Usage:**
```bash
# Run all linters
golangci-lint run

# Run with fixes
golangci-lint run --fix

# Verify configuration
golangci-lint config verify
```

## Project Structure

```
nostalgiza/
├── docs/           # Technical documentation from Pan Docs
│   ├── 00-overview.md
│   ├── 01-cpu.md
│   ├── 02-memory.md
│   ├── 03-cartridges.md
│   ├── 04-graphics.md
│   ├── 05-interrupts.md
│   ├── 06-input.md
│   ├── 07-timers.md
│   ├── 08-audio.md
│   └── 09-boot-sequence.md
├── cmd/
│   └── nostalgiza/ # Main CLI application
│       ├── main.go    # CLI commands
│       ├── display.go # Ebiten display integration
│       └── *_test.go  # Integration tests
├── internal/
│   ├── cpu/        # CPU emulation (implemented)
│   ├── memory/     # Memory bus and mapping (implemented)
│   ├── ppu/        # Picture Processing Unit (implemented)
│   ├── cartridge/  # Cartridge loading and MBC1 (implemented)
│   ├── emulator/   # Emulator orchestration (implemented)
│   ├── testrom/    # Test ROM runner (implemented)
│   ├── apu/        # Audio Processing Unit (planned)
│   └── input/      # Joypad input handling (planned)
└── testdata/       # Test ROMs
    └── blargg/     # Blargg's CPU instruction tests
```

## Documentation

Comprehensive Game Boy hardware documentation is in the `docs/` folder. Always refer to these docs when implementing emulator components. The docs are based on the Pan Docs (https://gbdev.io/pandocs/), the authoritative Game Boy technical reference.

## Implementation Guidance

### Implementation Status

#### Completed (Phases 1-3)
1. **CPU & Memory** ✅ (docs/01-cpu.md, docs/02-memory.md)
   - ✅ CPU registers, flags, and complete instruction set
   - ✅ Memory bus and address space mapping
   - ✅ Cycle-accurate timing

2. **Cartridge Loading** ✅ (docs/03-cartridges.md)
   - ✅ Cartridge header parsing
   - ✅ ROM-only cartridges
   - ✅ MBC1 support (most common)

3. **Graphics/PPU** ✅ (docs/04-graphics.md)
   - ✅ Tile rendering (8×8 pixels, 2bpp)
   - ✅ Background layer with scrolling (SCX/SCY)
   - ✅ Window layer
   - ✅ Sprite/object rendering (8×8 and 8×16)
   - ✅ PPU modes and timing (H-Blank, V-Blank, OAM Scan, Drawing)
   - ✅ VRAM/OAM access restrictions
   - ✅ Palette system (BGP, OBP0, OBP1)
   - ✅ Ebiten display integration

3.5. **DMA Transfer** ✅ (docs/04-graphics.md)
   - ✅ DMA transfer implementation (critical for sprite rendering in real games)
   - ✅ 160 M-cycle transfer from source to OAM
   - ✅ Memory access restrictions during DMA (HRAM only)
   - ✅ Cycle-accurate DMA timing

4. **Interrupts & Input** ✅ (docs/05-interrupts.md, docs/06-input.md)
   - ✅ V-Blank interrupt
   - ✅ Complete interrupt system (CPU handling, priority, servicing)
   - ✅ Interrupt Master Enable (IME) with EI/DI/RETI
   - ✅ EI instruction delayed enable behavior
   - ✅ HALT instruction with interrupt wake-up
   - ✅ Joypad interrupts
   - ✅ Joypad input handling (P1/JOYP register)
   - ✅ Keyboard input integration (Ebiten: Arrow keys, Z/X, Enter, Shift)
   - 🚧 STAT interrupts (LYC=LY implemented, H-Blank/V-Blank/OAM pending)

#### Planned (Phases 5+)
5. **Timers** (docs/07-timers.md)
   - ⏳ DIV and timer registers
   - ⏳ Timer interrupts

6. **Audio/APU** (docs/08-audio.md)
   - ⏳ Sound channels (can be done last)

### Code Organization
- Use standard Go project layout
- Keep components loosely coupled
- Use interfaces for major components (CPU, Memory, PPU, APU)
- Implement cycle-accurate timing from the start

### Testing
- Write unit tests for each component
- Use test ROMs (Blargg's test suite, Mooneye-GB)
- Start with simple games like Tetris
- Test each component in isolation before integration