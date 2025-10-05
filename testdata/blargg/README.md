# Blargg's Game Boy Test ROMs

This directory contains Blargg's comprehensive Game Boy test suite for validating emulator accuracy.

## Download Test ROMs

Test ROMs are not included in this repository due to licensing. Download them from:

**https://github.com/retrio/gb-test-roms**

### Installation

1. Clone or download the test ROM repository:
   ```bash
   git clone https://github.com/retrio/gb-test-roms.git
   ```

2. Copy the ROM files to this directory:
   ```bash
   # CPU instruction tests (recommended to start with)
   cp gb-test-roms/cpu_instrs/individual/*.gb testdata/blargg/cpu_instrs/

   # Or copy all test categories
   cp -r gb-test-roms/cpu_instrs testdata/blargg/
   cp -r gb-test-roms/instr_timing testdata/blargg/
   cp -r gb-test-roms/mem_timing testdata/blargg/
   ```

## Test Categories

### CPU Instructions (`cpu_instrs/`)
Tests individual CPU instruction behavior:
- `01-special.gb` - Special instructions
- `02-interrupts.gb` - Interrupt handling (requires Phase 4)
- `03-op sp,hl.gb` - SP and HL operations
- `04-op r,imm.gb` - Register and immediate operations
- `05-op rp.gb` - Register pair operations
- `06-ld r,r.gb` - Register load operations
- `07-jr,jp,call,ret,rst.gb` - Jump and call instructions
- `08-misc instrs.gb` - Miscellaneous instructions
- `09-op r,r.gb` - Register-to-register operations
- `10-bit ops.gb` - Bit operations
- `11-op a,(hl).gb` - Accumulator and (HL) operations

### Instruction Timing (`instr_timing/`)
Tests precise timing of CPU instructions (cycle-accurate).

### Memory Timing (`mem_timing/`)
Tests memory access timing and behavior.

### Other Tests
- `halt_bug.gb` - Tests HALT instruction edge case
- `interrupt_time/` - Interrupt timing tests
- `oam_bug/` - Object Attribute Memory bug tests

## Running Tests

### Command Line
```bash
# Run a single test
./nostalgiza test testdata/blargg/cpu_instrs/01-special.gb

# Run with verbose output
./nostalgiza test testdata/blargg/cpu_instrs/01-special.gb -v

# Run with custom timeout (in seconds)
./nostalgiza test testdata/blargg/cpu_instrs/01-special.gb --timeout 60
```

### Go Tests
```bash
# Run integration tests (requires test ROMs)
go test ./cmd/nostalgiza/...

# Skip tests if ROMs not available
go test ./cmd/nostalgiza/... -short
```

## Expected Output

Blargg's test ROMs output results via the Game Boy serial port. Successful tests typically output:
```
<test name>

Passed
```

Failed tests output:
```
<test name>

Failed #<error code>
```

## Current Status

Test ROM support requires:
- ✅ CPU instruction implementation (Phase 1)
- ✅ Memory system (Phase 1)
- ✅ Cartridge loading (Phase 2)
- ✅ Serial output handling (Phase 2.5)
- ❌ Interrupt system (Phase 4) - required for some tests
- ❌ Timer system (Phase 5) - required for timing tests
- ❌ PPU (Phase 3) - required for graphical tests

### Recommended Test Order
1. **CPU Instructions** - Start here
   - Tests 01, 03-11 should work without interrupts
   - Test 02 requires interrupt implementation (Phase 4)
2. **Instruction Timing** - After CPU is working
3. **Memory Timing** - After memory system is validated
4. **Interrupt/Timer Tests** - After Phases 4-5

## References

- Original source: http://blargg.parodius.com/gb-tests/ (archived)
- Current repository: https://github.com/retrio/gb-test-roms
- Test ROM documentation: See individual test ROM source files
