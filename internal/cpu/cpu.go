// Package cpu implements the Sharp SM83 CPU emulation for the Game Boy.
package cpu

// Memory interface for CPU to access memory bus.
type Memory interface {
	Read(addr uint16) uint8
	Write(addr uint16, value uint8)
}

// CPU represents the Sharp SM83 CPU.
type CPU struct {
	Registers *Registers
	Memory    Memory

	// Interrupt master enable flag
	IME bool

	// Halt and stop states
	halted  bool
	stopped bool

	// Cycle counter
	Cycles uint64
}

// New creates a new CPU instance.
func New(mem Memory) *CPU {
	return &CPU{
		Registers: NewRegisters(),
		Memory:    mem,
		IME:       false,
		halted:    false,
		stopped:   false,
		Cycles:    0,
	}
}

// Step executes one instruction and returns cycles taken.
func (c *CPU) Step() uint8 {
	// Handle halt state
	if c.halted {
		// TODO: Check for interrupts (Phase 4)
		// For now, just consume 1 M-cycle
		return 4
	}

	// Fetch instruction
	opcode := c.fetchByte()

	// Decode and execute
	var cycles uint8
	if opcode == 0xCB {
		// CB-prefixed instruction
		cbOpcode := c.fetchByte()
		cycles = c.executeCB(cbOpcode)
	} else {
		cycles = c.execute(opcode)
	}

	// Update cycle counter
	c.Cycles += uint64(cycles)

	return cycles
}

// fetchByte fetches the next byte from memory and increments PC.
func (c *CPU) fetchByte() uint8 {
	value := c.Memory.Read(c.Registers.PC)
	c.Registers.PC++
	return value
}

// fetchWord fetches the next word (16-bit) from memory and increments PC.
func (c *CPU) fetchWord() uint16 {
	low := uint16(c.fetchByte())
	high := uint16(c.fetchByte())
	return high<<8 | low
}

// push pushes a 16-bit value onto the stack.
func (c *CPU) push(value uint16) {
	c.Registers.SP -= 2
	c.Memory.Write(c.Registers.SP, uint8(value))      //nolint:gosec // G115: Intentional byte extraction from 16-bit value
	c.Memory.Write(c.Registers.SP+1, uint8(value>>8)) //nolint:gosec // G115: Intentional byte extraction from 16-bit value
}

// pop pops a 16-bit value from the stack.
func (c *CPU) pop() uint16 {
	low := uint16(c.Memory.Read(c.Registers.SP))
	high := uint16(c.Memory.Read(c.Registers.SP + 1))
	c.Registers.SP += 2
	return high<<8 | low
}

// Helper methods for arithmetic operations

// add8 performs 8-bit addition and sets flags.
func (c *CPU) add8(a, b uint8, carry bool) uint8 {
	carryVal := uint8(0)
	if carry && c.Registers.CarryFlag() {
		carryVal = 1
	}

	result := a + b + carryVal

	// Set flags
	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.ClearFlag(FlagN)
	c.Registers.SetFlagTo(FlagH, (a&0x0F)+(b&0x0F)+carryVal > 0x0F)
	c.Registers.SetFlagTo(FlagC, uint16(a)+uint16(b)+uint16(carryVal) > 0xFF)

	return result
}

// sub8 performs 8-bit subtraction and sets flags.
func (c *CPU) sub8(a, b uint8, carry bool) uint8 {
	carryVal := uint8(0)
	if carry && c.Registers.CarryFlag() {
		carryVal = 1
	}

	result := a - b - carryVal

	// Set flags
	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.SetFlag(FlagN)
	c.Registers.SetFlagTo(FlagH, (a&0x0F) < (b&0x0F)+carryVal)
	c.Registers.SetFlagTo(FlagC, uint16(a) < uint16(b)+uint16(carryVal))

	return result
}

// add16 performs 16-bit addition and sets flags (used for ADD HL, rr).
func (c *CPU) add16(a, b uint16) uint16 {
	result := a + b

	// For 16-bit ADD, only N, H, C are affected (Z is not affected)
	c.Registers.ClearFlag(FlagN)
	c.Registers.SetFlagTo(FlagH, (a&0x0FFF)+(b&0x0FFF) > 0x0FFF)
	c.Registers.SetFlagTo(FlagC, uint32(a)+uint32(b) > 0xFFFF)

	return result
}

// and performs bitwise AND and sets flags.
func (c *CPU) and(value uint8) uint8 {
	result := c.Registers.A & value

	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.ClearFlag(FlagN)
	c.Registers.SetFlag(FlagH)
	c.Registers.ClearFlag(FlagC)

	return result
}

// or performs bitwise OR and sets flags.
func (c *CPU) or(value uint8) uint8 {
	result := c.Registers.A | value

	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.ClearFlag(FlagN)
	c.Registers.ClearFlag(FlagH)
	c.Registers.ClearFlag(FlagC)

	return result
}

