package cpu

// executeCB executes a CB-prefixed opcode and returns the number of cycles taken.
func (c *CPU) executeCB(opcode uint8) uint8 {
	// Get the target register/memory based on lower 3 bits
	getTarget := func() *uint8 {
		switch opcode & 0x07 {
		case 0:
			return &c.Registers.B
		case 1:
			return &c.Registers.C
		case 2:
			return &c.Registers.D
		case 3:
			return &c.Registers.E
		case 4:
			return &c.Registers.H
		case 5:
			return &c.Registers.L
		case 6:
			return nil // (HL)
		case 7:
			return &c.Registers.A
		}
		return nil
	}

	// Helper to get value (handles (HL) case)
	getValue := func() uint8 {
		if opcode&0x07 == 6 {
			return c.Memory.Read(c.Registers.HL())
		}
		target := getTarget()
		if target != nil {
			return *target
		}
		return 0
	}

	// Helper to set value (handles (HL) case)
	setValue := func(value uint8) {
		if opcode&0x07 == 6 {
			c.Memory.Write(c.Registers.HL(), value)
		} else {
			target := getTarget()
			if target != nil {
				*target = value
			}
		}
	}

	// Determine operation type and bit number
	operation := (opcode >> 6) & 0x03
	bitNum := (opcode >> 3) & 0x07

	// Calculate cycles (most are 8, (HL) operations are 16, BIT (HL) is 12)
	cycles := uint8(8)
	if opcode&0x07 == 6 {
		if operation == 1 { // BIT
			cycles = 12
		} else {
			cycles = 16
		}
	}

	switch operation {
	case 0: // Rotates and shifts (0x00-0x3F)
		value := getValue()
		var result uint8

		switch bitNum {
		case 0: // RLC
			result = c.rlc(value)
		case 1: // RRC
			result = c.rrc(value)
		case 2: // RL
			result = c.rl(value)
		case 3: // RR
			result = c.rr(value)
		case 4: // SLA
			result = c.sla(value)
		case 5: // SRA
			result = c.sra(value)
		case 6: // SWAP
			result = c.swap(value)
		case 7: // SRL
			result = c.srl(value)
		}

		setValue(result)
		return cycles

	case 1: // BIT (0x40-0x7F)
		value := getValue()
		c.bit(value, bitNum)
		return cycles

	case 2: // RES (0x80-0xBF)
		value := getValue()
		result := value &^ (1 << bitNum)
		setValue(result)
		return cycles

	case 3: // SET (0xC0-0xFF)
		value := getValue()
		result := value | (1 << bitNum)
		setValue(result)
		return cycles
	}

	return cycles
}
