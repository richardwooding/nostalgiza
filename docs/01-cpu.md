# CPU Architecture

## Overview
The Game Boy uses a Sharp SM83 CPU, which is similar to the Intel 8080 and Zilog Z80 but with its own unique instruction set. It's an 8-bit CPU running at 4.194304 MHz (master clock) with a system clock of 1.048576 MHz.

## CPU Registers

The CPU has six 16-bit registers that can be accessed as 16-bit or as pairs of 8-bit registers:

| 16-bit | High (8-bit) | Low (8-bit) | Description |
|--------|--------------|-------------|-------------|
| AF     | A            | F           | Accumulator & Flags |
| BC     | B            | C           | General purpose |
| DE     | D            | E           | General purpose |
| HL     | H            | L           | General purpose (often used as pointer) |
| SP     | -            | -           | Stack Pointer |
| PC     | -            | -           | Program Counter |

### Accumulator (A)
The primary register for arithmetic and logic operations.

### Flags Register (F)
The lower 8 bits of the AF register contain CPU flags. Only the upper 4 bits are used:

| Bit | Name | Symbol | Description |
|-----|------|--------|-------------|
| 7   | Zero | Z      | Set if the result of an operation is zero |
| 6   | Subtraction | N | Set if the last operation was a subtraction (for BCD) |
| 5   | Half Carry | H | Set if carry from bit 3 to bit 4 (for BCD) |
| 4   | Carry | C | Set if carry/borrow or bit shifted out |
| 3-0 | -    | -      | Unused (always 0) |

### Flag Behavior

**Zero Flag (Z)**
- Set when an operation result equals zero
- Used for conditional jumps and branches

**Carry Flag (C)**
- Set when 8-bit addition exceeds $FF
- Set when 16-bit addition exceeds $FFFF
- Set when subtraction result is less than 0
- Set when rotate/shift operations shift out a "1" bit

**Half Carry Flag (H)**
- Set when there's a carry from bit 3 to bit 4
- Used primarily by the DAA (Decimal Adjust Accumulator) instruction for BCD arithmetic

**Subtraction Flag (N)**
- Set when the last operation was a subtraction
- Used by DAA instruction for BCD arithmetic

## Instruction Set

The SM83 has a CISC (Complex Instruction Set Computing) architecture with variable-length instructions:
- 1-byte instructions (most common)
- 2-byte instructions (instruction + 1 byte immediate/offset)
- 3-byte instructions (instruction + 2 byte immediate/address)
- CB-prefixed instructions (2 bytes: $CB + opcode)

### Instruction Categories

#### Load Instructions
- `LD r, r'` - Load register to register
- `LD r, n` - Load immediate value
- `LD r, (HL)` - Load from memory
- `LD (HL), r` - Store to memory
- `LD (nn), r` - Store to absolute address
- `PUSH/POP` - Stack operations

#### Arithmetic/Logic Instructions
- `ADD, ADC` - Addition (with/without carry)
- `SUB, SBC` - Subtraction (with/without carry)
- `AND, OR, XOR` - Bitwise operations
- `CP` - Compare (subtraction without storing result)
- `INC, DEC` - Increment/Decrement

#### Rotate/Shift Instructions
- `RLCA, RLA, RRCA, RRA` - Rotate accumulator
- `RLC, RL, RRC, RR` - Rotate (CB-prefixed)
- `SLA, SRA, SRL` - Shift (CB-prefixed)
- `SWAP` - Swap nibbles (CB-prefixed)

#### Bit Operations (CB-prefixed)
- `BIT n, r` - Test bit
- `SET n, r` - Set bit
- `RES n, r` - Reset bit

#### Jump/Call Instructions
- `JP nn` - Jump to address
- `JP cc, nn` - Conditional jump
- `JR n` - Relative jump
- `JR cc, n` - Conditional relative jump
- `CALL nn` - Call subroutine
- `CALL cc, nn` - Conditional call
- `RET` - Return from subroutine
- `RET cc` - Conditional return
- `RETI` - Return and enable interrupts
- `RST n` - Restart (call to fixed address)

#### Control Instructions
- `NOP` - No operation
- `HALT` - Halt CPU until interrupt
- `STOP` - Stop CPU and LCD
- `DI` - Disable interrupts
- `EI` - Enable interrupts
- `CCF` - Complement carry flag
- `SCF` - Set carry flag
- `DAA` - Decimal adjust accumulator

## Instruction Timing

Instructions are measured in **M-cycles** (machine cycles):
- 1 M-cycle = 4 clock cycles (T-states)
- Most instructions take 1-6 M-cycles
- Memory access adds cycles
- Failed conditional branches are faster than taken branches

### Example Timings
| Instruction | M-cycles | Clock cycles |
|-------------|----------|--------------|
| NOP         | 1        | 4            |
| LD r, r'    | 1        | 4            |
| LD r, n     | 2        | 8            |
| LD r, (HL)  | 2        | 8            |
| ADD A, r    | 1        | 4            |
| JP nn       | 4        | 16           |
| CALL nn     | 6        | 24           |

## Implementation Notes

### Instruction Decoding
The instruction set can be decoded using a 256-entry jump table for standard opcodes and another 256-entry table for CB-prefixed opcodes.

### Cycle Accuracy
For accurate emulation:
- Track cycles consumed by each instruction
- Update other components (PPU, timers) based on elapsed cycles
- Handle interrupt timing precisely

### Fetch-Decode-Execute Cycle
```
1. Fetch instruction byte from memory at PC
2. Increment PC
3. Decode instruction
4. Execute instruction
   - Fetch additional bytes if needed (operands)
   - Perform operation
   - Update flags as appropriate
5. Update cycle counter
6. Check for interrupts
```

### Common Implementation Pitfalls
- Incorrectly updating flags (especially half-carry)
- Not handling 16-bit arithmetic flags correctly
- Missing the difference between `ADD HL, rr` and 8-bit additions
- Incorrect CB-prefixed instruction decoding
- Not accounting for variable instruction timing

## References
- Full instruction set: https://gbdev.io/gb-opcodes/
- Opcode table: https://gbdev.io/gb-opcodes/optables/
- Pan Docs CPU section: https://gbdev.io/pandocs/CPU_Instruction_Set.html