// xor performs bitwise XOR and sets flags.
func (c *CPU) xor(value uint8) uint8 {
	result := c.Registers.A ^ value

	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.ClearFlag(FlagN)
	c.Registers.ClearFlag(FlagH)
	c.Registers.ClearFlag(FlagC)

	return result
}

// cp performs compare (subtraction without storing result) and sets flags.
func (c *CPU) cp(value uint8) {
	c.sub8(c.Registers.A, value, false)
}

// inc8 increments an 8-bit value and sets flags.
func (c *CPU) inc8(value uint8) uint8 {
	result := value + 1

	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.ClearFlag(FlagN)
	c.Registers.SetFlagTo(FlagH, (value&0x0F) == 0x0F)
	// Carry flag not affected

	return result
}

// dec8 decrements an 8-bit value and sets flags.
func (c *CPU) dec8(value uint8) uint8 {
	result := value - 1

	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.SetFlag(FlagN)
	c.Registers.SetFlagTo(FlagH, (value&0x0F) == 0)
	// Carry flag not affected

	return result
}

// Rotate and shift helpers

// rlc rotates left through carry.
func (c *CPU) rlc(value uint8) uint8 {
	carry := (value & 0x80) >> 7
	result := (value << 1) | carry

	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.ClearFlag(FlagN)
	c.Registers.ClearFlag(FlagH)
	c.Registers.SetFlagTo(FlagC, carry == 1)

	return result
}

// rl rotates left.
func (c *CPU) rl(value uint8) uint8 {
	carry := uint8(0)
	if c.Registers.CarryFlag() {
		carry = 1
	}
	newCarry := (value & 0x80) >> 7
	result := (value << 1) | carry

	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.ClearFlag(FlagN)
	c.Registers.ClearFlag(FlagH)
	c.Registers.SetFlagTo(FlagC, newCarry == 1)

	return result
}

// rrc rotates right through carry.
func (c *CPU) rrc(value uint8) uint8 {
	carry := value & 0x01
	result := (value >> 1) | (carry << 7)

	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.ClearFlag(FlagN)
	c.Registers.ClearFlag(FlagH)
	c.Registers.SetFlagTo(FlagC, carry == 1)

	return result
}

// rr rotates right.
func (c *CPU) rr(value uint8) uint8 {
	carry := uint8(0)
	if c.Registers.CarryFlag() {
		carry = 1
	}
	newCarry := value & 0x01
	result := (value >> 1) | (carry << 7)

	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.ClearFlag(FlagN)
	c.Registers.ClearFlag(FlagH)
	c.Registers.SetFlagTo(FlagC, newCarry == 1)

	return result
}

// sla shifts left arithmetic.
func (c *CPU) sla(value uint8) uint8 {
	carry := (value & 0x80) >> 7
	result := value << 1

	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.ClearFlag(FlagN)
	c.Registers.ClearFlag(FlagH)
	c.Registers.SetFlagTo(FlagC, carry == 1)

	return result
}

// sra shifts right arithmetic (preserves sign bit).
func (c *CPU) sra(value uint8) uint8 {
	carry := value & 0x01
	result := (value >> 1) | (value & 0x80)

	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.ClearFlag(FlagN)
	c.Registers.ClearFlag(FlagH)
	c.Registers.SetFlagTo(FlagC, carry == 1)

	return result
}

// srl shifts right logical.
func (c *CPU) srl(value uint8) uint8 {
	carry := value & 0x01
	result := value >> 1

	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.ClearFlag(FlagN)
	c.Registers.ClearFlag(FlagH)
	c.Registers.SetFlagTo(FlagC, carry == 1)

	return result
}

// swap swaps upper and lower nibbles.
func (c *CPU) swap(value uint8) uint8 {
	result := (value << 4) | (value >> 4)

	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.ClearFlag(FlagN)
	c.Registers.ClearFlag(FlagH)
	c.Registers.ClearFlag(FlagC)

	return result
}

// bit tests a bit.
func (c *CPU) bit(value uint8, bit uint8) {
	result := value & (1 << bit)

	c.Registers.SetFlagTo(FlagZ, result == 0)
	c.Registers.ClearFlag(FlagN)
	c.Registers.SetFlag(FlagH)
	// Carry flag not affected
}

// checkCondition checks jump/call conditions.
//
//nolint:unused // Will be used for conditional jumps in opcode implementation
func (c *CPU) checkCondition(cond uint8) bool {
	switch cond {
	case 0: // NZ - Not Zero
		return !c.Registers.ZeroFlag()
	case 1: // Z - Zero
		return c.Registers.ZeroFlag()
	case 2: // NC - Not Carry
		return !c.Registers.CarryFlag()
	case 3: // C - Carry
		return c.Registers.CarryFlag()
	default:
		return false
	}
}
