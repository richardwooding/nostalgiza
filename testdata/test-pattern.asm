; Simple Game Boy Test Pattern ROM
; Displays a basic checkerboard pattern to verify PPU rendering

SECTION "ROM", ROM0[$100]
    nop
    jp Start

    ; Padding to reach header
    ds $134 - @

    ; Header Data ($134-$14F)
    db "TEST PATTERN"              ; Title (12 bytes)
    ds 4                            ; Padding to 16 bytes
    db 0                            ; CGB flag
    db 0, 0                         ; New licensee code
    db 0                            ; SGB flag
    db 0                            ; Cartridge type (ROM only)
    db 0                            ; ROM size (32KB)
    db 0                            ; RAM size (None)
    db 1                            ; Destination code (Non-Japanese)
    db $33                          ; Old licensee code
    db 0                            ; Mask ROM version
    db 0                            ; Header checksum (will be fixed by rgbfix)
    dw 0                            ; Global checksum
Start:
    ; Disable interrupts
    di

    ; Turn off LCD
    call WaitVBlank
    xor a
    ld [rLCDC], a

    ; Clear VRAM
    ld hl, $8000
    ld bc, $2000
.clearVRAM:
    xor a
    ld [hl+], a
    dec bc
    ld a, b
    or c
    jr nz, .clearVRAM

    ; Load tile data (create checkerboard tiles)
    call LoadTileData

    ; Fill tilemap with alternating tiles
    call FillTilemap

    ; Set palette (dark to light)
    ld a, %11100100
    ld [rBGP], a

    ; Turn on LCD with BG enabled
    ld a, LCDCF_ON | LCDCF_BGON
    ld [rLCDC], a

    ; Enable VBlank interrupt
    ld a, IEF_VBLANK
    ld [rIE], a
    ei

    ; Main loop - just halt
.loop:
    halt
    jr .loop

; Wait for VBlank
WaitVBlank:
    ld a, [rLY]
    cp 144
    jr c, WaitVBlank
    ret

; Load two tiles: one solid, one checkered
LoadTileData:
    ; Tile 0: Solid white (color 0)
    ld hl, $8000
    ld b, 16
.tile0:
    xor a
    ld [hl+], a
    dec b
    jr nz, .tile0

    ; Tile 1: Checkered pattern
    ld hl, $8010
    ld de, CheckerData
    ld b, 16
.tile1:
    ld a, [de]
    inc de
    ld [hl+], a
    dec b
    jr nz, .tile1

    ; Tile 2: Solid dark (color 3)
    ld hl, $8020
    ld b, 16
.tile2:
    ld a, $FF
    ld [hl+], a
    dec b
    jr nz, .tile2

    ret

; Fill tilemap with checkerboard pattern
FillTilemap:
    ld hl, $9800      ; Start of tilemap
    ld d, 18          ; 18 rows (visible screen)

.rowLoop:
    ld e, 20          ; 20 tiles per row (visible width)

.colLoop:
    ; Calculate tile index based on position
    ld a, d
    and 1             ; Check if row is odd/even
    ld b, a
    ld a, e
    and 1             ; Check if column is odd/even
    xor b             ; XOR to create checkerboard

    ; Use tile 0 or 2 based on checkerboard pattern
    jr z, .useTile0
    ld a, 2           ; Dark tile
    jr .writeTile
.useTile0:
    xor a             ; Light tile
.writeTile:
    ld [hl+], a

    dec e
    jr nz, .colLoop

    ; Move to next row (skip remaining columns)
    ld bc, 12         ; 32 - 20 = 12 tiles to skip
    add hl, bc

    dec d
    jr nz, .rowLoop

    ret

; Tile data for checkered pattern (tile 1)
CheckerData:
    db %10101010, %10101010
    db %01010101, %01010101
    db %10101010, %10101010
    db %01010101, %01010101
    db %10101010, %10101010
    db %01010101, %01010101
    db %10101010, %10101010
    db %01010101, %01010101

; Hardware register definitions
DEF rLCDC EQU $FF40
DEF rSTAT EQU $FF41
DEF rLY   EQU $FF44
DEF rBGP  EQU $FF47
DEF rIE   EQU $FFFF

; LCDC flags
DEF LCDCF_ON    EQU %10000000
DEF LCDCF_BGON  EQU %00000001

; Interrupt flags
DEF IEF_VBLANK EQU %00000001
