# Interrupt System

## Overview
The Game Boy's interrupt system allows hardware components to pause CPU execution and call handler routines. This is essential for responding to events like V-Blank, button presses, and timer overflows.

## Interrupt Types

The Game Boy supports 5 hardware interrupts, listed by priority (highest to lowest):

| Bit | Address | Name | Description | Priority |
|-----|---------|------|-------------|----------|
| 0 | $0040 | V-Blank | Vertical blank (PPU) | 1 (highest) |
| 1 | $0048 | LCD STAT | LCD status triggers (PPU) | 2 |
| 2 | $0050 | Timer | Timer overflow | 3 |
| 3 | $0058 | Serial | Serial transfer complete | 4 |
| 4 | $0060 | Joypad | Button press | 5 (lowest) |

## Interrupt Registers

### Interrupt Enable (IE) - $FFFF
Controls which interrupts are enabled.

| Bit | Interrupt |
|-----|-----------|
| 0 | V-Blank |
| 1 | LCD STAT |
| 2 | Timer |
| 3 | Serial |
| 4 | Joypad |
| 5-7 | Unused |

- **1** = Interrupt enabled
- **0** = Interrupt disabled

### Interrupt Flag (IF) - $FF0F
Indicates which interrupts are currently requesting service.

| Bit | Interrupt |
|-----|-----------|
| 0 | V-Blank |
| 1 | LCD STAT |
| 2 | Timer |
| 3 | Serial |
| 4 | Joypad |
| 5-7 | Unused |

- **1** = Interrupt requested
- **0** = No request

### Interrupt Master Enable (IME)
Internal CPU flag (not directly readable/writable).

- **1** = Interrupts enabled globally
- **0** = Interrupts disabled globally

Modified by:
- `EI` instruction: Enables IME (takes effect after next instruction)
- `DI` instruction: Disables IME immediately
- `RETI` instruction: Returns from interrupt and enables IME
- Interrupt execution: Automatically disables IME

## Interrupt Handling Process

### Requesting an Interrupt
1. Hardware sets corresponding bit in IF ($FF0F)
2. Bit remains set until acknowledged (even if interrupt disabled)

### Interrupt Check (every instruction)
For an interrupt to be serviced:
1. IME must be 1 (globally enabled)
2. Corresponding bit in IE must be 1 (interrupt type enabled)
3. Corresponding bit in IF must be 1 (interrupt requested)

Check: `(IME) && (IE & IF & 0x1F)`

### Servicing an Interrupt
When an interrupt is serviced (5 M-cycles):

1. **Wait** for current instruction to complete
2. **Clear IME** (disable interrupts)
3. **Clear IF bit** for the serviced interrupt
4. **Push PC** onto stack (2 bytes, high then low)
5. **Jump** to interrupt handler address

Total: 5 M-cycles (20 clock cycles)

### Interrupt Priority
If multiple interrupts are requested simultaneously:
1. Check V-Blank (bit 0) first
2. Then LCD STAT (bit 1)
3. Then Timer (bit 2)
4. Then Serial (bit 3)
5. Finally Joypad (bit 4)

Only the highest priority interrupt is serviced.

### Returning from Interrupt
Use `RETI` instruction:
1. Pop PC from stack (return address)
2. Set IME = 1 (re-enable interrupts)

Alternatively, `RET` + `EI` can be used but is slower.

## Interrupt Handlers

### Handler Addresses
Interrupt handlers are typically short routines that:
1. Save register state (if needed)
2. Perform minimal work
3. Set flags for main code
4. Restore register state
5. Return with `RETI`

Example handler at $0040 (V-Blank):
```assembly
PUSH AF          ; Save registers
PUSH BC
; ... do work ...
POP BC           ; Restore registers
POP AF
RETI             ; Return and enable interrupts
```

### V-Blank Interrupt (Bit 0 - $0040)
- **Triggered**: When PPU enters mode 1 (V-Blank)
- **Timing**: After scanline 143 completes
- **Common uses**:
  - Update VRAM/OAM safely
  - Game loop synchronization
  - Frame timing

### LCD STAT Interrupt (Bit 1 - $0048)
- **Triggered**: Based on STAT register conditions:
  - LY = LYC (scanline match)
  - Mode 0 (H-Blank)
  - Mode 1 (V-Blank)
  - Mode 2 (OAM Scan)
- **Common uses**:
  - Scanline effects (parallax, window tricks)
  - Precise timing for raster effects

### Timer Interrupt (Bit 2 - $0050)
- **Triggered**: When TIMA register overflows
- **Common uses**:
  - Game timing
  - Music/sound timing
  - Periodic events

