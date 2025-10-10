package cpu

// execute executes a standard (non-CB) opcode and returns the number of cycles taken.
//
//nolint:gocognit,gocyclo // High complexity is inherent to opcode decoding (256 instructions)
func (c *CPU) execute(opcode uint8) uint8 {
	switch opcode {
	// 0x00-0x0F
	case 0x00: // NOP
		return 4
	case 0x01: // LD BC, nn
		c.Registers.SetBC(c.fetchWord())
		return 12
	case 0x02: // LD (BC), A
		c.Memory.Write(c.Registers.BC(), c.Registers.A)
		return 8
	case 0x03: // INC BC
		c.Registers.SetBC(c.Registers.BC() + 1)
		return 8
	case 0x04: // INC B
		c.Registers.B = c.inc8(c.Registers.B)
		return 4
	case 0x05: // DEC B
		c.Registers.B = c.dec8(c.Registers.B)
		return 4
	case 0x06: // LD B, n
		c.Registers.B = c.fetchByte()
		return 8
	case 0x07: // RLCA
		c.Registers.A = c.rlc(c.Registers.A)
		c.Registers.ClearFlag(FlagZ) // RLCA always clears Z
		return 4
	case 0x08: // LD (nn), SP
		addr := c.fetchWord()
		c.Memory.Write(addr, uint8(c.Registers.SP))      //nolint:gosec // G115: Intentional byte extraction
		c.Memory.Write(addr+1, uint8(c.Registers.SP>>8)) //nolint:gosec // G115: Intentional byte extraction
		return 20
	case 0x09: // ADD HL, BC
		c.Registers.SetHL(c.add16(c.Registers.HL(), c.Registers.BC()))
		return 8
	case 0x0A: // LD A, (BC)
		c.Registers.A = c.Memory.Read(c.Registers.BC())
		return 8
	case 0x0B: // DEC BC
		c.Registers.SetBC(c.Registers.BC() - 1)
		return 8
	case 0x0C: // INC C
		c.Registers.C = c.inc8(c.Registers.C)
		return 4
	case 0x0D: // DEC C
		c.Registers.C = c.dec8(c.Registers.C)
		return 4
	case 0x0E: // LD C, n
		c.Registers.C = c.fetchByte()
		return 8
	case 0x0F: // RRCA
		c.Registers.A = c.rrc(c.Registers.A)
		c.Registers.ClearFlag(FlagZ) // RRCA always clears Z
		return 4

	// 0x10-0x1F
	case 0x10: // STOP
		c.stopped = true
		c.fetchByte() // STOP is 2 bytes
		return 4
	case 0x11: // LD DE, nn
		c.Registers.SetDE(c.fetchWord())
		return 12
	case 0x12: // LD (DE), A
		c.Memory.Write(c.Registers.DE(), c.Registers.A)
		return 8
	case 0x13: // INC DE
		c.Registers.SetDE(c.Registers.DE() + 1)
		return 8
	case 0x14: // INC D
		c.Registers.D = c.inc8(c.Registers.D)
		return 4
	case 0x15: // DEC D
		c.Registers.D = c.dec8(c.Registers.D)
		return 4
	case 0x16: // LD D, n
		c.Registers.D = c.fetchByte()
		return 8
	case 0x17: // RLA
		c.Registers.A = c.rl(c.Registers.A)
		c.Registers.ClearFlag(FlagZ) // RLA always clears Z
		return 4
	case 0x18: // JR n
		offset := int8(c.fetchByte())                                  //nolint:gosec // G115: Intentional signed conversion for relative jump
		c.Registers.PC = uint16(int32(c.Registers.PC) + int32(offset)) //nolint:gosec // G115: Intentional for address calculation
		return 12
	case 0x19: // ADD HL, DE
		c.Registers.SetHL(c.add16(c.Registers.HL(), c.Registers.DE()))
		return 8
	case 0x1A: // LD A, (DE)
		c.Registers.A = c.Memory.Read(c.Registers.DE())
		return 8
	case 0x1B: // DEC DE
		c.Registers.SetDE(c.Registers.DE() - 1)
		return 8
	case 0x1C: // INC E
		c.Registers.E = c.inc8(c.Registers.E)
		return 4
	case 0x1D: // DEC E
		c.Registers.E = c.dec8(c.Registers.E)
		return 4
	case 0x1E: // LD E, n
		c.Registers.E = c.fetchByte()
		return 8
	case 0x1F: // RRA
		c.Registers.A = c.rr(c.Registers.A)
		c.Registers.ClearFlag(FlagZ) // RRA always clears Z
		return 4

	// 0x20-0x2F
	case 0x20: // JR NZ, n
		offset := int8(c.fetchByte()) //nolint:gosec // G115: Intentional signed conversion for relative jump
		if !c.Registers.ZeroFlag() {
			c.Registers.PC = uint16(int32(c.Registers.PC) + int32(offset)) //nolint:gosec // G115: Intentional for address calculation
			return 12
		}
		return 8
	case 0x21: // LD HL, nn
		c.Registers.SetHL(c.fetchWord())
		return 12
	case 0x22: // LD (HL+), A
		c.Memory.Write(c.Registers.HL(), c.Registers.A)
		c.Registers.SetHL(c.Registers.HL() + 1)
		return 8
	case 0x23: // INC HL
		c.Registers.SetHL(c.Registers.HL() + 1)
		return 8
	case 0x24: // INC H
		c.Registers.H = c.inc8(c.Registers.H)
		return 4
	case 0x25: // DEC H
		c.Registers.H = c.dec8(c.Registers.H)
		return 4
	case 0x26: // LD H, n
		c.Registers.H = c.fetchByte()
		return 8
	case 0x27: // DAA
		c.daa()
		return 4
	case 0x28: // JR Z, n
		offset := int8(c.fetchByte()) //nolint:gosec // G115: Intentional signed conversion for relative jump
		if c.Registers.ZeroFlag() {
			c.Registers.PC = uint16(int32(c.Registers.PC) + int32(offset)) //nolint:gosec // G115: Intentional for address calculation
			return 12
		}
		return 8
	case 0x29: // ADD HL, HL
		c.Registers.SetHL(c.add16(c.Registers.HL(), c.Registers.HL()))
		return 8
	case 0x2A: // LD A, (HL+)
		c.Registers.A = c.Memory.Read(c.Registers.HL())
		c.Registers.SetHL(c.Registers.HL() + 1)
		return 8
	case 0x2B: // DEC HL
		c.Registers.SetHL(c.Registers.HL() - 1)
		return 8
	case 0x2C: // INC L
		c.Registers.L = c.inc8(c.Registers.L)
		return 4
	case 0x2D: // DEC L
		c.Registers.L = c.dec8(c.Registers.L)
		return 4
	case 0x2E: // LD L, n
		c.Registers.L = c.fetchByte()
		return 8
	case 0x2F: // CPL
		c.Registers.A = ^c.Registers.A
		c.Registers.SetFlag(FlagN)
		c.Registers.SetFlag(FlagH)
		return 4

	// 0x30-0x3F
	case 0x30: // JR NC, n
		offset := int8(c.fetchByte()) //nolint:gosec // G115: Intentional signed conversion for relative jump
		if !c.Registers.CarryFlag() {
			c.Registers.PC = uint16(int32(c.Registers.PC) + int32(offset)) //nolint:gosec // G115: Intentional for address calculation
			return 12
		}
		return 8
	case 0x31: // LD SP, nn
		c.Registers.SP = c.fetchWord()
		return 12
	case 0x32: // LD (HL-), A
		c.Memory.Write(c.Registers.HL(), c.Registers.A)
		c.Registers.SetHL(c.Registers.HL() - 1)
		return 8
	case 0x33: // INC SP
		c.Registers.SP++
		return 8
	case 0x34: // INC (HL)
		addr := c.Registers.HL()
		c.Memory.Write(addr, c.inc8(c.Memory.Read(addr)))
		return 12
	case 0x35: // DEC (HL)
		addr := c.Registers.HL()
		c.Memory.Write(addr, c.dec8(c.Memory.Read(addr)))
		return 12
	case 0x36: // LD (HL), n
		c.Memory.Write(c.Registers.HL(), c.fetchByte())
		return 12
	case 0x37: // SCF
		c.Registers.ClearFlag(FlagN)
		c.Registers.ClearFlag(FlagH)
		c.Registers.SetFlag(FlagC)
		return 4
	case 0x38: // JR C, n
		offset := int8(c.fetchByte()) //nolint:gosec // G115: Intentional signed conversion for relative jump
		if c.Registers.CarryFlag() {
			c.Registers.PC = uint16(int32(c.Registers.PC) + int32(offset)) //nolint:gosec // G115: Intentional for address calculation
			return 12
		}
		return 8
	case 0x39: // ADD HL, SP
		c.Registers.SetHL(c.add16(c.Registers.HL(), c.Registers.SP))
		return 8
	case 0x3A: // LD A, (HL-)
		c.Registers.A = c.Memory.Read(c.Registers.HL())
		c.Registers.SetHL(c.Registers.HL() - 1)
		return 8
	case 0x3B: // DEC SP
		c.Registers.SP--
		return 8
	case 0x3C: // INC A
		c.Registers.A = c.inc8(c.Registers.A)
		return 4
	case 0x3D: // DEC A
		c.Registers.A = c.dec8(c.Registers.A)
		return 4
	case 0x3E: // LD A, n
		c.Registers.A = c.fetchByte()
		return 8
	case 0x3F: // CCF
		c.Registers.ClearFlag(FlagN)
		c.Registers.ClearFlag(FlagH)
		if c.Registers.CarryFlag() {
			c.Registers.ClearFlag(FlagC)
		} else {
			c.Registers.SetFlag(FlagC)
		}
		return 4

	// 0x40-0x4F: LD r, r' instructions
	case 0x40: // LD B, B
		return 4
	case 0x41: // LD B, C
		c.Registers.B = c.Registers.C
		return 4
	case 0x42: // LD B, D
		c.Registers.B = c.Registers.D
		return 4
	case 0x43: // LD B, E
		c.Registers.B = c.Registers.E
		return 4
	case 0x44: // LD B, H
		c.Registers.B = c.Registers.H
		return 4
	case 0x45: // LD B, L
		c.Registers.B = c.Registers.L
		return 4
	case 0x46: // LD B, (HL)
		c.Registers.B = c.Memory.Read(c.Registers.HL())
		return 8
	case 0x47: // LD B, A
		c.Registers.B = c.Registers.A
		return 4
	case 0x48: // LD C, B
		c.Registers.C = c.Registers.B
		return 4
	case 0x49: // LD C, C
		return 4
	case 0x4A: // LD C, D
		c.Registers.C = c.Registers.D
		return 4
	case 0x4B: // LD C, E
		c.Registers.C = c.Registers.E
		return 4
	case 0x4C: // LD C, H
		c.Registers.C = c.Registers.H
		return 4
	case 0x4D: // LD C, L
		c.Registers.C = c.Registers.L
		return 4
	case 0x4E: // LD C, (HL)
		c.Registers.C = c.Memory.Read(c.Registers.HL())
		return 8
	case 0x4F: // LD C, A
		c.Registers.C = c.Registers.A
		return 4

	// 0x50-0x5F: More LD r, r' instructions
	case 0x50: // LD D, B
		c.Registers.D = c.Registers.B
		return 4
	case 0x51: // LD D, C
		c.Registers.D = c.Registers.C
		return 4
	case 0x52: // LD D, D
		return 4
	case 0x53: // LD D, E
		c.Registers.D = c.Registers.E
		return 4
	case 0x54: // LD D, H
		c.Registers.D = c.Registers.H
		return 4
	case 0x55: // LD D, L
		c.Registers.D = c.Registers.L
		return 4
	case 0x56: // LD D, (HL)
		c.Registers.D = c.Memory.Read(c.Registers.HL())
		return 8
	case 0x57: // LD D, A
		c.Registers.D = c.Registers.A
		return 4
	case 0x58: // LD E, B
		c.Registers.E = c.Registers.B
		return 4
	case 0x59: // LD E, C
		c.Registers.E = c.Registers.C
		return 4
	case 0x5A: // LD E, D
		c.Registers.E = c.Registers.D
		return 4
	case 0x5B: // LD E, E
		return 4
	case 0x5C: // LD E, H
		c.Registers.E = c.Registers.H
		return 4
	case 0x5D: // LD E, L
		c.Registers.E = c.Registers.L
		return 4
	case 0x5E: // LD E, (HL)
		c.Registers.E = c.Memory.Read(c.Registers.HL())
		return 8
	case 0x5F: // LD E, A
		c.Registers.E = c.Registers.A
		return 4

	// 0x60-0x6F: More LD r, r' instructions
	case 0x60: // LD H, B
		c.Registers.H = c.Registers.B
		return 4
	case 0x61: // LD H, C
		c.Registers.H = c.Registers.C
		return 4
	case 0x62: // LD H, D
		c.Registers.H = c.Registers.D
		return 4
	case 0x63: // LD H, E
		c.Registers.H = c.Registers.E
		return 4
	case 0x64: // LD H, H
		return 4
	case 0x65: // LD H, L
		c.Registers.H = c.Registers.L
		return 4
	case 0x66: // LD H, (HL)
		c.Registers.H = c.Memory.Read(c.Registers.HL())
		return 8
	case 0x67: // LD H, A
		c.Registers.H = c.Registers.A
		return 4
	case 0x68: // LD L, B
		c.Registers.L = c.Registers.B
		return 4
	case 0x69: // LD L, C
		c.Registers.L = c.Registers.C
		return 4
	case 0x6A: // LD L, D
		c.Registers.L = c.Registers.D
		return 4
	case 0x6B: // LD L, E
		c.Registers.L = c.Registers.E
		return 4
	case 0x6C: // LD L, H
		c.Registers.L = c.Registers.H
		return 4
	case 0x6D: // LD L, L
		return 4
	case 0x6E: // LD L, (HL)
		c.Registers.L = c.Memory.Read(c.Registers.HL())
		return 8
	case 0x6F: // LD L, A
		c.Registers.L = c.Registers.A
		return 4

	// 0x70-0x7F: LD (HL), r and LD A, r instructions
	case 0x70: // LD (HL), B
		c.Memory.Write(c.Registers.HL(), c.Registers.B)
		return 8
	case 0x71: // LD (HL), C
		c.Memory.Write(c.Registers.HL(), c.Registers.C)
		return 8
	case 0x72: // LD (HL), D
		c.Memory.Write(c.Registers.HL(), c.Registers.D)
		return 8
	case 0x73: // LD (HL), E
		c.Memory.Write(c.Registers.HL(), c.Registers.E)
		return 8
	case 0x74: // LD (HL), H
		c.Memory.Write(c.Registers.HL(), c.Registers.H)
		return 8
	case 0x75: // LD (HL), L
		c.Memory.Write(c.Registers.HL(), c.Registers.L)
		return 8
	case 0x76: // HALT
		// HALT is special: on hardware it fetches with IR = [PC] (no increment)
		// but we've already incremented PC in fetchByte(), so undo it
		// However, if wasHaltBug is true, fetchByte() already didn't increment PC,
		// so we shouldn't decrement it (prevents double decrement in repeated HALT scenario)
		if !c.wasHaltBug {
			c.Registers.PC--
		}
		c.halted = true
		return 4
	case 0x77: // LD (HL), A
		c.Memory.Write(c.Registers.HL(), c.Registers.A)
		return 8
	case 0x78: // LD A, B
		c.Registers.A = c.Registers.B
		return 4
	case 0x79: // LD A, C
		c.Registers.A = c.Registers.C
		return 4
	case 0x7A: // LD A, D
		c.Registers.A = c.Registers.D
		return 4
	case 0x7B: // LD A, E
		c.Registers.A = c.Registers.E
		return 4
	case 0x7C: // LD A, H
		c.Registers.A = c.Registers.H
		return 4
	case 0x7D: // LD A, L
		c.Registers.A = c.Registers.L
		return 4
	case 0x7E: // LD A, (HL)
		c.Registers.A = c.Memory.Read(c.Registers.HL())
		return 8
	case 0x7F: // LD A, A
		return 4

	// 0x80-0x8F: Arithmetic operations with A
	case 0x80: // ADD A, B
		c.Registers.A = c.add8(c.Registers.A, c.Registers.B, false)
		return 4
	case 0x81: // ADD A, C
		c.Registers.A = c.add8(c.Registers.A, c.Registers.C, false)
		return 4
	case 0x82: // ADD A, D
		c.Registers.A = c.add8(c.Registers.A, c.Registers.D, false)
		return 4
	case 0x83: // ADD A, E
		c.Registers.A = c.add8(c.Registers.A, c.Registers.E, false)
		return 4
	case 0x84: // ADD A, H
		c.Registers.A = c.add8(c.Registers.A, c.Registers.H, false)
		return 4
	case 0x85: // ADD A, L
		c.Registers.A = c.add8(c.Registers.A, c.Registers.L, false)
		return 4
	case 0x86: // ADD A, (HL)
		c.Registers.A = c.add8(c.Registers.A, c.Memory.Read(c.Registers.HL()), false)
		return 8
	case 0x87: // ADD A, A
		c.Registers.A = c.add8(c.Registers.A, c.Registers.A, false)
		return 4
	case 0x88: // ADC A, B
		c.Registers.A = c.add8(c.Registers.A, c.Registers.B, true)
		return 4
	case 0x89: // ADC A, C
		c.Registers.A = c.add8(c.Registers.A, c.Registers.C, true)
		return 4
	case 0x8A: // ADC A, D
		c.Registers.A = c.add8(c.Registers.A, c.Registers.D, true)
		return 4
	case 0x8B: // ADC A, E
		c.Registers.A = c.add8(c.Registers.A, c.Registers.E, true)
		return 4
	case 0x8C: // ADC A, H
		c.Registers.A = c.add8(c.Registers.A, c.Registers.H, true)
		return 4
	case 0x8D: // ADC A, L
		c.Registers.A = c.add8(c.Registers.A, c.Registers.L, true)
		return 4
	case 0x8E: // ADC A, (HL)
		c.Registers.A = c.add8(c.Registers.A, c.Memory.Read(c.Registers.HL()), true)
		return 8
	case 0x8F: // ADC A, A
		c.Registers.A = c.add8(c.Registers.A, c.Registers.A, true)
		return 4

	// 0x90-0x9F: Subtraction operations
	case 0x90: // SUB B
		c.Registers.A = c.sub8(c.Registers.A, c.Registers.B, false)
		return 4
	case 0x91: // SUB C
		c.Registers.A = c.sub8(c.Registers.A, c.Registers.C, false)
		return 4
	case 0x92: // SUB D
		c.Registers.A = c.sub8(c.Registers.A, c.Registers.D, false)
		return 4
	case 0x93: // SUB E
		c.Registers.A = c.sub8(c.Registers.A, c.Registers.E, false)
		return 4
	case 0x94: // SUB H
		c.Registers.A = c.sub8(c.Registers.A, c.Registers.H, false)
		return 4
	case 0x95: // SUB L
		c.Registers.A = c.sub8(c.Registers.A, c.Registers.L, false)
		return 4
	case 0x96: // SUB (HL)
		c.Registers.A = c.sub8(c.Registers.A, c.Memory.Read(c.Registers.HL()), false)
		return 8
	case 0x97: // SUB A
		c.Registers.A = c.sub8(c.Registers.A, c.Registers.A, false)
		return 4
	case 0x98: // SBC A, B
		c.Registers.A = c.sub8(c.Registers.A, c.Registers.B, true)
		return 4
	case 0x99: // SBC A, C
		c.Registers.A = c.sub8(c.Registers.A, c.Registers.C, true)
		return 4
	case 0x9A: // SBC A, D
		c.Registers.A = c.sub8(c.Registers.A, c.Registers.D, true)
		return 4
	case 0x9B: // SBC A, E
		c.Registers.A = c.sub8(c.Registers.A, c.Registers.E, true)
		return 4
	case 0x9C: // SBC A, H
		c.Registers.A = c.sub8(c.Registers.A, c.Registers.H, true)
		return 4
	case 0x9D: // SBC A, L
		c.Registers.A = c.sub8(c.Registers.A, c.Registers.L, true)
		return 4
	case 0x9E: // SBC A, (HL)
		c.Registers.A = c.sub8(c.Registers.A, c.Memory.Read(c.Registers.HL()), true)
		return 8
	case 0x9F: // SBC A, A
		c.Registers.A = c.sub8(c.Registers.A, c.Registers.A, true)
		return 4

	// 0xA0-0xAF: Logic operations
	case 0xA0: // AND B
		c.Registers.A = c.and(c.Registers.B)
		return 4
	case 0xA1: // AND C
		c.Registers.A = c.and(c.Registers.C)
		return 4
	case 0xA2: // AND D
		c.Registers.A = c.and(c.Registers.D)
		return 4
	case 0xA3: // AND E
		c.Registers.A = c.and(c.Registers.E)
		return 4
	case 0xA4: // AND H
		c.Registers.A = c.and(c.Registers.H)
		return 4
	case 0xA5: // AND L
		c.Registers.A = c.and(c.Registers.L)
		return 4
	case 0xA6: // AND (HL)
		c.Registers.A = c.and(c.Memory.Read(c.Registers.HL()))
		return 8
	case 0xA7: // AND A
		c.Registers.A = c.and(c.Registers.A)
		return 4
	case 0xA8: // XOR B
		c.Registers.A = c.xor(c.Registers.B)
		return 4
	case 0xA9: // XOR C
		c.Registers.A = c.xor(c.Registers.C)
		return 4
	case 0xAA: // XOR D
		c.Registers.A = c.xor(c.Registers.D)
		return 4
	case 0xAB: // XOR E
		c.Registers.A = c.xor(c.Registers.E)
		return 4
	case 0xAC: // XOR H
		c.Registers.A = c.xor(c.Registers.H)
		return 4
	case 0xAD: // XOR L
		c.Registers.A = c.xor(c.Registers.L)
		return 4
	case 0xAE: // XOR (HL)
		c.Registers.A = c.xor(c.Memory.Read(c.Registers.HL()))
		return 8
	case 0xAF: // XOR A
		c.Registers.A = c.xor(c.Registers.A)
		return 4

	// 0xB0-0xBF: OR and CP operations
	case 0xB0: // OR B
		c.Registers.A = c.or(c.Registers.B)
		return 4
	case 0xB1: // OR C
		c.Registers.A = c.or(c.Registers.C)
		return 4
	case 0xB2: // OR D
		c.Registers.A = c.or(c.Registers.D)
		return 4
	case 0xB3: // OR E
		c.Registers.A = c.or(c.Registers.E)
		return 4
	case 0xB4: // OR H
		c.Registers.A = c.or(c.Registers.H)
		return 4
	case 0xB5: // OR L
		c.Registers.A = c.or(c.Registers.L)
		return 4
	case 0xB6: // OR (HL)
		c.Registers.A = c.or(c.Memory.Read(c.Registers.HL()))
		return 8
	case 0xB7: // OR A
		c.Registers.A = c.or(c.Registers.A)
		return 4
	case 0xB8: // CP B
		c.cp(c.Registers.B)
		return 4
	case 0xB9: // CP C
		c.cp(c.Registers.C)
		return 4
	case 0xBA: // CP D
		c.cp(c.Registers.D)
		return 4
	case 0xBB: // CP E
		c.cp(c.Registers.E)
		return 4
	case 0xBC: // CP H
		c.cp(c.Registers.H)
		return 4
	case 0xBD: // CP L
		c.cp(c.Registers.L)
		return 4
	case 0xBE: // CP (HL)
		c.cp(c.Memory.Read(c.Registers.HL()))
		return 8
	case 0xBF: // CP A
		c.cp(c.Registers.A)
		return 4

	// 0xC0-0xCF: Returns, pops, jumps, and calls
	case 0xC0: // RET NZ
		if !c.Registers.ZeroFlag() {
			c.Registers.PC = c.pop()
			return 20
		}
		return 8
	case 0xC1: // POP BC
		c.Registers.SetBC(c.pop())
		return 12
	case 0xC2: // JP NZ, nn
		addr := c.fetchWord()
		if !c.Registers.ZeroFlag() {
			c.Registers.PC = addr
			return 16
		}
		return 12
	case 0xC3: // JP nn
		c.Registers.PC = c.fetchWord()
		return 16
	case 0xC4: // CALL NZ, nn
		addr := c.fetchWord()
		if !c.Registers.ZeroFlag() {
			c.push(c.Registers.PC)
			c.Registers.PC = addr
			return 24
		}
		return 12
	case 0xC5: // PUSH BC
		c.push(c.Registers.BC())
		return 16
	case 0xC6: // ADD A, n
		c.Registers.A = c.add8(c.Registers.A, c.fetchByte(), false)
		return 8
	case 0xC7: // RST 00H
		c.push(c.Registers.PC)
		c.Registers.PC = 0x00
		return 16
	case 0xC8: // RET Z
		if c.Registers.ZeroFlag() {
			c.Registers.PC = c.pop()
			return 20
		}
		return 8
	case 0xC9: // RET
		c.Registers.PC = c.pop()
		return 16
	case 0xCA: // JP Z, nn
		addr := c.fetchWord()
		if c.Registers.ZeroFlag() {
			c.Registers.PC = addr
			return 16
		}
		return 12
	case 0xCB: // CB prefix
		// This should be handled in Step(), not here
		panic("CB prefix should not reach execute()")
	case 0xCC: // CALL Z, nn
		addr := c.fetchWord()
		if c.Registers.ZeroFlag() {
			c.push(c.Registers.PC)
			c.Registers.PC = addr
			return 24
		}
		return 12
	case 0xCD: // CALL nn
		addr := c.fetchWord()
		c.push(c.Registers.PC)
		c.Registers.PC = addr
		return 24
	case 0xCE: // ADC A, n
		c.Registers.A = c.add8(c.Registers.A, c.fetchByte(), true)
		return 8
	case 0xCF: // RST 08H
		c.push(c.Registers.PC)
		c.Registers.PC = 0x08
		return 16

	// 0xD0-0xDF: More returns, pops, jumps, and calls
	case 0xD0: // RET NC
		if !c.Registers.CarryFlag() {
			c.Registers.PC = c.pop()
			return 20
		}
		return 8
	case 0xD1: // POP DE
		c.Registers.SetDE(c.pop())
		return 12
	case 0xD2: // JP NC, nn
		addr := c.fetchWord()
		if !c.Registers.CarryFlag() {
			c.Registers.PC = addr
			return 16
		}
		return 12
	case 0xD3: // Invalid opcode
		panic("Invalid opcode 0xD3")
	case 0xD4: // CALL NC, nn
		addr := c.fetchWord()
		if !c.Registers.CarryFlag() {
			c.push(c.Registers.PC)
			c.Registers.PC = addr
			return 24
		}
		return 12
	case 0xD5: // PUSH DE
		c.push(c.Registers.DE())
		return 16
	case 0xD6: // SUB n
		c.Registers.A = c.sub8(c.Registers.A, c.fetchByte(), false)
		return 8
	case 0xD7: // RST 10H
		c.push(c.Registers.PC)
		c.Registers.PC = 0x10
		return 16
	case 0xD8: // RET C
		if c.Registers.CarryFlag() {
			c.Registers.PC = c.pop()
			return 20
		}
		return 8
	case 0xD9: // RETI
		c.Registers.PC = c.pop()
		c.IME = true
		return 16
	case 0xDA: // JP C, nn
		addr := c.fetchWord()
		if c.Registers.CarryFlag() {
			c.Registers.PC = addr
			return 16
		}
		return 12
	case 0xDB: // Invalid opcode
		panic("Invalid opcode 0xDB")
	case 0xDC: // CALL C, nn
		addr := c.fetchWord()
		if c.Registers.CarryFlag() {
			c.push(c.Registers.PC)
			c.Registers.PC = addr
			return 24
		}
		return 12
	case 0xDD: // Invalid opcode
		panic("Invalid opcode 0xDD")
	case 0xDE: // SBC A, n
		c.Registers.A = c.sub8(c.Registers.A, c.fetchByte(), true)
		return 8
	case 0xDF: // RST 18H
		c.push(c.Registers.PC)
		c.Registers.PC = 0x18
		return 16

	// 0xE0-0xEF: I/O operations and more
	case 0xE0: // LDH (n), A
		c.Memory.Write(0xFF00+uint16(c.fetchByte()), c.Registers.A)
		return 12
	case 0xE1: // POP HL
		c.Registers.SetHL(c.pop())
		return 12
	case 0xE2: // LD (C), A
		c.Memory.Write(0xFF00+uint16(c.Registers.C), c.Registers.A)
		return 8
	case 0xE3: // Invalid opcode
		panic("Invalid opcode 0xE3")
	case 0xE4: // Invalid opcode
		panic("Invalid opcode 0xE4")
	case 0xE5: // PUSH HL
		c.push(c.Registers.HL())
		return 16
	case 0xE6: // AND n
		c.Registers.A = c.and(c.fetchByte())
		return 8
	case 0xE7: // RST 20H
		c.push(c.Registers.PC)
		c.Registers.PC = 0x20
		return 16
	case 0xE8: // ADD SP, n
		offset := int8(c.fetchByte())                           //nolint:gosec // G115: Intentional signed conversion for relative jump
		result := uint16(int32(c.Registers.SP) + int32(offset)) //nolint:gosec // G115: Intentional for SP offset calculation
		// Flags for ADD SP, n are different
		c.Registers.ClearFlag(FlagZ)
		c.Registers.ClearFlag(FlagN)
		c.Registers.SetFlagTo(FlagH, (c.Registers.SP&0x0F)+(uint16(offset)&0x0F) > 0x0F) //nolint:gosec // G115: Intentional for flag calculation
		c.Registers.SetFlagTo(FlagC, (c.Registers.SP&0xFF)+(uint16(offset)&0xFF) > 0xFF) //nolint:gosec // G115: Intentional for flag calculation
		c.Registers.SP = result
		return 16
	case 0xE9: // JP (HL)
		c.Registers.PC = c.Registers.HL()
		return 4
	case 0xEA: // LD (nn), A
		c.Memory.Write(c.fetchWord(), c.Registers.A)
		return 16
	case 0xEB: // Invalid opcode
		panic("Invalid opcode 0xEB")
	case 0xEC: // Invalid opcode
		panic("Invalid opcode 0xEC")
	case 0xED: // Invalid opcode
		panic("Invalid opcode 0xED")
	case 0xEE: // XOR n
		c.Registers.A = c.xor(c.fetchByte())
		return 8
	case 0xEF: // RST 28H
		c.push(c.Registers.PC)
		c.Registers.PC = 0x28
		return 16

	// 0xF0-0xFF: I/O operations and more
	case 0xF0: // LDH A, (n)
		c.Registers.A = c.Memory.Read(0xFF00 + uint16(c.fetchByte()))
		return 12
	case 0xF1: // POP AF
		c.Registers.SetAF(c.pop())
		return 12
	case 0xF2: // LD A, (C)
		c.Registers.A = c.Memory.Read(0xFF00 + uint16(c.Registers.C))
		return 8
	case 0xF3: // DI
		c.IME = false
		c.pendingIME = false // Cancel any pending EI
		return 4
	case 0xF4: // Invalid opcode
		panic("Invalid opcode 0xF4")
	case 0xF5: // PUSH AF
		c.push(c.Registers.AF())
		return 16
	case 0xF6: // OR n
		c.Registers.A = c.or(c.fetchByte())
		return 8
	case 0xF7: // RST 30H
		c.push(c.Registers.PC)
		c.Registers.PC = 0x30
		return 16
	case 0xF8: // LD HL, SP+n
		offset := int8(c.fetchByte())                           //nolint:gosec // G115: Intentional signed conversion for relative jump
		result := uint16(int32(c.Registers.SP) + int32(offset)) //nolint:gosec // G115: Intentional for SP offset calculation
		// Flags for LD HL, SP+n
		c.Registers.ClearFlag(FlagZ)
		c.Registers.ClearFlag(FlagN)
		c.Registers.SetFlagTo(FlagH, (c.Registers.SP&0x0F)+(uint16(offset)&0x0F) > 0x0F) //nolint:gosec // G115: Intentional for flag calculation
		c.Registers.SetFlagTo(FlagC, (c.Registers.SP&0xFF)+(uint16(offset)&0xFF) > 0xFF) //nolint:gosec // G115: Intentional for flag calculation
		c.Registers.SetHL(result)
		return 12
	case 0xF9: // LD SP, HL
		c.Registers.SP = c.Registers.HL()
		return 8
	case 0xFA: // LD A, (nn)
		c.Registers.A = c.Memory.Read(c.fetchWord())
		return 16
	case 0xFB: // EI
		// EI enables interrupts AFTER the next instruction executes
		c.pendingIME = true
		return 4
	case 0xFC: // Invalid opcode
		panic("Invalid opcode 0xFC")
	case 0xFD: // Invalid opcode
		panic("Invalid opcode 0xFD")
	case 0xFE: // CP n
		c.cp(c.fetchByte())
		return 8
	case 0xFF: // RST 38H
		c.push(c.Registers.PC)
		c.Registers.PC = 0x38
		return 16

	default:
		panic("Unknown opcode")
	}
}

// daa performs Decimal Adjust Accumulator (DAA) operation.
func (c *CPU) daa() {
	a := c.Registers.A

	if !c.Registers.SubtractFlag() { //nolint:nestif // Complex nested logic is required for BCD adjustment
		// After addition
		if c.Registers.CarryFlag() || a > 0x99 {
			a += 0x60
			c.Registers.SetFlag(FlagC)
		}
		if c.Registers.HalfCarryFlag() || (a&0x0F) > 0x09 {
			a += 0x06
		}
	} else {
		// After subtraction
		if c.Registers.CarryFlag() {
			a -= 0x60
		}
		if c.Registers.HalfCarryFlag() {
			a -= 0x06
		}
	}

	c.Registers.A = a
	c.Registers.SetFlagTo(FlagZ, a == 0)
	c.Registers.ClearFlag(FlagH)
}
