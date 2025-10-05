# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

NostalgiZA is a Game Boy (DMG) emulator written in Go.

- **Language**: Go 1.25.0
- **Module**: github.com/richardwooding/nostalgiza
- **License**: MIT
- **Target Platform**: Game Boy (DMG - original monochrome model)

## Development Commands

```bash
# Build the project
go build ./...

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run a specific test
go test -run TestName ./path/to/package

# Format code
go fmt ./...

# Run linter (if installed)
golangci-lint run

# Tidy dependencies
go mod tidy
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
├── cmd/            # Main applications (to be created)
├── internal/       # Internal packages (to be created)
│   ├── cpu/       # CPU emulation
│   ├── memory/    # Memory management and MBCs
│   ├── ppu/       # Graphics (Picture Processing Unit)
│   ├── apu/       # Audio (Audio Processing Unit)
│   ├── cartridge/ # Cartridge loading and MBC implementations
│   └── input/     # Joypad input handling
└── pkg/           # Public packages (to be created)
```

## Documentation

Comprehensive Game Boy hardware documentation is in the `docs/` folder. Always refer to these docs when implementing emulator components. The docs are based on the Pan Docs (https://gbdev.io/pandocs/), the authoritative Game Boy technical reference.

## Implementation Guidance

### Recommended Implementation Order
1. **CPU & Memory** (docs/01-cpu.md, docs/02-memory.md)
   - Implement CPU registers, flags, and basic instruction set
   - Create memory bus and basic memory mapping

2. **Cartridge Loading** (docs/03-cartridges.md)
   - Parse cartridge headers
   - Implement ROM-only cartridges first
   - Add MBC1 support (most common)

3. **Graphics/PPU** (docs/04-graphics.md)
   - Implement basic tile rendering
   - Add background layer
   - Implement PPU modes and timing
   - Add sprites

4. **Interrupts & Input** (docs/05-interrupts.md, docs/06-input.md)
   - Implement interrupt system
   - Add joypad input
   - Wire up V-Blank interrupt

5. **Timers** (docs/07-timers.md)
   - Implement DIV and timer registers

6. **Audio/APU** (docs/08-audio.md)
   - Implement sound channels (can be done last)

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