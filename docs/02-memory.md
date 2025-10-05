# Memory Map

## Overview
The Game Boy has a 16-bit address bus, providing access to 64 KiB of address space. This space is divided into different regions for ROM, RAM, I/O registers, and other purposes.

## Memory Layout

| Address Range | Size      | Description |
|---------------|-----------|-------------|
| 0000-3FFF     | 16 KiB    | ROM Bank 00 (fixed) |
| 4000-7FFF     | 16 KiB    | ROM Bank 01-NN (switchable via MBC) |
| 8000-9FFF     | 8 KiB     | Video RAM (VRAM) |
| A000-BFFF     | 8 KiB     | External RAM (cartridge RAM, if present) |
| C000-CFFF     | 4 KiB     | Work RAM Bank 0 (WRAM) |
| D000-DFFF     | 4 KiB     | Work RAM Bank 1 (WRAM) |
| E000-FDFF     | 7.5 KiB   | Echo RAM (mirror of C000-DDFF) |
| FE00-FE9F     | 160 bytes | Object Attribute Memory (OAM) |
| FEA0-FEFF     | 96 bytes  | Not Usable (prohibited) |
| FF00-FF7F     | 128 bytes | I/O Registers |
| FF80-FFFE     | 127 bytes | High RAM (HRAM) |
| FFFF          | 1 byte    | Interrupt Enable Register (IE) |

## Memory Regions Detail

### ROM Bank 00 (0000-3FFF)
- **Fixed**: Always maps to the first 16 KiB of the cartridge ROM
- Contains:
  - Interrupt vectors (0000-00FF)
  - Cartridge header (0100-014F)
  - Program entry point (typically starts at 0100 or 0150)
- **Access**: Read-only

### ROM Bank 01-NN (4000-7FFF)
- **Switchable**: Bank number controlled by Memory Bank Controller (MBC)
- Allows access to larger ROMs by swapping banks
- **Access**: Read-only
- Bank switching is done by writing to specific ROM addresses (see Cartridges doc)

### Video RAM - VRAM (8000-9FFF)
- **8 KiB** of video memory
- Contains:
  - Tile data (8000-97FF)
  - Background tile maps (9800-9BFF, 9C00-9FFF)
- **Access Restrictions**:
  - CPU cannot access during PPU mode 3 (drawing)
  - Reads return $FF when inaccessible
  - Writes are ignored when inaccessible

### External RAM (A000-BFFF)
- **8 KiB** window into cartridge RAM (if present)
- Size and banking controlled by MBC
- Often battery-backed for save games
- **Access**: Read/write when enabled via MBC
- Must be explicitly enabled by writing to MBC registers

### Work RAM - WRAM (C000-DFFF)
- **8 KiB** of general-purpose RAM
- C000-CFFF: Bank 0 (always accessible)
- D000-DFFF: Bank 1 (switchable on Game Boy Color)
- **Access**: Always accessible for read/write

### Echo RAM (E000-FDFF)
- **Mirror** of WRAM (C000-DDFF)
- Reading/writing affects the corresponding WRAM address
- **Nintendo's guidance**: "Prohibited" - should not be used
- Some games use it anyway, so emulators should support it

### Object Attribute Memory - OAM (FE00-FE9F)
- **160 bytes** for sprite attributes (40 sprites × 4 bytes each)
- Each sprite entry:
  - Byte 0: Y position
  - Byte 1: X position
  - Byte 2: Tile index
  - Byte 3: Attributes (palette, flip, priority)
- **Access Restrictions**:
  - Inaccessible during PPU mode 2 (OAM scan) and mode 3 (drawing)
  - Can be quickly filled using DMA transfer

### Not Usable (FEA0-FEFF)
- **96 bytes** of unusable memory
- Behavior is undefined
- Should return $00 or $FF on reads

### I/O Registers (FF00-FF7F)
- **128 bytes** of memory-mapped hardware registers
- Controls all Game Boy hardware:
  - Joypad (FF00)
  - Serial transfer (FF01-FF02)
  - Timer (FF04-FF07)
  - Audio (FF10-FF26, FF30-FF3F)
  - LCD/PPU (FF40-FF4B)
  - DMA transfer (FF46)
  - Other control registers
- See specific hardware documentation for details

### High RAM - HRAM (FF80-FFFE)
- **127 bytes** of fast RAM
- Always accessible (not affected by VRAM/OAM restrictions)
- Commonly used for:
  - Critical interrupt handlers
  - DMA transfer routine
  - Time-sensitive code
- Slightly faster access than WRAM

### Interrupt Enable Register (FFFF)
- **1 byte** register controlling interrupt enable flags
- See Interrupts documentation for details

## Memory Access Timing

### Standard Access
- Most memory reads/writes take 1 M-cycle (4 clock cycles)

### Restricted Access
- **VRAM**: Inaccessible during PPU mode 3
- **OAM**: Inaccessible during PPU modes 2 and 3
- When inaccessible:
  - Reads return $FF
  - Writes are ignored

## DMA Transfer

### OAM DMA (Direct Memory Access)
- Fast transfer of 160 bytes to OAM
- Triggered by writing to register FF46
- Source: XX00-XX9F (where XX is the value written)
- Destination: FE00-FE9F
- **Duration**: 160 M-cycles
- **During DMA**: Only HRAM is accessible
  - DMA routine must run from HRAM
  - Typically copies a small routine to HRAM first

### Example DMA Routine (to be placed in HRAM)
```assembly
; Write to FF46 to start DMA
ld a, HIGH(source)  ; Source address high byte
ldh (FF46), a       ; Start DMA

; Wait for DMA to complete (160 cycles)
ld a, 40            ; Delay loop
wait:
dec a
jr nz, wait
ret
```

## Memory Management Implementation

### Read Operation
1. Check address range
2. Check access restrictions (PPU mode for VRAM/OAM)
3. Route to appropriate memory/handler:
   - ROM → Cartridge
   - VRAM → PPU memory
   - External RAM → Cartridge RAM
   - WRAM → Work RAM
   - Echo RAM → Mirror to WRAM
   - OAM → PPU OAM
   - I/O → Hardware registers
   - HRAM → High RAM

### Write Operation
1. Check address range
2. Check access restrictions
3. Handle special cases:
   - ROM writes → MBC control
   - I/O register writes → Hardware updates
   - DMA register → Trigger DMA
4. Route to appropriate memory

### Memory Mapping Considerations
- Use arrays/slices for each region
- ROM banks: Array of bank pointers
- RAM banks: Similar to ROM banks
- I/O registers: Use handlers for read/write side effects
- Echo RAM: Redirect to WRAM addresses

## Implementation Tips

### Memory Interface
```go
type Memory interface {
    Read(addr uint16) uint8
    Write(addr uint16, value uint8)
}
```

### Banking
- Track current ROM bank (default: 1)
- Track current RAM bank (default: 0)
- Handle MBC-specific banking logic

### Access Control
- Check PPU mode before allowing VRAM/OAM access
- Return $FF for inaccessible reads
- Ignore inaccessible writes

### I/O Registers
- Implement handlers for each register
- Some registers are read-only, write-only, or read/write
- Some bits within registers may be read-only
- Handle side effects (e.g., writing to DIV resets it to 0)

## Common Pitfalls
- Not implementing Echo RAM mirroring
- Not restricting VRAM/OAM access during PPU modes
- Not handling MBC writes to ROM addresses
- Incorrect DMA implementation
- Not handling unused bits in I/O registers correctly
- Forgetting that external RAM must be enabled via MBC
