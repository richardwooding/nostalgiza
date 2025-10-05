; Simpler Game Boy Test Pattern ROM (no HALT, just infinite loop)
; Displays a basic checkerboard pattern to verify PPU rendering

SECTION "ROM", ROM0[$100]
    nop
    jp Start

    ; Padding to reach header
    ds $134 - @

    ; Header Data ($134-$14F)
    db "SIMPLE TEST"                ; Title (11 bytes)
    ds 5                             ; Padding to 16 bytes
    db 0                             ; CGB flag
    db 0, 0                          ; New licensee code
    db 0                             ; SGB flag
    db 0                             ; Cartridge type (ROM only)
    db 0                             ; ROM size (32KB)
    db 0                             ; RAM size (None)
    db 1                             ; Destination code (Non-Japanese)
    db $33                           ; Old licensee code
    db 0                             ; Mask ROM version
    db 0                             ; Header checksum (will be fixed by rgbfix)
    dw 0                             ; Global checksum
Start:
    ; Disable interrupts
    di

    ; Turn off LCD
    call WaitVBlank
    xor a
    ld [rLCDC], a

    ; Load simple tile data
    call LoadTileData

    ; Fill tilemap with checkerboard
    call FillTilemap

    ; Set palette (dark to light)
    ld a, %11100100
    ld [rBGP], a

    ; Turn on LCD with BG enabled
    ld a, LCDCF_ON | LCDCF_BGON
    ld [rLCDC], a

    ; Simple infinite loop (no halt, no interrupts)
.loop:
    nop
    nop
    nop
    nop
    jr .loop

; Wait for VBlank
WaitVBlank:
    ld a, [rLY]
    cp 144
    jr c, WaitVBlank
    ret

; Load two simple tiles
LoadTileData:
    ; Tile 0: All white (00)
    ld hl, $8000
    ld b, 16
.tile0:
    xor a
    ld [hl+], a
    dec b
    jr nz, .tile0

    ; Tile 1: All black (11)
    ld hl, $8010
    ld b, 16
.tile1:
    ld a, $FF
    ld [hl+], a
    dec b
    jr nz, .tile1

    ret

; Fill tilemap with alternating 0/1 pattern
FillTilemap:
    ld hl, $9800
    ld c, 0           ; Row counter

.rowLoop:
    ld b, 0           ; Column counter

.colLoop:
    ; XOR row and column to get checkerboard
    ld a, c
    and 1
    ld d, a
    ld a, b
    and 1
    xor d

    ; Write tile number (0 or 1)
    ld [hl+], a

    inc b
    ld a, b
    cp 32             ; Full tilemap width
    jr nz, .colLoop

    inc c
    ld a, c
    cp 32             ; Full tilemap height
    jr nz, .rowLoop

    ret

; Hardware register definitions
DEF rLCDC EQU $FF40
DEF rLY   EQU $FF44
DEF rBGP  EQU $FF47

; LCDC flags
DEF LCDCF_ON    EQU %10000000
DEF LCDCF_BGON  EQU %00000001
