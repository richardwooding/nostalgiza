# Graphics System (PPU)

## Overview
The Game Boy's Picture Processing Unit (PPU) renders graphics to a 160×144 pixel LCD display using a tile-based system. The PPU operates independently from the CPU, running in parallel and restricting certain memory accesses.

## Display Specifications
- **Resolution**: 160 × 144 pixels
- **Colors**: 4 shades of gray (2 bits per pixel)
- **Refresh Rate**: 59.73 Hz
- **Frame Time**: ~16.74 ms (70224 clock cycles)

## Tile System

### Tiles
- **Size**: 8×8 pixels
- **Format**: 2 bits per pixel (4 colors/shades)
- **Data Size**: 16 bytes per tile (2 bytes per row)
- **Encoding**: Each pixel row uses 2 bytes:
  - Byte 1: Lower bits of color indices
  - Byte 2: Upper bits of color indices
  - Color value = (bit from byte2 << 1) | bit from byte1

### Tile Data
Two addressing modes:

**Mode 1: $8000 Addressing (Unsigned)**
- Tiles stored at $8000-$8FFF
- Tile indices 0-255 (unsigned)
- Index 0 = $8000, Index 1 = $8010, etc.

**Mode 2: $8800 Addressing (Signed)**
- Tiles stored at $8800-$97FF
- Tile indices -128 to 127 (signed)
- Index 0 = $9000, Index -1 = $8FF0, Index 1 = $9010

### Tile Maps
Two 32×32 tile maps (background/window):
- **Map 0**: $9800-$9BFF
- **Map 1**: $9C00-$9FFF

Each map entry is 1 byte (tile index).

## Rendering Layers

The PPU renders three layers (back to front):

### 1. Background Layer
- Scrollable 256×256 pixel grid (32×32 tiles)
- Position controlled by SCX/SCY registers
- Wraps around at edges
- Can be disabled (appears white)

### 2. Window Layer
- Fixed rectangular overlay
- Position controlled by WX/WY registers
- No scrolling, no transparency
- Rendered over background
- Can be disabled

### 3. Object Layer (Sprites)
- Up to 40 sprites total
- Max 10 sprites per scanline
- Size: 8×8 or 8×16 pixels (switchable)
- Attributes: position, tile, palette, flip, priority
- Can appear behind or in front of background/window

## PPU Modes

The PPU cycles through 4 modes while rendering each frame:

| Mode | Name | Duration | Description |
|------|------|----------|-------------|
| 2 | OAM Scan | 80 dots | Searching for objects on current line |
| 3 | Drawing | 172-289 dots | Rendering pixels (variable length) |
| 0 | H-Blank | 87-204 dots | Horizontal blank (end of line) |
| 1 | V-Blank | 4560 dots | Vertical blank (10 lines) |

### Frame Structure
- **Visible Lines 0-143**: Modes 2 → 3 → 0 (each line)
- **Lines 144-153**: Mode 1 (V-Blank)
- **Total**: 154 lines × 456 dots = 70224 dots per frame

### Mode Timing (per scanline)
```
Mode 2 (OAM Scan):   80 dots
Mode 3 (Drawing):    172-289 dots (varies)
Mode 0 (H-Blank):    87-204 dots (remainder of 456)
```

### Memory Access Restrictions
- **Mode 2**: OAM inaccessible to CPU
- **Mode 3**: Both VRAM and OAM inaccessible to CPU
- **Modes 0, 1**: All video memory accessible

## LCD Control Register (LCDC - $FF40)

| Bit | Name | Description |
|-----|------|-------------|
| 7 | LCD Enable | 0=Off, 1=On |
| 6 | Window Tile Map | 0=$9800-$9BFF, 1=$9C00-$9FFF |
| 5 | Window Enable | 0=Off, 1=On |
| 4 | BG/Win Tile Data | 0=$8800-$97FF, 1=$8000-$8FFF |
| 3 | BG Tile Map | 0=$9800-$9BFF, 1=$9C00-$9FFF |
| 2 | OBJ Size | 0=8×8, 1=8×16 |
| 1 | OBJ Enable | 0=Off, 1=On |
| 0 | BG/Win Enable | 0=Off, 1=On |

## LCD Status Register (STAT - $FF41)

| Bit | Name | Description |
|-----|------|-------------|
| 6 | LYC Interrupt | Interrupt when LY=LYC |
| 5 | Mode 2 Interrupt | Interrupt on mode 2 (OAM) |
| 4 | Mode 1 Interrupt | Interrupt on mode 1 (V-Blank) |
| 3 | Mode 0 Interrupt | Interrupt on mode 0 (H-Blank) |
| 2 | LYC=LY Flag | 1 when LY equals LYC |
| 1-0 | Mode Flag | Current PPU mode (0-3) |

## PPU Registers

| Address | Name | Description |
|---------|------|-------------|
| FF40 | LCDC | LCD Control |
| FF41 | STAT | LCD Status |
| FF42 | SCY | Background Scroll Y |
| FF43 | SCX | Background Scroll X |
| FF44 | LY | Current Scanline (0-153) |
| FF45 | LYC | Scanline Compare |
| FF46 | DMA | OAM DMA Transfer |
| FF47 | BGP | Background Palette |
| FF48 | OBP0 | Object Palette 0 |
| FF49 | OBP1 | Object Palette 1 |
| FF4A | WY | Window Y Position |
| FF4B | WX | Window X Position + 7 |

