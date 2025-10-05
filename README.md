# NostalgiZA

A Game Boy (DMG) emulator written in Go.

## Overview

NostalgiZA is a cycle-accurate Game Boy emulator targeting the original Game Boy (DMG - Dot Matrix Game) hardware. The goal is to provide accurate emulation of the Game Boy's CPU, graphics, audio, and input systems.

## Features

### Implemented
- [x] Sharp SM83 CPU emulation (all opcodes, flags, timing)
- [x] Memory management and bus
- [x] Cartridge loading (ROM-only and MBC1)
- [x] Picture Processing Unit (PPU) with tile-based rendering
  - Background layer with scrolling
  - Window layer
  - Sprite/object rendering (8x8 and 8x16)
  - PPU modes and timing (H-Blank, V-Blank, OAM Scan, Drawing)
  - DMA transfer for OAM
- [x] Graphics display (Ebiten integration)
- [x] Interrupt system (V-Blank, STAT, Timer, Joypad)
- [x] Joypad input (keyboard mapping)
- [x] Timer system (DIV, TIMA, TMA, TAC registers)
- [x] Test ROM support (Blargg's CPU instruction tests)

### Planned
- [ ] Audio Processing Unit (APU) with 4 sound channels
- [ ] Additional MBC support (MBC2, MBC3, MBC5)
- [ ] Save state support
- [ ] Debugger and disassembler

## Requirements

- **Go 1.25.0 or later** (intentionally using latest Go version features)

## Documentation

Comprehensive technical documentation is available in the [`docs/`](docs/) folder:

- [00-overview.md](docs/00-overview.md) - Project overview and Game Boy specifications
- [01-cpu.md](docs/01-cpu.md) - CPU architecture and instruction set
- [02-memory.md](docs/02-memory.md) - Memory map and management
- [03-cartridges.md](docs/03-cartridges.md) - Cartridge format and MBCs
- [04-graphics.md](docs/04-graphics.md) - Graphics system (PPU)
- [05-interrupts.md](docs/05-interrupts.md) - Interrupt system
- [06-input.md](docs/06-input.md) - Joypad input
- [07-timers.md](docs/07-timers.md) - Timer registers
- [08-audio.md](docs/08-audio.md) - Audio system (APU)
- [09-boot-sequence.md](docs/09-boot-sequence.md) - Boot ROM and power-up state

## Building

```bash
# Build all packages
go build ./...

# Build the nostalgiza CLI
go build ./cmd/nostalgiza
```

## Usage

### CLI Commands

```bash
# Display cartridge information
./nostalgiza info <rom-file>

# Run a Game Boy ROM with graphics
./nostalgiza run <rom-file>

# Run a test ROM and report results
./nostalgiza test <test-rom> [--timeout 30] [-v]
```

### Examples

```bash
# Show ROM information
./nostalgiza info game.gb

# Run a Game Boy ROM (opens window with graphics)
./nostalgiza run game.gb

# Run a test ROM
./nostalgiza test testdata/blargg/cpu_instrs/01-special.gb

# Run with verbose output
./nostalgiza test testdata/blargg/cpu_instrs/01-special.gb -v
```

## Testing

### Unit Tests
```bash
# Run all unit tests
go test ./...

# Run with coverage
go test -cover ./...
```

### Integration Tests (Blargg's Test ROMs)
```bash
# Download test ROMs first (see testdata/blargg/README.md)
git clone https://github.com/retrio/gb-test-roms.git

# Run integration tests
go test ./cmd/nostalgiza/...

# Skip integration tests (short mode)
go test ./cmd/nostalgiza/... -short
```

See [testdata/blargg/README.md](testdata/blargg/README.md) for detailed test ROM setup and usage.

## Resources

- [Pan Docs](https://gbdev.io/pandocs/) - The primary Game Boy technical reference
- [GB Opcode Table](https://gbdev.io/gb-opcodes/) - Complete CPU instruction reference
- [Test ROMs](https://github.com/retrio/gb-test-roms) - Test suite for emulator validation

## License

MIT License - see [LICENSE](LICENSE) for details
