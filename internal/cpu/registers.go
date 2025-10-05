package cpu

// Flags represents CPU flag register bits.
const (
	FlagZ uint8 = 0b10000000 // Zero flag (bit 7)
	FlagN uint8 = 0b01000000 // Subtraction flag (bit 6)
	FlagH uint8 = 0b00100000 // Half-carry flag (bit 5)
	FlagC uint8 = 0b00010000 // Carry flag (bit 4)
)

// Registers represents the SM83 CPU registers.
type Registers struct {
	A  uint8  // Accumulator
	F  uint8  // Flags (only upper 4 bits used)
	B  uint8  // General purpose
	C  uint8  // General purpose
	D  uint8  // General purpose
	E  uint8  // General purpose
	H  uint8  // General purpose (high byte of HL pointer)
	L  uint8  // General purpose (low byte of HL pointer)
	SP uint16 // Stack pointer
	PC uint16 // Program counter
}

// NewRegisters creates a new Registers instance with default values.
func NewRegisters() *Registers {
	return &Registers{
		A:  0x01,
		F:  0xB0,
		B:  0x00,
		C:  0x13,
		D:  0x00,
		E:  0xD8,
		H:  0x01,
		L:  0x4D,
		SP: 0xFFFE,
		PC: 0x0100,
	}
}

// 16-bit register pair getters

// AF returns the 16-bit AF register pair.
func (r *Registers) AF() uint16 {
	return uint16(r.A)<<8 | uint16(r.F)
}

// BC returns the 16-bit BC register pair.
func (r *Registers) BC() uint16 {
	return uint16(r.B)<<8 | uint16(r.C)
}

// DE returns the 16-bit DE register pair.
func (r *Registers) DE() uint16 {
	return uint16(r.D)<<8 | uint16(r.E)
}

// HL returns the 16-bit HL register pair.
func (r *Registers) HL() uint16 {
	return uint16(r.H)<<8 | uint16(r.L)
}

// 16-bit register pair setters

// SetAF sets the 16-bit AF register pair.
func (r *Registers) SetAF(value uint16) {
	r.A = uint8(value >> 8)   //nolint:gosec // G115: Intentional byte extraction from 16-bit register
	r.F = uint8(value) & 0xF0 //nolint:gosec // G115: Lower 4 bits always 0
}

// SetBC sets the 16-bit BC register pair.
func (r *Registers) SetBC(value uint16) {
	r.B = uint8(value >> 8) //nolint:gosec // G115: Intentional byte extraction from 16-bit register
	r.C = uint8(value)      //nolint:gosec // G115: Intentional byte extraction from 16-bit register
}

// SetDE sets the 16-bit DE register pair.
func (r *Registers) SetDE(value uint16) {
	r.D = uint8(value >> 8) //nolint:gosec // G115: Intentional byte extraction from 16-bit register
	r.E = uint8(value)      //nolint:gosec // G115: Intentional byte extraction from 16-bit register
}

// SetHL sets the 16-bit HL register pair.
func (r *Registers) SetHL(value uint16) {
	r.H = uint8(value >> 8) //nolint:gosec // G115: Intentional byte extraction from 16-bit register
	r.L = uint8(value)      //nolint:gosec // G115: Intentional byte extraction from 16-bit register
}

// Flag operations

// GetFlag checks if a flag is set.
func (r *Registers) GetFlag(flag uint8) bool {
	return r.F&flag != 0
}

// SetFlag sets a flag to 1.
func (r *Registers) SetFlag(flag uint8) {
	r.F |= flag
}

// ClearFlag sets a flag to 0.
func (r *Registers) ClearFlag(flag uint8) {
	r.F &^= flag
}

// SetFlagTo sets a flag to a specific boolean value.
func (r *Registers) SetFlagTo(flag uint8, value bool) {
	if value {
		r.SetFlag(flag)
	} else {
		r.ClearFlag(flag)
	}
}

// Individual flag getters

// ZeroFlag returns the Zero flag state.
func (r *Registers) ZeroFlag() bool {
	return r.GetFlag(FlagZ)
}

// SubtractFlag returns the Subtract flag state.
func (r *Registers) SubtractFlag() bool {
	return r.GetFlag(FlagN)
}

// HalfCarryFlag returns the Half-carry flag state.
func (r *Registers) HalfCarryFlag() bool {
	return r.GetFlag(FlagH)
}

// CarryFlag returns the Carry flag state.
func (r *Registers) CarryFlag() bool {
	return r.GetFlag(FlagC)
}
