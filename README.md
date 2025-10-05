# NostalgiZA

A Game Boy (DMG) emulator written in Go.

## Overview

NostalgiZA is a cycle-accurate Game Boy emulator targeting the original Game Boy (DMG - Dot Matrix Game) hardware. The goal is to provide accurate emulation of the Game Boy's CPU, graphics, audio, and input systems.

## Features (Planned)

- [ ] Sharp SM83 CPU emulation
- [ ] Picture Processing Unit (PPU) with tile-based rendering
- [ ] Audio Processing Unit (APU) with 4 sound channels
- [ ] Memory Bank Controller (MBC) support
- [ ] Joypad input
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
go build ./...
```

## Testing

```bash
go test ./...
```

## Resources

- [Pan Docs](https://gbdev.io/pandocs/) - The primary Game Boy technical reference
- [GB Opcode Table](https://gbdev.io/gb-opcodes/) - Complete CPU instruction reference
- [Test ROMs](https://github.com/retrio/gb-test-roms) - Test suite for emulator validation

## License

MIT License - see [LICENSE](LICENSE) for details
