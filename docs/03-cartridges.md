# Cartridges and Memory Bank Controllers

## Cartridge Overview
Game Boy cartridges contain:
- ROM (program code and assets)
- Optional RAM (for save data)
- Optional battery (for persistent saves)
- Memory Bank Controller (MBC) chip (for larger games)
- Optional additional hardware (RTC, rumble, etc.)

## Cartridge Header

The cartridge header occupies addresses **$0100-$014F** and contains metadata about the game.

### Header Structure

| Address | Size | Name | Description |
|---------|------|------|-------------|
| 0100-0103 | 4 bytes | Entry Point | Initial jump instruction (usually `JP $0150`) |
| 0104-0133 | 48 bytes | Nintendo Logo | Required bitmap checked by boot ROM |
| 0134-0143 | 16 bytes | Title | Game title (ASCII, uppercase) |
| 013F-0142 | 4 bytes | Manufacturer Code | Publisher identifier (overlaps with title) |
| 0143 | 1 byte | CGB Flag | Game Boy Color compatibility |
| 0144-0145 | 2 bytes | New Licensee Code | Two-character publisher code |
| 0146 | 1 byte | SGB Flag | Super Game Boy features ($00 or $03) |
| 0147 | 1 byte | Cartridge Type | MBC type and features |
| 0148 | 1 byte | ROM Size | Total ROM size |
| 0149 | 1 byte | RAM Size | External RAM size |
| 014A | 1 byte | Destination Code | $00=Japan, $01=Overseas |
| 014B | 1 byte | Old Licensee Code | $33=Use new licensee code |
| 014C | 1 byte | Version Number | Game version (usually $00) |
| 014D | 1 byte | Header Checksum | Checksum of bytes $0134-$014C |
| 014E-014F | 2 bytes | Global Checksum | Checksum of entire ROM (not verified) |

### Cartridge Type ($0147)

| Value | Type | Description |
|-------|------|-------------|
| $00 | ROM ONLY | No MBC, 32 KiB ROM max |
| $01 | MBC1 | Basic banking |
| $02 | MBC1+RAM | With external RAM |
| $03 | MBC1+RAM+BATTERY | With battery-backed RAM |
| $05 | MBC2 | Built-in RAM |
| $06 | MBC2+BATTERY | With battery |
| $08 | ROM+RAM | 32 KiB ROM with RAM |
| $09 | ROM+RAM+BATTERY | With battery |
| $0F | MBC3+TIMER+BATTERY | With RTC |
| $10 | MBC3+TIMER+RAM+BATTERY | With RTC and RAM |
| $11 | MBC3 | Basic banking |
| $12 | MBC3+RAM | With RAM |
| $13 | MBC3+RAM+BATTERY | With battery-backed RAM |
| $19 | MBC5 | Advanced banking |
| $1A | MBC5+RAM | With RAM |
| $1B | MBC5+RAM+BATTERY | With battery-backed RAM |
| $1C | MBC5+RUMBLE | With rumble motor |
| $1D | MBC5+RUMBLE+RAM | With rumble and RAM |
| $1E | MBC5+RUMBLE+RAM+BATTERY | Full features |

### ROM Size ($0148)

| Value | Size | Banks |
|-------|------|-------|
| $00 | 32 KiB | 2 (no banking) |
| $01 | 64 KiB | 4 |
| $02 | 128 KiB | 8 |
| $03 | 256 KiB | 16 |
| $04 | 512 KiB | 32 |
| $05 | 1 MiB | 64 |
| $06 | 2 MiB | 128 |
| $07 | 4 MiB | 256 |
| $08 | 8 MiB | 512 |

Formula: Banks = 2 << value

### RAM Size ($0149)

| Value | Size | Banks |
|-------|------|-------|
| $00 | None | 0 |
| $01 | Unused | - |
| $02 | 8 KiB | 1 |
| $03 | 32 KiB | 4 (banks of 8 KiB) |
| $04 | 128 KiB | 16 |
| $05 | 64 KiB | 8 |

### Header Checksum ($014D)

Validates bytes $0134-$014C:
```
checksum = 0
for each byte in $0134-$014C:
    checksum = checksum - byte - 1
checksum = checksum & $FF
```

The boot ROM verifies this checksum and won't boot if incorrect.

## Memory Bank Controllers (MBCs)

MBCs are chips in the cartridge that enable bank switching, expanding the Game Boy's 16-bit address space.

### No MBC (ROM Only)

**Specifications:**
- Max 32 KiB ROM (addresses $0000-$7FFF)
- Optional 8 KiB RAM (addresses $A000-$BFFF)
- No banking

**Implementation:**
- Simple direct mapping
- No special write handling needed

### MBC1

The most common MBC, found in early Game Boy games.

**Specifications:**
- Max 2 MiB ROM (125 banks of 16 KiB)
- Max 32 KiB RAM (4 banks of 8 KiB)
- Two banking modes

**Register Map:**