### Serial Interrupt (Bit 3 - $0058)
- **Triggered**: When serial transfer completes
- **Common uses**:
  - Link cable communication
  - Data transfer between Game Boys

### Joypad Interrupt (Bit 4 - $0060)
- **Triggered**: When any button is pressed
- **Common uses**:
  - Wake from HALT/STOP
  - Button press detection
- **Note**: Can be glitchy, often polled instead

## EI Instruction Delay

The `EI` instruction has a special behavior:
- Enables interrupts AFTER the next instruction executes
- Allows safe pattern: `EI` followed by `RET` or `HALT`

Example:
```assembly
EI       ; Enable interrupts (takes effect after next instruction)
RET      ; Return (interrupts enabled after this)
```

## Nested Interrupts

By default, interrupts cannot interrupt each other (IME is cleared when handling an interrupt).

To allow nested interrupts:
```assembly
InterruptHandler:
    PUSH AF
    EI              ; Re-enable interrupts
    ; ... do work ...
    DI              ; Disable before returning
    POP AF
    RETI
```

**Warning**: Risk of stack overflow if not careful.

## HALT Instruction

The `HALT` instruction stops CPU execution until an interrupt occurs.

### Normal Behavior
When IME = 1:
1. CPU stops executing instructions
2. When interrupt requested: service interrupt, resume execution
3. Low power consumption

### HALT Bug
When IME = 0 and an interrupt is pending:
- CPU exits HALT immediately
- **Bug**: Next instruction's first byte is executed twice
- Workaround: Check for this condition in emulator

## Implementation

### Interrupt Check (every instruction)
```go
func (cpu *CPU) checkInterrupts() {
    if !cpu.ime {
        return
    }

    // Check for pending interrupts
    pending := cpu.mem.Read(0xFF0F) & cpu.mem.Read(0xFFFF) & 0x1F

    if pending == 0 {
        return
    }

    // Find highest priority interrupt
    for bit := 0; bit < 5; bit++ {
        if pending & (1 << bit) != 0 {
            cpu.serviceInterrupt(bit)
            break
        }
    }
}
```

### Servicing Interrupt
```go
func (cpu *CPU) serviceInterrupt(bit int) {
    // Disable interrupts
    cpu.ime = false

    // Clear IF bit
    ifReg := cpu.mem.Read(0xFF0F)
    cpu.mem.Write(0xFF0F, ifReg & ^(1 << bit))

    // Push PC to stack
    cpu.pushStack(cpu.pc)

    // Jump to handler
    handlers := []uint16{0x40, 0x48, 0x50, 0x58, 0x60}
    cpu.pc = handlers[bit]

    // Consume 5 M-cycles
    cpu.cycles += 20
}
```

### Requesting Interrupt
```go
func requestInterrupt(mem *Memory, bit int) {
    ifReg := mem.Read(0xFF0F)
    mem.Write(0xFF0F, ifReg | (1 << bit))
}
```

### EI Instruction
```go
func executeEI(cpu *CPU) {
    cpu.pendingIME = true  // Enable after next instruction
}

// In main loop, after executing instruction:
if cpu.pendingIME {
    cpu.ime = true
    cpu.pendingIME = false
}
```

## Testing Interrupts

Test cases:
1. Single interrupt (V-Blank)
2. Multiple interrupts (priority)
3. Interrupt during interrupt (nested)
4. EI delay behavior
5. HALT with IME=1
6. HALT bug (IME=0)
7. IF bit persistence
8. Disabled interrupts (IE register)

## Common Pitfalls

- Not clearing IF bit when servicing interrupt (infinite loop)
- Servicing interrupts when IME = 0
- Not handling EI instruction delay
- Incorrect interrupt priority
- Not disabling IME during interrupt service
- Missing HALT bug behavior
- Not checking both IE and IF registers
- Forgetting that IF bits can be set even when interrupts disabled

## Usage Example

### V-Blank Synchronization
```assembly
; Enable V-Blank interrupt
ld a, 1
ldh (IE), a    ; Enable V-Blank in IE
ei             ; Enable interrupts globally

MainLoop:
    halt           ; Wait for V-Blank
    call UpdateGame
    jr MainLoop

; V-Blank handler at $0040
VBlankHandler:
    push af
    ; Update VRAM safely
    call UpdateGraphics
    pop af
    reti
```

## References
- Pan Docs Interrupts: https://gbdev.io/pandocs/Interrupts.html
- Pan Docs Reducing Power: https://gbdev.io/pandocs/Reducing_Power_Consumption.html
