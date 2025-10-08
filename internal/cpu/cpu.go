// Package cpu implements the Sharp SM83 CPU emulation for the Game Boy.
package cpu

// Interrupt bit positions in IE/IF registers.
const (
	InterruptVBlank uint8 = 0 // V-Blank interrupt (highest priority)
	InterruptSTAT   uint8 = 1 // LCD STAT interrupt
	InterruptTimer  uint8 = 2 // Timer interrupt
	InterruptSerial uint8 = 3 // Serial interrupt
	InterruptJoypad uint8 = 4 // Joypad interrupt (lowest priority)
)

// Interrupt handler addresses.
var interruptHandlers = [5]uint16{
	0x0040, // V-Blank
	0x0048, // LCD STAT
	0x0050, // Timer
	0x0058, // Serial
	0x0060, // Joypad
}

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

	// Pending IME for EI instruction (delayed enable).
	// The EI instruction enables interrupts AFTER the next instruction executes.
	// This flag tracks that we need to set IME=true after the current instruction completes.
	pendingIME bool

	// Halt and stop states
	halted  bool
	stopped bool

	// HALT bug: when HALT is executed with IME=0 and an interrupt pending,
	// the PC doesn't increment after the next instruction fetch, causing
	// the first byte to be read twice
	haltBug bool

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
	// Check for interrupts before executing instruction
	if interruptCycles := c.checkInterrupts(); interruptCycles > 0 {
		c.Cycles += uint64(interruptCycles)
		return interruptCycles
	}

	// Handle halt state
	if c.halted {
		// Check if interrupt pending (will exit HALT)
		ie := c.Memory.Read(0xFFFF)
		ifReg := c.Memory.Read(0xFF0F)
		if (ie & ifReg & 0x1F) != 0 {
			c.halted = false

			// PC is currently at the HALT instruction (HALT decremented it)
			// Move PC forward to point to the instruction after HALT
			c.Registers.PC++

			// HALT bug: if IME=0 and interrupt pending, PC doesn't increment after next fetch
			// This causes the byte AFTER HALT to be read twice.
			//
			// At this point, PC points to the byte after HALT.
			// The next Step() will:
			//   1. fetchByte() reads the byte after HALT, PC doesn't increment (haltBug=true)
			//   2. Execute that instruction
			//   3. Next fetchByte() reads the same byte again, PC increments normally
			//   4. Execute the same instruction again (or use as operand if 2-byte)
			//
			// For 2-byte instructions, the opcode byte is used as both opcode and operand.
			if !c.IME {
				c.haltBug = true
			}
		}
		// Consume 1 M-cycle while halted (4 T-cycles)
		c.Cycles += 4
		return 4
	}

	// Fetch instruction
	opcode := c.fetchByte()

	// Clear haltBug flag after opcode fetch but before executing
	// This ensures:
	// 1. The flag is still set during the opcode fetch (preventing PC increment)
	// 2. The flag is clear for operand fetches in multi-byte instructions
	// 3. The flag is clear when executing HALT again (preventing double PC decrement)
	if c.haltBug {
		c.haltBug = false
	}

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

	// Handle delayed IME from EI instruction
	if c.pendingIME {
		c.IME = true
		c.pendingIME = false
	}

	return cycles
}

// fetchByte fetches the next byte from memory and increments PC.
func (c *CPU) fetchByte() uint8 {
	value := c.Memory.Read(c.Registers.PC)

	// HALT bug: when haltBug is active, the PC doesn't increment on the first fetch,
	// causing the byte to be read again.
	// Note: We don't clear the flag here - it's cleared after instruction execution
	// in Step(). This prevents double PC decrements when the bugged byte is another HALT.
	if !c.haltBug {
		c.Registers.PC++
	}

	return value
}

// fetchWord fetches the next word (16-bit) from memory and increments PC.
func (c *CPU) fetchWord() uint16 {
	low := uint16(c.fetchByte())
	high := uint16(c.fetchByte())
	return high<<8 | low
}

// push pushes a 16-bit value onto the stack.
// Note: SP is decremented first (pre-decrement), then values are written.
func (c *CPU) push(value uint16) {
	c.Registers.SP -= 2
	c.Memory.Write(c.Registers.SP, uint8(value))      //nolint:gosec // G115: Intentional byte extraction from 16-bit value
	c.Memory.Write(c.Registers.SP+1, uint8(value>>8)) //nolint:gosec // G115: Intentional byte extraction from 16-bit value
}

// pop pops a 16-bit value from the stack.
// Note: Values are read first, then SP is incremented (post-increment).
// This asymmetry with push() is intentional and matches Game Boy hardware behavior.
func (c *CPU) pop() uint16 {
	low := uint16(c.Memory.Read(c.Registers.SP))
	high := uint16(c.Memory.Read(c.Registers.SP + 1))
	c.Registers.SP += 2
	return high<<8 | low
}

// checkInterrupts checks for pending interrupts and services them if IME is enabled.
// Returns the number of cycles consumed (20 if interrupt serviced, 0 otherwise).
func (c *CPU) checkInterrupts() uint8 {
	// Interrupts only serviced if IME is enabled
	if !c.IME {
		return 0
	}

	// Read IE and IF registers
	ie := c.Memory.Read(0xFFFF)    // Interrupt Enable
	ifReg := c.Memory.Read(0xFF0F) // Interrupt Flag

	// Check for pending interrupts (IE & IF & 0x1F)
	pending := ie & ifReg & 0x1F

	if pending == 0 {
		return 0
	}

	// Find highest priority interrupt (lowest bit number)
	for bit := uint8(0); bit < 5; bit++ {
		if pending&(1<<bit) != 0 {
			c.serviceInterrupt(bit)
			return 20 // Interrupt service takes 5 M-cycles = 20 clock cycles
		}
	}

	return 0
}

// serviceInterrupt services an interrupt.
func (c *CPU) serviceInterrupt(bit uint8) {
	// Exit HALT state if active
	// Note: This is defensive - normally interrupts are only serviced in Step() which
	// checks interrupts before the halted state, so halted should already be false.
	// However, we clear it here for safety in case serviceInterrupt is called directly.
	c.halted = false

	// Disable interrupts
	c.IME = false
	c.pendingIME = false

	// Clear IF bit for this interrupt
	ifReg := c.Memory.Read(0xFF0F)
	c.Memory.Write(0xFF0F, ifReg&^(1<<bit))

	// Push PC onto stack
	c.push(c.Registers.PC)

	// Jump to interrupt handler address
	c.Registers.PC = interruptHandlers[bit]
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
