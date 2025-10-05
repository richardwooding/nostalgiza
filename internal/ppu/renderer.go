package ppu

// renderScanline renders the current scanline to the framebuffer.
// This is called during mode 3 (drawing) for each scanline.
func (p *PPU) renderScanline() {
	// Only render if LCD is enabled
	if p.lcdc&LCDCLCDEnable == 0 {
		return
	}

	// Render background if enabled
	if p.lcdc&LCDCBGWindowEnable != 0 {
		p.renderBackground()
	} else {
		// If BG is disabled, fill with white (color 0)
		p.clearScanline()
	}

	// Render window if enabled
	if p.lcdc&LCDCWindowEnable != 0 {
		p.renderWindow()
	}

	// Render sprites if enabled
	if p.lcdc&LCDCOBJEnable != 0 {
		p.renderSprites()
	}
}

// clearScanline fills the current scanline with white (color index 0).
func (p *PPU) clearScanline() {
	offset := int(p.ly) * ScreenWidth
	for x := 0; x < ScreenWidth; x++ {
		p.framebuffer[offset+x] = 0
	}
}

// renderBackground renders the background layer for the current scanline.
func (p *PPU) renderBackground() {
	// Determine which tile map to use
	tileMapBase := uint16(0x1800) // 0x9800 - 0x8000
	if p.lcdc&LCDCBGTileMap != 0 {
		tileMapBase = 0x1C00 // 0x9C00 - 0x8000
	}

	// Determine tile data addressing mode
	useSigned := p.lcdc&LCDCBGTileData == 0
	tileDataBase := uint16(0x0000)
	if useSigned {
		tileDataBase = 0x0800 // 0x8800 - 0x8000
	}

	// Calculate Y position in background map (with scrolling)
	y := uint16(p.ly) + uint16(p.scy)
	tileRow := (y / 8) % 32 // 32 tiles per row in tile map

	// Render each pixel of the scanline
	for x := uint16(0); x < ScreenWidth; x++ {
		// Calculate X position in background map (with scrolling)
		scrolledX := x + uint16(p.scx)
		tileCol := (scrolledX / 8) % 32 // 32 tiles per column

		// Get tile index from tile map
		tileMapAddr := tileMapBase + (tileRow * 32) + tileCol
		tileIndex := p.vram[tileMapAddr]

		// Calculate tile data address
		tileAddr := p.getTileDataAddr(tileIndex, useSigned, tileDataBase)

		// Get pixel within tile
		tileY := y % 8
		tileX := scrolledX % 8

		// Get pixel color index
		colorIndex := p.getTilePixel(tileAddr, tileX, tileY)

		// Apply background palette
		color := p.applyPalette(colorIndex, p.bgp)

		// Write to framebuffer
		p.framebuffer[int(p.ly)*ScreenWidth+int(x)] = color
	}
}

// renderWindow renders the window layer for the current scanline.
func (p *PPU) renderWindow() {
	// Window must be visible on this scanline
	if p.ly < p.wy {
		return
	}

	// Determine which tile map to use for window
	tileMapBase := uint16(0x1800) // 0x9800 - 0x8000
	if p.lcdc&LCDCWindowTileMap != 0 {
		tileMapBase = 0x1C00 // 0x9C00 - 0x8000
	}

	// Determine tile data addressing mode
	useSigned := p.lcdc&LCDCBGTileData == 0
	tileDataBase := uint16(0x0000)
	if useSigned {
		tileDataBase = 0x0800
	}

	// Calculate window Y coordinate (no scrolling)
	windowY := uint16(p.ly) - uint16(p.wy)
	tileRow := (windowY / 8) % 32

	// Window X position is offset by 7
	windowXOffset := int16(p.wx) - 7
	if windowXOffset < 0 {
		windowXOffset = 0
	}

	// Render each pixel of the window on this scanline
	for x := uint16(0); x < ScreenWidth; x++ {
		// Check if this pixel is in the window
		if int16(x) < windowXOffset {
			continue
		}

		windowX := uint16(int16(x) - windowXOffset) //nolint:gosec // Intentional conversion
		tileCol := (windowX / 8) % 32

		// Get tile index from window tile map
		tileMapAddr := tileMapBase + (tileRow * 32) + tileCol
		tileIndex := p.vram[tileMapAddr]

		// Calculate tile data address
		tileAddr := p.getTileDataAddr(tileIndex, useSigned, tileDataBase)

		// Get pixel within tile
		tileY := windowY % 8
		tileX := windowX % 8

		// Get pixel color index
		colorIndex := p.getTilePixel(tileAddr, tileX, tileY)

		// Apply background palette
		color := p.applyPalette(colorIndex, p.bgp)

		// Write to framebuffer
		p.framebuffer[int(p.ly)*ScreenWidth+int(x)] = color
	}
}

