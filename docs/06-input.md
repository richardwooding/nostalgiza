# Joypad Input

## Overview
The Game Boy has 8 buttons organized as a 2×4 matrix. Input is read through the Joypad register (P1/JOYP) at address $FF00.

## Button Layout

### Physical Buttons
**Action Buttons:**
- A
- B
- Select
- Start

**Direction Buttons:**
- Up
- Down
- Left
- Right

## Joypad Register (P1/JOYP - $FF00)

The register is used to both select which button group to read and return the button states.

### Register Format

| Bit | Name | Description |
|-----|------|-------------|
| 7-6 | - | Not used (read as 1) |
| 5 | P15 | Select Action buttons (0=Select) |
| 4 | P14 | Select Direction buttons (0=Select) |
| 3 | P13 | Input: Down or Start (0=Pressed) |
| 2 | P12 | Input: Up or Select (0=Pressed) |
| 1 | P11 | Input: Left or B (0=Pressed) |
| 0 | P10 | Input: Right or A (0=Pressed) |

### Important Notes
- **Inverted logic**: 0 = pressed, 1 = not pressed
- **Select bits (4-5)**: Writing 0 selects that button group
- **Both groups**: Can be selected simultaneously (reads OR of both)
- **No selection**: When both bits 4-5 are 1, lower nibble reads as $F

## Button Mapping

### When P14 = 0 (Direction buttons selected)

| Bit | Button |
|-----|--------|
| 3 | Down |
| 2 | Up |
| 1 | Left |
| 0 | Right |

### When P15 = 0 (Action buttons selected)

| Bit | Button |
|-----|--------|
| 3 | Start |
| 2 | Select |
| 1 | B |
| 0 | A |

## Reading Input

### Typical Reading Sequence
1. Select button group (write to P1)
2. Short delay (read multiple times for stability)
3. Read button states (read from P1)
4. Repeat for other button group if needed

### Example Code
```assembly
; Read D-Pad
ld a, $20        ; Select direction buttons (P14=0)
ldh (P1), a      ; Write to joypad register
ldh a, (P1)      ; Read multiple times for stability
ldh a, (P1)      ; (hardware needs time to settle)
cpl              ; Invert bits (so 1=pressed)
and $0F          ; Mask lower 4 bits
ld b, a          ; Save direction state

; Read Buttons
ld a, $10        ; Select action buttons (P15=0)
ldh (P1), a
ldh a, (P1)      ; Read multiple times
ldh a, (P1)
cpl              ; Invert bits
and $0F          ; Mask lower 4 bits
or b             ; Combine with direction state

; Reset joypad
ld a, $30        ; Deselect all
ldh (P1), a
```

## Joypad Interrupt

### Triggering
- Interrupt requested when any button is pressed
- Specifically: when any input line goes from High (1) to Low (0)
- Bit 4 in IF register ($FF0F)

### Usage
- Wake Game Boy from HALT or STOP mode
- Detect button presses without polling

### Limitations
- Can be unreliable due to button bounce
- May trigger multiple times per press
- Most games poll the joypad instead

### Enabling
```assembly
ld a, $10        ; Enable joypad interrupt
ldh (IE), a      ; Write to interrupt enable register
ei               ; Enable interrupts globally
```

## Implementation

### State Tracking
```go
type Joypad struct {
    selectAction    bool  // P15 select
    selectDirection bool  // P14 select

    // Button states (true = pressed)
    buttonA      bool
    buttonB      bool
    buttonStart  bool
    buttonSelect bool
    buttonUp     bool
    buttonDown   bool
    buttonLeft   bool
    buttonRight  bool
}
```

### Reading P1 Register
```go
func (j *Joypad) Read() uint8 {
    result := uint8(0xFF)  // Default: all bits set

    if !j.selectAction {
        // Action buttons selected (P15=0)
        if j.buttonStart  { result &^= 0x08 }  // Bit 3
        if j.buttonSelect { result &^= 0x04 }  // Bit 2
        if j.buttonB      { result &^= 0x02 }  // Bit 1
        if j.buttonA      { result &^= 0x01 }  // Bit 0
    }

    if !j.selectDirection {
        // Direction buttons selected (P14=0)
        if j.buttonDown  { result &^= 0x08 }  // Bit 3
        if j.buttonUp    { result &^= 0x04 }  // Bit 2
        if j.buttonLeft  { result &^= 0x02 }  // Bit 1
        if j.buttonRight { result &^= 0x01 }  // Bit 0
    }

    // Set select bits based on selection
    if j.selectAction    { result |= 0x20 }
    if j.selectDirection { result |= 0x10 }

    // Upper 2 bits always 1
    result |= 0xC0

    return result
}
```