| Address Range | Function |
|---------------|----------|
| 0000-1FFF | RAM Enable (write $0A to enable, $00 to disable) |
| 2000-3FFF | ROM Bank Number (lower 5 bits) |
| 4000-5FFF | RAM Bank Number / ROM Bank Number (upper 2 bits) |
| 6000-7FFF | Banking Mode Select |

**Banking Modes:**
- Mode 0 (default): ROM banking mode
  - Full ROM addressing (up to 2 MiB)
  - Only RAM bank 0 accessible
- Mode 1: RAM banking mode
  - Limited ROM (up to 512 KiB)
  - All RAM banks accessible

**ROM Banking:**
- Bank 0 always mapped to $0000-$3FFF
- Banks 1-127 mapped to $4000-$7FFF
- Writing $00 to bank register selects bank $01
- Banks $20, $40, $60 cannot be selected (automatically increment to next)

**Implementation Notes:**
- RAM must be enabled before access
- Bank 0 is fixed in lower ROM area
- Handle special cases for banks $00, $20, $40, $60

### MBC2

**Specifications:**
- Max 256 KiB ROM (16 banks)
- Built-in 512 × 4 bits RAM (no external RAM)
- Simpler than MBC1

**Register Map:**

| Address Range | Function |
|---------------|----------|
| 0000-3FFF | RAM Enable (if bit 8 of address is 0) / ROM Bank (if bit 8 is 1) |

**RAM:**
- Only lower 4 bits of each byte are used
- Upper 4 bits read as 1s ($F0 | value)
- Only 512 bytes (not 8 KiB)

**Implementation Notes:**
- Check bit 8 of address to determine register
- Mask RAM values to 4 bits

### MBC3

**Specifications:**
- Max 2 MiB ROM (128 banks)
- Max 32 KiB RAM (4 banks)
- Optional Real-Time Clock (RTC)

**Register Map:**

| Address Range | Function |
|---------------|----------|
| 0000-1FFF | RAM and Timer Enable |
| 2000-3FFF | ROM Bank Number (7 bits, $01-$7F) |
| 4000-5FFF | RAM Bank Number ($00-$03) or RTC Register ($08-$0C) |
| 6000-7FFF | Latch Clock Data |

**RTC Registers:**
- $08: Seconds (0-59)
- $09: Minutes (0-59)
- $0A: Hours (0-23)
- $0B: Days (lower 8 bits)
- $0C: Days (upper 1 bit), Halt flag, Day carry

**Implementation Notes:**
- Writing $00 then $01 to $6000-$7FFF latches RTC values
- RTC requires tracking real time
- Bank 0 cannot be selected (auto-corrects to 1)

### MBC5

The most advanced MBC, required for Game Boy Color compatibility.

**Specifications:**
- Max 8 MiB ROM (512 banks)
- Max 128 KiB RAM (16 banks)
- Only MBC guaranteed to work in CGB Double Speed Mode

**Register Map:**

| Address Range | Function |
|---------------|----------|
| 0000-1FFF | RAM Enable |
| 2000-2FFF | ROM Bank Number (lower 8 bits) |
| 3000-3FFF | ROM Bank Number (9th bit) |
| 4000-5FFF | RAM Bank Number (4 bits) |

**ROM Banking:**
- 9-bit bank number (0-511)
- Bank 0 CAN be selected (unlike MBC1/3)
- No automatic bank correction

**Implementation Notes:**
- Combine two writes for 9-bit ROM bank
- No special case for bank 0
- Supports rumble in some cartridges (uses bit 3 of RAM bank register)

## Implementation Strategy

### Loading ROM
1. Read entire ROM file into memory
2. Parse header at $0100-$014F
3. Validate header checksum (optional, but recommended)
4. Determine MBC type from byte $0147
5. Determine ROM/RAM sizes from bytes $0148-$0149
6. Initialize appropriate MBC handler

### MBC Interface
```go
type MBC interface {
    ReadROM(addr uint16) uint8
    WriteROM(addr uint16, value uint8)
    ReadRAM(addr uint16) uint8
    WriteRAM(addr uint16, value uint8)
}
```

### Banking Logic
- Track current ROM bank (defaults vary by MBC)
- Track current RAM bank (default 0)
- Track RAM enable flag (default disabled)
- Handle writes to ROM addresses as register updates

### Save Data
- Persist RAM contents when cartridge has battery
- Common formats: .sav files (raw RAM dump)
- Save on RAM change or periodically

## Testing
- Test ROMs: Blargg's test suite includes MBC tests
- Verify banking behavior with multi-bank games
- Test save/load functionality
- Check edge cases (invalid banks, disabled RAM, etc.)

## Common Pitfalls
- Not handling RAM enable/disable
- Incorrect bank 0 handling (varies by MBC)
- Missing MBC1 special bank cases ($20, $40, $60)
- Not masking MBC2 RAM to 4 bits
- Forgetting MBC5's 9-bit bank number
- Not implementing battery-backed RAM persistence
