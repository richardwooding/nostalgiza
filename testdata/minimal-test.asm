; Minimal Game Boy Test - Just set up tiles and tilemap, don't touch LCD
; The bootrom already initializes the LCD, so we just need to provide graphics data

SECTION "ROM", ROM0[$100]
    nop
    jp Start

    ds $134 - @
    db "MINIMAL TEST"
    ds 4
    db 0, 0, 0, 0, 0, 0, 0, 1, $33, 0, 0
    dw 0

Start:
    di

    ; Just set up tiles and loop - don't touch LCDC
    ; The bootrom already set LCDC=$91 (LCD on, BG on)

    ; Load tile 0 - solid color 0 (white)
    ld hl, $8000
    ld b, 16
.tile0:
    xor a
    ld [hl+], a
    dec b
    jr nz, .tile0

    ; Load tile 1 - solid color 3 (black)
    ld hl, $8010
    ld b, 16
.tile1:
    ld a, $FF
    ld [hl+], a
    dec b
    jr nz, .tile1

    ; Fill tilemap with checkerboard
    ld hl, $9800
    ld d, 0

.row:
    ld e, 0
.col:
    ; Calculate tile (row XOR col) & 1
    ld a, d
    xor e
    and 1
    ld [hl+], a

    inc e
    ld a, e
    cp 32
    jr nz, .col

    inc d
    ld a, d
    cp 32
    jr nz, .row

    ; Set palette
    ld a, %11100100
    ld [$FF47], a

    ; Infinite loop
.loop:
    jr .loop
