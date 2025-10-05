# Boot Sequence and Power-Up State

## Overview
When powered on, the Game Boy executes a boot ROM that:
1. Initializes hardware
2. Displays the Nintendo logo
3. Plays the startup sound
4. Validates the cartridge
5. Transfers control to the game

## Boot ROM

### Boot Process
The Game Boy does **not** start at $0100 as commonly believed. It starts at **$0000** where the boot ROM is initially mapped.

### Boot ROM Mapping
- **At power-on**: Boot ROM mapped to $0000-$00FF
- **After boot**: Boot ROM unmapped, cartridge ROM visible at $0000-$00FF

### Boot ROM Size
- DMG/MGB (Original/Pocket): 256 bytes
- CGB/AGB (Color/Advance): 2048 bytes

## DMG/MGB Boot Sequence

### Steps
1. **Initialize stack pointer** to $FFFE
2. **Clear VRAM** ($8000-$9FFF)
3. **Initialize audio** (setup registers)
4. **Load Nintendo logo** from cartridge header ($0104-$0133)
5. **Decode logo** to tile data
6. **Setup tilemap** for logo display
7. **Scroll logo down** with animation
8. **Play "ba-ding" sound**
9. **Verify logo** matches expected bitmap
10. **Verify header checksum** at $014D
11. **Unmap boot ROM**
12. **Jump to $0100** (game entry point)

### Logo Verification
The boot ROM contains the Nintendo logo bitmap and compares it with the cartridge header:
- If mismatch: **Lock up** (infinite loop)
- If match: Continue boot

This was a legal strategy - every Game Boy game must include Nintendo's trademarked logo, which the boot ROM verifies.

### Header Checksum
Checksum calculation (bytes $0134-$014C):
```
checksum = 0
for addr in $0134..$014C:
    checksum = checksum - byte[addr] - 1
checksum = checksum & $FF
```

If checksum at $014D doesn't match: **Lock up**

## CGB Boot Sequence

More complex than DMG:
1. **Initialize hardware**
2. **Load and display logo**
3. **Allow palette selection** during logo animation
   - Hold directional buttons for different color schemes
4. **Determine compatibility mode**:
   - CGB mode (Color Game Boy)
   - DMG mode (original Game Boy compatibility)
5. **Configure hardware** based on mode
6. **Verify logo and checksum**
7. **Unmap boot ROM**
8. **Jump to $0100**

## Power-Up Register State

### DMG (Original Game Boy)

| Register | Value | Notes |
|----------|-------|-------|
| AF | $01B0 | A=$01 (DMG), F=$B0 |
| BC | $0013 | |
| DE | $00D8 | |
| HL | $014D | Points to header checksum |
| SP | $FFFE | Stack pointer |
| PC | $0100 | Entry point |

### CGB (Game Boy Color)

| Register | Value | Notes |
|----------|-------|-------|
| AF | $1180 | A=$11 (CGB), F=$80 |
| BC | $0000 | |
| DE | $FF56 | |
| HL | $000D | |
| SP | $FFFE | Stack pointer |
| PC | $0100 | Entry point |

Register A indicates the hardware model:
- $01: DMG
- $FF: MGB (Game Boy Pocket)
- $11: CGB (Game Boy Color)

### Memory State

**VRAM:**
- Cleared to $00 (DMG)
- Logo tiles loaded at specific addresses

**I/O Registers:**
Values vary by model, important ones:

| Address | DMG | CGB | Description |
|---------|-----|-----|-------------|
| FF00 (P1) | $CF | $CF | Joypad (no buttons) |
| FF04 (DIV) | $AB | $AB | Divider |
| FF05 (TIMA) | $00 | $00 | Timer counter |
| FF06 (TMA) | $00 | $00 | Timer modulo |
| FF07 (TAC) | $00 | $00 | Timer control |
| FF0F (IF) | $E1 | $E1 | Interrupt flags |
| FF10-FF26 | Audio | Audio | Sound registers |
| FF40 (LCDC) | $91 | $91 | LCD control |
| FF42 (SCY) | $00 | $00 | Scroll Y |
| FF43 (SCX) | $00 | $00 | Scroll X |
| FF44 (LY) | $00 | $00 | LCD Y coord |
| FF45 (LYC) | $00 | $00 | LY compare |
| FF47 (BGP) | $FC | $FC | BG palette |
| FF48 (OBP0) | $FF | $FF | OBJ palette 0 |
| FF49 (OBP1) | $FF | $FF | OBJ palette 1 |
| FF4A (WY) | $00 | $00 | Window Y |
| FF4B (WX) | $00 | $00 | Window X |
| FFFF (IE) | $00 | $00 | Interrupt enable |

## Skipping Boot ROM

Most emulators skip the boot ROM and jump directly to $0100:

### Why Skip?
- Faster startup
- Don't need boot ROM file
- Logo display not essential for gameplay

### Requirements
1. Set registers to post-boot values
2. Initialize memory state
3. Setup I/O registers
4. Start at PC = $0100

