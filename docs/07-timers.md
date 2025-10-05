# Timer and Divider Registers

## Overview
The Game Boy has a timer system consisting of a divider register and a programmable timer. These provide timing functionality for games and can trigger interrupts.

## Timer Registers

| Address | Name | Description | Access |
|---------|------|-------------|--------|
| FF04 | DIV | Divider Register | R/W |
| FF05 | TIMA | Timer Counter | R/W |
| FF06 | TMA | Timer Modulo | R/W |
| FF07 | TAC | Timer Control | R/W |

## DIV - Divider Register ($FF04)

### Behavior
- **Size**: 8-bit register (16-bit internal counter)
- **Increment Rate**: 16384 Hz (every 256 clock cycles)
  - In CGB Double Speed Mode: 32768 Hz
- **Always Running**: Cannot be stopped
- **Reset on Write**: Writing any value resets DIV to $00

### Internal Counter
- DIV is the upper 8 bits of a 16-bit counter
- Counter increments every clock cycle
- DIV increments every 256 CPU cycles (64 M-cycles)

### Uses
- Random number generation
- Simple timing delays
- Frame-independent timing

### Implementation
```go
type Timer struct {
    divCounter uint16  // Internal 16-bit counter
    // ... other fields
}

func (t *Timer) updateDIV(cycles int) {
    oldDiv := t.divCounter >> 8
    t.divCounter += uint16(cycles)
    newDiv := t.divCounter >> 8

    // DIV is upper 8 bits
    // Check for overflow to trigger timer
}

func (t *Timer) readDIV() uint8 {
    return uint8(t.divCounter >> 8)
}

func (t *Timer) writeDIV(value uint8) {
    // Any write resets to 0
    t.divCounter = 0
}
```

## TIMA - Timer Counter ($FF05)

### Behavior
- **Size**: 8-bit register
- **Increment Rate**: Configurable via TAC register
- **Overflow**: When TIMA overflows ($FF → $00):
  1. TIMA is reset to TMA value
  2. Timer interrupt is requested (IF bit 2)

### Overflow Timing
The overflow behavior has specific timing:
1. TIMA overflows from $FF to $00
2. **Next cycle**: TIMA is reloaded with TMA value
3. **Same cycle**: Timer interrupt is requested

### Uses
- Game timing loops
- Sound/music timing
- Periodic events
- Animation timing

## TMA - Timer Modulo ($FF06)

### Behavior
- **Size**: 8-bit register
- **Purpose**: Value loaded into TIMA when it overflows
- **Default**: Usually $00
- **Allows**: Custom interrupt frequencies

### Calculating Interrupt Frequency
```
Frequency = Input Clock / (256 - TMA)
```

Example:
- Input clock: 4096 Hz
- TMA: $F0 (240)
- Overflow every: 256 - 240 = 16 counts
- Interrupt rate: 4096 / 16 = 256 Hz

## TAC - Timer Control ($FF07)

### Register Format

| Bit | Name | Description |
|-----|------|-------------|
| 7-3 | - | Unused |
| 2 | Enable | Timer enable (1=On, 0=Off) |
| 1-0 | Clock | Clock select (input frequency) |

### Clock Select (Bits 1-0)

| Value | Frequency | Period |
|-------|-----------|--------|
| 00 | 4096 Hz | 1024 CPU cycles |
| 01 | 262144 Hz | 16 CPU cycles |
| 10 | 65536 Hz | 64 CPU cycles |
| 11 | 16384 Hz | 256 CPU cycles |