## Palettes

Each palette maps color indices (0-3) to actual shades:

### Background Palette (BGP - $FF47)
- Bits 7-6: Color for index 3 (darkest)
- Bits 5-4: Color for index 2
- Bits 3-2: Color for index 1
- Bits 1-0: Color for index 0 (lightest)

### Object Palettes (OBP0, OBP1 - $FF48, $FF49)
- Same format as BGP
- Color index 0 is transparent for objects

### Color Values
- 0: White
- 1: Light gray
- 2: Dark gray
- 3: Black

## Object Attribute Memory (OAM)

Each sprite has 4 bytes in OAM:

| Byte | Description |
|------|-------------|
| 0 | Y Position (actual Y - 16) |
| 1 | X Position (actual X - 8) |
| 2 | Tile Index |
| 3 | Attributes/Flags |

### Object Attributes (Byte 3)

| Bit | Name | Description |
|-----|------|-------------|
| 7 | Priority | 0=Above BG, 1=Behind BG colors 1-3 |
| 6 | Y Flip | 0=Normal, 1=Flipped vertically |
| 5 | X Flip | 0=Normal, 1=Flipped horizontally |
| 4 | Palette | 0=OBP0, 1=OBP1 |
| 3-0 | - | Unused (CGB uses for bank/palette) |

### Object Priority
1. Smaller X coordinate has priority
2. If X is equal, earlier in OAM has priority
3. Max 10 objects per scanline (lowest priority dropped)

## Rendering Pipeline

### Per-Frame Process
1. **Mode 2** (OAM Scan): Find up to 10 objects on current scanline
2. **Mode 3** (Drawing): Render background, window, and objects
3. **Mode 0** (H-Blank): Wait until end of scanline
4. Repeat for lines 0-143
5. **Mode 1** (V-Blank): Lines 144-153, trigger V-Blank interrupt

### Per-Pixel Rendering (Mode 3)
1. Fetch background tile and pixel
2. Apply background scroll
3. If window is visible, fetch window pixel instead
4. Fetch object pixels for this position
5. Mix layers based on priority
6. Apply palette
7. Output pixel to LCD

### Background Rendering
1. Calculate tile map position from scroll and pixel coordinates
2. Fetch tile index from tile map
3. Fetch tile data from VRAM
4. Extract pixel color index
5. Apply BGP palette

### Object Rendering
1. Check which objects overlap current pixel
2. For each object (up to 10):
   - Check X position
   - Fetch tile data
   - Apply flip attributes
   - Extract pixel color index
3. Select highest priority non-transparent pixel
4. Apply OBP0/OBP1 palette
5. Mix with background based on priority flag

## Implementation Strategy

### PPU State
```go
type PPU struct {
    mode int        // Current PPU mode (0-3)
    dots int        // Dot counter for current scanline
    ly int          // Current scanline (0-153)

    vram [8192]uint8
    oam [160]uint8

    // Registers
    lcdc, stat, scy, scx uint8
    lyc uint8
    bgp, obp0, obp1 uint8
    wy, wx uint8

    framebuffer [160][144]uint8
}
```

### Update Cycle
```
1. Advance dot counter by CPU cycles elapsed
2. Check if mode should change
3. Update STAT register
4. Trigger interrupts if enabled
5. If in mode 3, render pixels
```

### Mode Transitions
```
Mode 2 (OAM Scan) → Mode 3 (Drawing) → Mode 0 (H-Blank)
                                             ↓
When LY = 144: → Mode 1 (V-Blank)
When LY = 0:   ← Back to Mode 2
```

## V-Blank Interrupt

The V-Blank interrupt occurs when the PPU enters mode 1:
- **When**: After line 143 is complete
- **Duration**: Lines 144-153 (10 scanlines)
- **Purpose**: Safe time to update VRAM/OAM and perform game logic
- **Typical use**: Game loop synchronization

## Common Effects

### Screen Shake
Modify SCX/SCY during V-Blank

### Parallax Scrolling
Change SCX/SCY mid-frame using STAT interrupts

### Window Scanline Effects
Enable/disable window per scanline for split-screen effects

## Implementation Tips

### Cycle Accuracy
- Track dots (T-cycles / 4)
- Update PPU state every CPU instruction
- Mode 3 length varies based on rendering complexity

### Optimization
- Pre-calculate tile addresses
- Cache palette mappings
- Only render visible pixels
- Use lookup tables for bit operations

### Testing
- Display tile data viewer
- Background/window toggle
- Sprite viewer
- VRAM viewer

## Common Pitfalls
- Not restricting VRAM/OAM access during modes 2/3
- Incorrect window position (WX is offset by 7)
- Wrong sprite Y/X offsets (Y-16, X-8)
- Missing sprite priority handling
- Not handling mode 3 variable timing
- Incorrect tile data decoding
- Wrong palette application
- Missing V-Blank interrupt

## References
- Pan Docs Graphics: https://gbdev.io/pandocs/Graphics.html
- Pan Docs Rendering: https://gbdev.io/pandocs/Rendering.html
- Pan Docs LCDC: https://gbdev.io/pandocs/LCDC.html
