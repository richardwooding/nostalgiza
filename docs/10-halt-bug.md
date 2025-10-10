# HALT Bug Documentation

## Overview

The HALT bug is a well-documented hardware quirk in the Game Boy CPU (Sharp SM83/LR35902) that occurs when the HALT instruction (opcode 0x76) is executed under specific conditions. This bug is crucial for accurate Game Boy emulation as some commercial games rely on this behavior.

## Hardware Background

### Normal Instruction Fetch Behavior

On the Game Boy CPU, most instructions follow this fetch pattern:
```
IR = [PC+]  // Read from PC into Instruction Register, then increment PC
```

This means the Program Counter (PC) is always one byte ahead of the currently executing instruction after the fetch phase.

### HALT's Unique Fetch Behavior

The HALT instruction is special - it's the **only** instruction that fetches differently:
```
IR = [PC]   // Read from PC into Instruction Register, WITHOUT incrementing PC
```

This fetch quirk is the root cause of the HALT bug.

**Source**: Game Boy CPU Internals (SonoSooS decapped chip analysis)

## HALT Bug Conditions

The bug triggers when **all** of the following conditions are met:

1. **HALT instruction executes** (opcode 0x76)
2. **IME = 0** (Interrupt Master Enable flag is disabled)
3. **Interrupt pending** (`[IE] & [IF] & 0x1F != 0`)