// renderSprites renders sprites (objects) for the current scanline.
//
//nolint:gocognit // Sprite rendering is inherently complex
func (p *PPU) renderSprites() {
	spriteHeight := uint16(8)
	if p.lcdc&LCDCOBJSize != 0 {
		spriteHeight = 16
	}

	// Reset sprite buffer (reuse allocation to reduce GC pressure)
	p.spriteBuffer = p.spriteBuffer[:0]

	// Scan OAM for sprites on this scanline
	for i := 0; i < 40; i++ {
		oamAddr := i * 4

		y := int16(p.oam[oamAddr]) - 16
		x := int16(p.oam[oamAddr+1]) - 8
		tileIndex := p.oam[oamAddr+2]
		attrs := p.oam[oamAddr+3]

		// Check if sprite is on this scanline
		scanline := int16(p.ly)
		if scanline >= y && scanline < y+int16(spriteHeight) { //nolint:gosec // Intentional conversion
			p.spriteBuffer = append(p.spriteBuffer, sprite{
				x:         x,
				y:         y,
				tileIndex: tileIndex,
				attrs:     attrs,
				oamIndex:  i,
			})

			// Max 10 sprites per scanline
			if len(p.spriteBuffer) >= 10 {
				break
			}
		}
	}

	// Render sprites in reverse order (higher priority last)
	for i := len(p.spriteBuffer) - 1; i >= 0; i-- {
		spr := p.spriteBuffer[i]

		// Calculate which line of the sprite to render
		spriteLine := uint16(int16(p.ly) - spr.y) //nolint:gosec // Intentional conversion

		// Apply Y flip
		if spr.attrs&SpriteAttrYFlip != 0 {
			spriteLine = spriteHeight - 1 - spriteLine
		}

		// For 8x16 sprites, use two tiles
		tileIndex := uint16(spr.tileIndex)
		if spriteHeight == 16 {
			// In 8x16 mode, bit 0 is ignored
			tileIndex &= 0xFE
			// Use second tile for bottom half
			if spriteLine >= 8 {
				tileIndex++
				spriteLine -= 8
			}
		}

		// Get tile data address (sprites always use 0x8000 addressing)
		tileAddr := tileIndex * 16

		// Render each pixel of the sprite
		for x := uint16(0); x < 8; x++ {
			pixelX := spr.x + int16(x)

			// Skip pixels outside screen
			if pixelX < 0 || pixelX >= ScreenWidth {
				continue
			}

			// Apply X flip
			tileX := x
			if spr.attrs&SpriteAttrXFlip != 0 {
				tileX = 7 - x
			}

			// Get pixel color index
			colorIndex := p.getTilePixel(tileAddr, tileX, spriteLine)

			// Color 0 is transparent for sprites
			if colorIndex == 0 {
				continue
			}

			// Check sprite priority
			bgColor := p.framebuffer[int(p.ly)*ScreenWidth+int(pixelX)]
			if spr.attrs&SpriteAttrPriority != 0 && bgColor != 0 {
				// Sprite is behind BG colors 1-3
				continue
			}

			// Apply sprite palette
			palette := p.obp0
			if spr.attrs&SpriteAttrPalette != 0 {
				palette = p.obp1
			}
			color := p.applyPalette(colorIndex, palette)

			// Write to framebuffer
			p.framebuffer[int(p.ly)*ScreenWidth+int(pixelX)] = color
		}
	}
}

// getTileDataAddr calculates the address of tile data.
func (p *PPU) getTileDataAddr(tileIndex uint8, useSigned bool, base uint16) uint16 {
	if useSigned {
		// Signed addressing: base at 0x9000 (0x0800 in VRAM)
		signedIndex := int16(int8(tileIndex))                              //nolint:gosec // Intentional signed conversion
		return uint16(int32(base) + int32(0x0800) + int32(signedIndex)*16) //nolint:gosec // Intentional conversion
	}
	// Unsigned addressing: base at 0x8000 (0x0000 in VRAM)
	return base + uint16(tileIndex)*16
}

// getTilePixel gets a pixel from a tile.
// Tiles are 8x8 pixels, 2 bits per pixel, stored as 16 bytes.
func (p *PPU) getTilePixel(tileAddr, x, y uint16) uint8 {
	// Each row is 2 bytes
	lineAddr := tileAddr + (y * 2)

	// Get the two bytes for this line
	byte1 := p.vram[lineAddr]
	byte2 := p.vram[lineAddr+1]

	// Extract the bit for this pixel (bit 7 is pixel 0, bit 0 is pixel 7)
	bitPos := 7 - x
	bit1 := (byte1 >> bitPos) & 1
	bit2 := (byte2 >> bitPos) & 1

	// Combine to get color index (0-3)
	return (bit2 << 1) | bit1
}

// applyPalette applies a palette to convert a color index (0-3) to a shade (0-3).
func (p *PPU) applyPalette(colorIndex, palette uint8) uint8 {
	// Extract 2-bit shade for this color index
	shift := colorIndex * 2
	return (palette >> shift) & 0x03
}