### Implementation
```go
func (cpu *CPU) SkipBootROM() {
    // Set registers (DMG)
    cpu.a = 0x01
    cpu.f = 0xB0
    cpu.b = 0x00
    cpu.c = 0x13
    cpu.d = 0x00
    cpu.e = 0xD8
    cpu.h = 0x01
    cpu.l = 0x4D
    cpu.sp = 0xFFFE
    cpu.pc = 0x0100

    // Initialize I/O registers
    cpu.mem.Write(0xFF00, 0xCF)  // P1
    cpu.mem.Write(0xFF04, 0xAB)  // DIV
    cpu.mem.Write(0xFF05, 0x00)  // TIMA
    cpu.mem.Write(0xFF06, 0x00)  // TMA
    cpu.mem.Write(0xFF07, 0x00)  // TAC
    cpu.mem.Write(0xFF0F, 0xE1)  // IF
    cpu.mem.Write(0xFF40, 0x91)  // LCDC
    cpu.mem.Write(0xFF42, 0x00)  // SCY
    cpu.mem.Write(0xFF43, 0x00)  // SCX
    cpu.mem.Write(0xFF45, 0x00)  // LYC
    cpu.mem.Write(0xFF47, 0xFC)  // BGP
    cpu.mem.Write(0xFF48, 0xFF)  // OBP0
    cpu.mem.Write(0xFF49, 0xFF)  // OBP1
    cpu.mem.Write(0xFF4A, 0x00)  // WY
    cpu.mem.Write(0xFF4B, 0x00)  // WX
    cpu.mem.Write(0xFFFF, 0x00)  // IE

    // Initialize audio registers (NR52, etc.)
    cpu.mem.Write(0xFF26, 0xF1)  // Audio on, all channels
    // ... other audio registers
}
```

## Boot ROM Emulation

### Full Emulation
For accuracy, some emulators include the actual boot ROM:

**Advantages:**
- Authentic startup experience
- Perfect accuracy
- Displays Nintendo logo and plays sound

**Disadvantages:**
- Requires boot ROM file (copyright concerns)
- Slower startup
- More complex to implement

### Boot ROM Sources
Official boot ROMs are copyrighted by Nintendo. Options:
1. Extract from real hardware (legal if you own it)
2. Use open-source replacements
3. Skip entirely (most common)

### Boot ROM Detection
Games shouldn't rely on boot ROM behavior, but some do:
- Check register values to detect hardware model
- Rely on specific memory states
- Timing-dependent code

## Entry Point ($0100-$0103)

The first 4 bytes of the cartridge typically contain:
```assembly
nop          ; $00 - Required by boot ROM
jp $0150     ; $C3 $50 $01 - Jump to actual start
```

The NOP at $0100 is verified by the boot ROM on some models.

## Implementation Considerations

### For Testing
Start without boot ROM:
1. Simpler implementation
2. Faster iteration
3. Focus on game logic first

### For Accuracy
Add boot ROM later:
1. Implement boot ROM loading
2. Handle unmapping at correct time
3. Verify logo and checksum
4. Animate logo scroll

### Model Detection
Games check register A to determine hardware:
```assembly
cp $01       ; DMG?
jr z, .dmg
cp $11       ; CGB?
jr z, .cgb
; ... handle other models
```

## Testing

### Without Boot ROM
1. Set registers correctly
2. Verify I/O register values
3. Test games that check hardware model

### With Boot ROM
1. Verify logo display
2. Check checksum validation
3. Test lock-up on invalid header
4. Verify correct register values after boot

## Common Pitfalls

- Incorrect register values after skip
- Missing I/O register initialization
- Wrong hardware model in register A
- Not handling boot ROM unmapping
- Incorrect LCDC value (screen off vs on)
- Missing audio register initialization
- Wrong stack pointer value
- Not starting at PC=$0100

## Boot ROM Unmapping

### Unmapping Register
On DMG, writing to $FF50 unmaps the boot ROM:
```assembly
ld a, $01
ldh ($50), a    ; Unmap boot ROM
```

After this:
- Boot ROM no longer accessible
- Cartridge ROM visible at $0000-$00FF
- **Cannot be remapped** (one-way operation)

### Implementation
```go
type Memory struct {
    bootROMEnabled bool
    bootROM []byte
    // ...
}

func (m *Memory) Read(addr uint16) uint8 {
    if addr < 0x0100 && m.bootROMEnabled {
        return m.bootROM[addr]
    }
    // ... normal cartridge read
}

func (m *Memory) Write(addr uint16, value uint8) {
    if addr == 0xFF50 && value != 0 {
        m.bootROMEnabled = false  // Unmap boot ROM
    }
    // ... other write handling
}
```

## References
- Pan Docs Power Up Sequence: https://gbdev.io/pandocs/Power_Up_Sequence.html
- Boot ROM Disassembly: https://gbdev.io/gb-asm-tutorial/part1/boot.html
- Hardware Models: https://gbdev.io/pandocs/Power_Up_Sequence.html#cpu-registers