When these conditions are met:
- HALT exits immediately (doesn't wait for interrupt)
- **PC fails to increment normally**
- The byte immediately after HALT is **read twice**

## Technical Explanation

### What Happens on Real Hardware

1. **HALT Execution**:
   ```
   Address 0x0100: 0x76 (HALT)
   Address 0x0101: 0x04 (INC B)
   ```

2. **Fetch HALT**: `IR = [PC]` reads 0x76 from 0x0100, PC stays at 0x0100

3. **Execute HALT**: CPU enters halted state

4. **Exit HALT** (bug condition met):
   - CPU exits HALT immediately
   - PC is still at 0x0100 (the HALT instruction)

5. **Next Instruction Fetch**:
   - Should fetch from 0x0101 (INC B)
   - But due to the bug, fetches from 0x0100 again (reads HALT byte)
   - **PC doesn't increment on this fetch**

6. **Subsequent Fetch**:
   - Now fetches from 0x0101 (INC B) normally
   - PC increments to 0x0102

**Result**: The byte at 0x0101 is read twice - once as the opcode, once as its first operand (if it's a 2-byte instruction) or executed twice (if it's a 1-byte instruction).

### Example: 1-Byte Instruction

```assembly
0x0100: HALT       ; Opcode 0x76
0x0101: INC B      ; Opcode 0x04 (1-byte)
0x0102: NOP        ; Next instruction
```

With bug (IME=0, interrupt pending):
1. HALT at 0x0100
2. Exit HALT, PC still at 0x0100
3. Fetch reads 0x0101 (INC B), PC doesn't increment → PC = 0x0101
4. Execute INC B, B++
5. Fetch reads 0x0101 (INC B) again, PC increments → PC = 0x0102
6. Execute INC B again, B++
7. **Result**: B incremented TWICE

### Example: 2-Byte Instruction

```assembly
0x0100: HALT       ; Opcode 0x76
0x0101: LD A, n    ; Opcode 0x3E (2-byte)
0x0102: 0x42       ; Operand for LD A
0x0103: NOP        ; Next instruction
```

With bug (IME=0, interrupt pending):
1. HALT at 0x0100
2. Exit HALT, PC still at 0x0100
3. Fetch opcode reads 0x0101 (0x3E = LD A, n), PC doesn't increment → PC = 0x0101
4. Fetch operand reads 0x0101 (0x3E again!), PC increments → PC = 0x0102
5. Execute `LD A, 0x3E` (NOT `LD A, 0x42`)
6. **Result**: The opcode byte is used as both opcode AND operand

## Real-World Game Examples

From research on Game Boy games that rely on the HALT bug:

**Example from commercial game** (address 0x0E3A):
```
Without bug: 76 f0 8c → HALT, LDH A, [$FF8C]
With bug:    76 f0 8c → HALT, LDH A, [$FFF0], ADC H
```

The game relies on the `ADC H` instruction that only executes due to the bug!

**Source**: Tricky-to-emulate games documentation (gbdev.gg8.se)

## Emulation Implementation

### Approach 1: PC Decrement (Recommended)

Since most emulators use a unified `fetchByte()` function that always increments PC, the simplest fix is to decrement PC after fetching HALT:

```go
case 0x76: // HALT
    // HALT is special: on hardware it fetches with IR = [PC] (no increment)
    // but we've already incremented PC in fetchByte(), so undo it
    c.Registers.PC--
    c.halted = true
    return 4
```

### Approach 2: Conditional Increment in fetchByte()

Some emulators special-case HALT in the fetch function:

```go
func (c *CPU) fetchByte() uint8 {
    value := c.Memory.Read(c.Registers.PC)
    if value != 0x76 {  // Not HALT
        c.Registers.PC++
    }
    return value
}
```

**Issue**: This creates a chicken-and-egg problem - you need to know the opcode before deciding whether to increment, but you're in the middle of fetching it.

**Recommendation**: Use Approach 1 (PC decrement in HALT handler)

## Edge Cases

### 1. HALT after EI

When HALT follows an EI (Enable Interrupts) instruction:
- Interrupt is serviced normally
- Returns to the HALT instruction
- Waits for another interrupt

**Priority**: EI behavior takes precedence over HALT bug

### 2. HALT before RST

When HALT is immediately followed by an RST instruction:
- Return address points to the RST itself
- A RET would re-execute the RST
- Can cause unexpected behavior

### 3. Repeated HALT

The bug description states: "this behaviour can repeat if said byte executes another halt instruction"

If the byte after HALT is another HALT (0x76):
- First HALT exits with bug
- Reads 0x76 twice
- Executes HALT again
- Process repeats

**Warning**: This can create infinite loops if not handled carefully!

## Testing

### Blargg's halt_bug.gb Test ROM

The authoritative test for HALT bug behavior is Blargg's `halt_bug.gb` test ROM, part of his cpu_instrs test suite.

**Test ROM behavior**:
- Sets up specific HALT bug scenarios
- Verifies the byte after HALT is executed twice
- Checks PC positioning is correct
- Tests both 1-byte and 2-byte instruction cases

### Unit Test Recommendations

1. **Test PC Position**: Verify PC is at HALT opcode after HALT executes
2. **Test 1-Byte Instruction**: Verify instruction executes twice (e.g., INC B)
3. **Test 2-Byte Instruction**: Verify opcode byte used as operand
4. **Test No Bug (IME=1)**: Verify normal HALT behavior when IME=1
5. **Test No Bug (No Interrupt)**: Verify normal HALT when no interrupt pending

## References

### Primary Sources
- **Pan Docs**: https://gbdev.io/pandocs/halt.html
- **Game Boy CPU Internals** (SonoSooS): https://gist.github.com/SonoSooS/c0055300670d678b5ae8433e20bea595
- **Game Boy Complete Technical Reference** (gekkio): https://gekkio.fi/files/gb-docs/gbctr.pdf

### Additional Resources
- **Tricky-to-emulate games**: https://gbdev.gg8.se/wiki/articles/Tricky-to-emulate_games
- **Test ROMs**: https://gbdev.gg8.se/wiki/articles/Test_ROMs
- **Programming for the Gameboy/Hardware errata**: https://en.wikiversity.org/wiki/Programming_for_the_Gameboy/Hardware_errata_and_bugs

### Community Discussions
- **HALT Bug - nesdev.org**: https://forums.nesdev.org/viewtopic.php?t=14591
- **BizHawk HALT Bug Issue**: https://github.com/TASEmulators/BizHawk/issues/1187

## Implementation Notes for NostalgiZA

### Current Implementation Status
- HALT bug flag system: ✅ Implemented
- Bug condition detection: ✅ Correct (IME=0, interrupt pending)
- Bug trigger timing: ✅ On exit from HALT
- PC positioning: ❌ **ISSUE** - PC incremented during HALT fetch

### Fix Applied
Modified `internal/cpu/opcodes.go` to decrement PC after HALT fetch:
```go
case 0x76: // HALT
    c.Registers.PC--  // Undo fetchByte() increment - HALT does IR = [PC]
    c.halted = true
    return 4
```

This ensures PC is positioned at the HALT instruction when halted=true, matching hardware behavior.

### Test Coverage
- [x] TestHALT - basic HALT behavior
- [x] TestHALTBug - bug with 1-byte NOP instruction
- [x] TestHALTNoBug - no bug when IME=1
- [x] TestHALTBugWith2ByteInstruction - bug with 2-byte LD instruction
- [x] TestHALTBugPCPosition - verify PC at correct position

### Blargg Test Status
- Before fix: ❌ Timeout (infinite loop)
- After fix: ✅ Expected to pass

---

*This documentation synthesizes research from Pan Docs, decapped chip analysis, community forums, and multiple emulator implementations to provide a comprehensive understanding of the Game Boy HALT bug.*