### Writing P1 Register
```go
func (j *Joypad) Write(value uint8) {
    // Only bits 4-5 are writable
    j.selectAction = (value & 0x20) != 0
    j.selectDirection = (value & 0x10) != 0
}
```

### Button Press Handler
```go
func (j *Joypad) PressButton(button string) {
    switch button {
    case "A":      j.buttonA = true
    case "B":      j.buttonB = true
    case "Start":  j.buttonStart = true
    case "Select": j.buttonSelect = true
    case "Up":     j.buttonUp = true
    case "Down":   j.buttonDown = true
    case "Left":   j.buttonLeft = true
    case "Right":  j.buttonRight = true
    }

    // Request joypad interrupt
    requestInterrupt(JOYPAD_INTERRUPT)
}
```

### Button Release Handler
```go
func (j *Joypad) ReleaseButton(button string) {
    switch button {
    case "A":      j.buttonA = false
    case "B":      j.buttonB = false
    case "Start":  j.buttonStart = false
    case "Select": j.buttonSelect = false
    case "Up":     j.buttonUp = false
    case "Down":   j.buttonDown = false
    case "Left":   j.buttonLeft = false
    case "Right":  j.buttonRight = false
    }
}
```

## Key Mapping

Map keyboard/controller inputs to Game Boy buttons:

### Common Mappings
| GB Button | Keyboard | Controller |
|-----------|----------|------------|
| A | Z or X | A or B |
| B | A or C | X or Y |
| Start | Enter | Start |
| Select | Shift | Select |
| Up | Arrow Up | D-Pad Up |
| Down | Arrow Down | D-Pad Down |
| Left | Arrow Left | D-Pad Left |
| Right | Arrow Right | D-Pad Right |

## Timing Considerations

### Read Stability
- Hardware requires 2-4 reads for stability
- First read may return incorrect values
- Games typically read 2-6 times in succession

### Polling Frequency
- Most games poll once per frame (during V-Blank)
- Some games poll more frequently for better responsiveness

### Button Bounce
- Physical buttons can "bounce" (rapid press/release)
- Games handle debouncing in software
- Emulator usually doesn't need to simulate bounce

## Simultaneous Presses

### Opposite Directions
- Hardware allows Up+Down or Left+Right simultaneously
- Real Game Boy d-pad makes this physically impossible
- Some games have glitches when this happens
- **Recommendation**: Block opposite direction combinations in emulator

```go
// Prevent opposite directions
if j.buttonUp && j.buttonDown {
    j.buttonUp = false
    j.buttonDown = false
}
if j.buttonLeft && j.buttonRight {
    j.buttonLeft = false
    j.buttonRight = false
}
```

## Testing

### Test Cases
1. Single button press
2. Multiple button presses
3. All buttons pressed simultaneously
4. Direction selection vs Action selection
5. No selection (bits 4-5 both set)
6. Both selections (bits 4-5 both clear)
7. Joypad interrupt triggering

### Visual Testing
- Display button states on screen
- Use a simple input test ROM
- Verify bit patterns match expected values

## Common Pitfalls

- Forgetting inverted logic (0=pressed, 1=not pressed)
- Not masking upper bits (6-7 should always be 1)
- Not implementing the read delay (multiple reads)
- Reading without selecting button group first
- Not handling simultaneous button presses
- Missing joypad interrupt implementation
- Not allowing opposite direction presses (or incorrectly allowing them)

## STOP Mode Wake-Up

When using STOP mode (low power):
- Joypad interrupt can wake the system
- Must enable joypad interrupt before STOP
- Button press will exit STOP and trigger interrupt

```assembly
ld a, $10        ; Enable joypad interrupt
ldh (IE), a
ei
stop             ; Enter low power mode
; ... resumes here after button press
```

## References
- Pan Docs Joypad Input: https://gbdev.io/pandocs/Joypad_Input.html