### Enable Bit
- **1**: Timer counting enabled
- **0**: Timer stopped (TIMA doesn't increment)
- DIV always runs regardless of this bit

## Timer Operation

### Update Flow
1. CPU executes instruction (N cycles)
2. Update DIV counter (every cycle)
3. If timer enabled:
   - Update timer counter based on TAC frequency
   - If TIMA overflows:
     - Request timer interrupt
     - Reload TIMA with TMA

### Frequency Divider
The timer uses the DIV internal counter to derive its frequencies:

| TAC (1-0) | Bit of DIV Counter | Frequency |
|-----------|-------------------|-----------|
| 00 | Bit 9 | 4096 Hz |
| 01 | Bit 3 | 262144 Hz |
| 10 | Bit 5 | 65536 Hz |
| 11 | Bit 7 | 16384 Hz |

### Falling Edge Detection
TIMA increments on a **falling edge** of the selected DIV bit.

**Important**: Writing to DIV can trigger a falling edge and increment TIMA!

## Implementation

### Timer State
```go
type Timer struct {
    divCounter uint16  // 16-bit internal counter
    tima       uint8   // Timer counter
    tma        uint8   // Timer modulo
    tac        uint8   // Timer control

    enabled    bool
    clockSelect uint8
}
```

### Update Timer
```go
func (t *Timer) Update(cycles int) {
    // Update DIV (always)
    oldDiv := t.divCounter
    t.divCounter += uint16(cycles)

    // Update TIMA if enabled
    if t.enabled {
        // Determine which bit to check based on clock select
        var bitPosition uint16
        switch t.clockSelect {
        case 0: bitPosition = 9   // 4096 Hz
        case 1: bitPosition = 3   // 262144 Hz
        case 2: bitPosition = 5   // 65536 Hz
        case 3: bitPosition = 7   // 16384 Hz
        }

        mask := uint16(1 << bitPosition)

        // Check for falling edge
        oldBit := (oldDiv & mask) != 0
        newBit := (t.divCounter & mask) != 0

        if oldBit && !newBit {
            t.incrementTIMA()
        }
    }
}

func (t *Timer) incrementTIMA() {
    t.tima++

    if t.tima == 0 {
        // Overflow occurred
        t.tima = t.tma
        requestInterrupt(TIMER_INTERRUPT)
    }
}
```

### Reading Registers
```go
func (t *Timer) Read(addr uint16) uint8 {
    switch addr {
    case 0xFF04:
        return uint8(t.divCounter >> 8)
    case 0xFF05:
        return t.tima
    case 0xFF06:
        return t.tma
    case 0xFF07:
        return t.tac | 0xF8  // Upper bits read as 1
    }
    return 0xFF
}
```

### Writing Registers
```go
func (t *Timer) Write(addr uint16, value uint8) {
    switch addr {
    case 0xFF04:
        // Any write resets DIV
        t.divCounter = 0

    case 0xFF05:
        t.tima = value

    case 0xFF06:
        t.tma = value

    case 0xFF07:
        t.tac = value & 0x07  // Only lower 3 bits writable
        t.enabled = (value & 0x04) != 0
        t.clockSelect = value & 0x03
    }
}
```

## Edge Cases and Quirks

### DIV Write Side Effect
Writing to DIV resets the internal counter, which can cause:
- Falling edge on the selected timer bit
- Unexpected TIMA increment

### TAC Change Side Effect
Changing TAC (enable or clock select) can cause:
- Falling edge detection
- Immediate TIMA increment

### TIMA Write During Overflow
If TIMA is written during the 4-cycle overflow period:
- Write can prevent the TMA reload
- Behavior is complex and cycle-dependent
- Most emulators don't implement this perfectly

### Interrupt Delay
Timer interrupt is requested the same M-cycle as overflow:
- Interrupt may be serviced 1-4 M-cycles later
- Depends on current CPU instruction

## Timer Usage Examples

### Fixed Rate Interrupt (60 Hz)
```assembly
; Setup timer for ~60 Hz interrupt
ld a, 0
ldh (DIV), a     ; Reset DIV

ld a, 6          ; TMA = 6
ldh (TMA), a     ; Overflow every 250 counts

ld a, 5          ; TAC = enable + 4096 Hz
ldh (TAC), a     ; 4096 / 250 ≈ 16.4 interrupts/sec

; Enable timer interrupt
ld a, $04
ldh (IE), a
ei
```

### Delay Using DIV
```assembly
; Wait approximately 1 second
WaitOneSecond:
    ldh a, (DIV)
    ld b, a
.loop:
    ldh a, (DIV)
    sub b
    cp 60          ; DIV increments 16384 times/sec
    jr c, .loop    ; Wait until 60 increments
    ret
```

## Testing

### Test Cases
1. DIV increments at correct rate
2. DIV reset on write
3. TIMA increments at each frequency
4. TIMA overflow and TMA reload
5. Timer interrupt triggered on overflow
6. Timer enable/disable
7. TAC frequency changes
8. Edge cases (DIV write, TAC changes)

### Test ROMs
- **Mooneye-GB**: timer tests
- **Blargg's timer tests**: Comprehensive timing tests

## Common Pitfalls

- Not implementing DIV as upper 8 bits of 16-bit counter
- Incorrect timer frequencies
- Not using falling edge detection
- Missing DIV write side effects
- Wrong overflow behavior (TMA reload timing)
- Not requesting interrupt on overflow
- Forgetting that DIV always runs
- Incorrect bit masking for TAC register
- Not handling timer disable correctly

## Performance Optimization

### Cycle Batching
Instead of updating every cycle:
1. Track cycles since last update
2. Batch update DIV and timer
3. Handle multiple increments/overflows

### Lookup Tables
Pre-calculate:
- DIV bit positions for each clock select
- Increment thresholds
- Overflow behaviors

## References
- Pan Docs Timer: https://gbdev.io/pandocs/Timer_and_Divider_Registers.html
- Pan Docs Timer Obscure Behaviour: https://gbdev.io/pandocs/Timer_Obscure_Behaviour.html
