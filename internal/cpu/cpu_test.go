package cpu

import (
	"testing"
)

// mockMemory is a simple memory implementation for testing.
type mockMemory struct {
	data [0x10000]uint8
}

func (m *mockMemory) Read(addr uint16) uint8 {
	return m.data[addr]
}

func (m *mockMemory) Write(addr uint16, value uint8) {
	m.data[addr] = value
}

func newMockMemory() *mockMemory {
	return &mockMemory{}
}

// setupCPU creates a CPU and mock memory for testing.
func setupCPU() (*CPU, *mockMemory) {
	mem := newMockMemory()
	cpu := New(mem)
	return cpu, mem
}

func TestRegisters(t *testing.T) {
	r := NewRegisters()

	// Test 16-bit register pairs
	r.SetBC(0x1234)
	if r.BC() != 0x1234 {
		t.Errorf("BC() = %04X, want 0x1234", r.BC())
	}
	if r.B != 0x12 || r.C != 0x34 {
		t.Errorf("B = %02X, C = %02X, want 0x12, 0x34", r.B, r.C)
	}

	r.SetDE(0x5678)
	if r.DE() != 0x5678 {
		t.Errorf("DE() = %04X, want 0x5678", r.DE())
	}

	r.SetHL(0x9ABC)
	if r.HL() != 0x9ABC {
		t.Errorf("HL() = %04X, want 0x9ABC", r.HL())
	}

	// Test flags
	r.SetFlag(FlagZ)
	if !r.ZeroFlag() {
		t.Error("Zero flag should be set")
	}

	r.ClearFlag(FlagZ)
	if r.ZeroFlag() {
		t.Error("Zero flag should be clear")
	}

	// Test F register lower bits always 0
	r.SetAF(0x12FF)
	if r.F != 0xF0 {
		t.Errorf("F = %02X, want 0xF0 (lower 4 bits should be 0)", r.F)
	}
}

func TestNOP(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)

	// NOP instruction
	mem.data[0x0100] = 0x00

	cycles := cpu.Step()
	if cycles != 4 {
		t.Errorf("NOP cycles = %d, want 4", cycles)
	}
	if cpu.Registers.PC != 0x0101 {
		t.Errorf("PC = %04X, want 0x0101", cpu.Registers.PC)
	}
}

func TestLD(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)

	// LD B, 0x42
	mem.data[0x0100] = 0x06 // LD B, n
	mem.data[0x0101] = 0x42

	cycles := cpu.Step()
	if cycles != 8 {
		t.Errorf("LD B, n cycles = %d, want 8", cycles)
	}
	if cpu.Registers.B != 0x42 {
		t.Errorf("B = %02X, want 0x42", cpu.Registers.B)
	}
	if cpu.Registers.PC != 0x0102 {
		t.Errorf("PC = %04X, want 0x0102", cpu.Registers.PC)
	}
}

func TestADD8(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)

	// Test ADD A, n (0x3A + 0x0C = 0x46)
	// Lower nibbles: 0xA + 0xC = 0x16, half-carry should be set
	cpu.Registers.A = 0x3A
	mem.data[0x0100] = 0xC6 // ADD A, n
	mem.data[0x0101] = 0x0C

	cpu.Step()

	if cpu.Registers.A != 0x46 {
		t.Errorf("A = %02X, want 0x46", cpu.Registers.A)
	}
	if cpu.Registers.ZeroFlag() {
		t.Error("Zero flag should not be set")
	}
	if cpu.Registers.SubtractFlag() {
		t.Error("Subtract flag should not be set")
	}
	if !cpu.Registers.HalfCarryFlag() {
		t.Error("Half-carry flag should be set (0xA + 0xC > 0xF)")
	}
	if cpu.Registers.CarryFlag() {
		t.Error("Carry flag should not be set")
	}

	// Test ADD with carry
	cpu.Registers.PC = 0x0100
	cpu.Registers.A = 0xFF
	mem.data[0x0100] = 0xC6 // ADD A, n
	mem.data[0x0101] = 0x01

	cpu.Step()

	if cpu.Registers.A != 0x00 {
		t.Errorf("A = %02X, want 0x00", cpu.Registers.A)
	}
	if !cpu.Registers.ZeroFlag() {
		t.Error("Zero flag should be set")
	}
	if !cpu.Registers.HalfCarryFlag() {
		t.Error("Half-carry flag should be set")
	}
	if !cpu.Registers.CarryFlag() {
		t.Error("Carry flag should be set")
	}
}

func TestSUB8(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)

	// Test SUB n
	cpu.Registers.A = 0x3E
	mem.data[0x0100] = 0xD6 // SUB n
	mem.data[0x0101] = 0x0F

	cpu.Step()

	if cpu.Registers.A != 0x2F {
		t.Errorf("A = %02X, want 0x2F", cpu.Registers.A)
	}
	if !cpu.Registers.SubtractFlag() {
		t.Error("Subtract flag should be set")
	}
	if !cpu.Registers.HalfCarryFlag() {
		t.Error("Half-carry flag should be set")
	}
	if cpu.Registers.CarryFlag() {
		t.Error("Carry flag should not be set")
	}
}

func TestAND(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)

	// Test AND n
	cpu.Registers.A = 0x5A
	mem.data[0x0100] = 0xE6 // AND n
	mem.data[0x0101] = 0x3F

	cpu.Step()

	if cpu.Registers.A != 0x1A {
		t.Errorf("A = %02X, want 0x1A", cpu.Registers.A)
	}
	if !cpu.Registers.HalfCarryFlag() {
		t.Error("Half-carry flag should be set for AND")
	}
}

func TestXOR(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)

	// Test XOR A (common pattern to zero A)
	cpu.Registers.A = 0x42
	mem.data[0x0100] = 0xAF // XOR A

	cpu.Step()

	if cpu.Registers.A != 0x00 {
		t.Errorf("A = %02X, want 0x00", cpu.Registers.A)
	}
	if !cpu.Registers.ZeroFlag() {
		t.Error("Zero flag should be set")
	}
	if cpu.Registers.SubtractFlag() {
		t.Error("Subtract flag should not be set")
	}
	if cpu.Registers.HalfCarryFlag() {
		t.Error("Half-carry flag should not be set")
	}
	if cpu.Registers.CarryFlag() {
		t.Error("Carry flag should not be set")
	}
}

func TestINCDEC(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)

	// Test INC B
	cpu.Registers.B = 0x0F
	mem.data[0x0100] = 0x04 // INC B

	cpu.Step()

	if cpu.Registers.B != 0x10 {
		t.Errorf("B = %02X, want 0x10", cpu.Registers.B)
	}
	if !cpu.Registers.HalfCarryFlag() {
		t.Error("Half-carry flag should be set")
	}

	// Test DEC B
	cpu.Registers.PC = 0x0100
	cpu.Registers.B = 0x01
	mem.data[0x0100] = 0x05 // DEC B

	cpu.Step()

	if cpu.Registers.B != 0x00 {
		t.Errorf("B = %02X, want 0x00", cpu.Registers.B)
	}
	if !cpu.Registers.ZeroFlag() {
		t.Error("Zero flag should be set")
	}
	if !cpu.Registers.SubtractFlag() {
		t.Error("Subtract flag should be set")
	}
}

func TestJP(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)

	// Test JP nn
	mem.data[0x0100] = 0xC3 // JP nn
	mem.data[0x0101] = 0x50
	mem.data[0x0102] = 0x01 // Address 0x0150

	cpu.Step()

	if cpu.Registers.PC != 0x0150 {
		t.Errorf("PC = %04X, want 0x0150", cpu.Registers.PC)
	}
}

func TestJR(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)

	// Test JR n (positive offset)
	mem.data[0x0100] = 0x18 // JR n
	mem.data[0x0101] = 0x05 // +5

	cpu.Step()

	// PC is at 0x0102 after fetching, +5 = 0x0107
	if cpu.Registers.PC != 0x0107 {
		t.Errorf("PC = %04X, want 0x0107", cpu.Registers.PC)
	}

	// Test JR n (negative offset)
	cpu.Registers.PC = 0x0100
	mem.data[0x0100] = 0x18 // JR n
	mem.data[0x0101] = 0xFE // -2

	cpu.Step()

	// PC is at 0x0102 after fetching, -2 = 0x0100
	if cpu.Registers.PC != 0x0100 {
		t.Errorf("PC = %04X, want 0x0100", cpu.Registers.PC)
	}
}

func TestCALLRET(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)
	cpu.Registers.SP = 0xFFFE

	// Test CALL nn
	mem.data[0x0100] = 0xCD // CALL nn
	mem.data[0x0101] = 0x50
	mem.data[0x0102] = 0x01 // Address 0x0150

	cpu.Step()

	if cpu.Registers.PC != 0x0150 {
		t.Errorf("PC = %04X, want 0x0150", cpu.Registers.PC)
	}
	if cpu.Registers.SP != 0xFFFC {
		t.Errorf("SP = %04X, want 0xFFFC", cpu.Registers.SP)
	}

	// Check return address on stack
	returnAddr := uint16(mem.data[0xFFFC]) | uint16(mem.data[0xFFFD])<<8
	if returnAddr != 0x0103 {
		t.Errorf("Return address = %04X, want 0x0103", returnAddr)
	}

	// Test RET
	mem.data[0x0150] = 0xC9 // RET

	cpu.Step()

	if cpu.Registers.PC != 0x0103 {
		t.Errorf("PC = %04X, want 0x0103", cpu.Registers.PC)
	}
	if cpu.Registers.SP != 0xFFFE {
		t.Errorf("SP = %04X, want 0xFFFE", cpu.Registers.SP)
	}
}

func TestPUSHPOP(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)
	cpu.Registers.SP = 0xFFFE

	// Test PUSH BC
	cpu.Registers.SetBC(0x1234)
	mem.data[0x0100] = 0xC5 // PUSH BC

	cpu.Step()

	if cpu.Registers.SP != 0xFFFC {
		t.Errorf("SP = %04X, want 0xFFFC", cpu.Registers.SP)
	}

	// Test POP DE
	cpu.Registers.PC = 0x0100
	mem.data[0x0100] = 0xD1 // POP DE

	cpu.Step()

	if cpu.Registers.DE() != 0x1234 {
		t.Errorf("DE = %04X, want 0x1234", cpu.Registers.DE())
	}
	if cpu.Registers.SP != 0xFFFE {
		t.Errorf("SP = %04X, want 0xFFFE", cpu.Registers.SP)
	}
}

func TestCBRotate(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)

	// Test RLC B
	cpu.Registers.B = 0x85 // 10000101
	mem.data[0x0100] = 0xCB
	mem.data[0x0101] = 0x00 // RLC B

	cpu.Step()

	if cpu.Registers.B != 0x0B { // 00001011
		t.Errorf("B = %02X, want 0x0B", cpu.Registers.B)
	}
	if !cpu.Registers.CarryFlag() {
		t.Error("Carry flag should be set")
	}
}

func TestCBBit(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)

	// Test BIT 7, A
	cpu.Registers.A = 0x80
	mem.data[0x0100] = 0xCB
	mem.data[0x0101] = 0x7F // BIT 7, A

	cpu.Step()

	if cpu.Registers.ZeroFlag() {
		t.Error("Zero flag should not be set (bit 7 is 1)")
	}
	if !cpu.Registers.HalfCarryFlag() {
		t.Error("Half-carry flag should be set for BIT")
	}

	// Test BIT 6, A
	cpu.Registers.PC = 0x0100
	cpu.Registers.A = 0x80
	mem.data[0x0100] = 0xCB
	mem.data[0x0101] = 0x77 // BIT 6, A

	cpu.Step()

	if !cpu.Registers.ZeroFlag() {
		t.Error("Zero flag should be set (bit 6 is 0)")
	}
}

func TestCBSetRes(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)

	// Test SET 3, B
	cpu.Registers.B = 0x00
	mem.data[0x0100] = 0xCB
	mem.data[0x0101] = 0xD8 // SET 3, B

	cpu.Step()

	if cpu.Registers.B != 0x08 {
		t.Errorf("B = %02X, want 0x08", cpu.Registers.B)
	}

	// Test RES 3, B
	cpu.Registers.PC = 0x0100
	cpu.Registers.B = 0xFF
	mem.data[0x0100] = 0xCB
	mem.data[0x0101] = 0x98 // RES 3, B

	cpu.Step()

	if cpu.Registers.B != 0xF7 {
		t.Errorf("B = %02X, want 0xF7", cpu.Registers.B)
	}
}

func TestHALT(t *testing.T) {
	mem := newMockMemory()
	cpu := New(mem)

	// Test HALT
	mem.data[0x0100] = 0x76 // HALT

	cpu.Step()

	if !cpu.halted {
		t.Error("CPU should be halted")
	}

	// Next step should do nothing
	cycles := cpu.Step()
	if cycles != 4 {
		t.Errorf("Halted CPU cycles = %d, want 4", cycles)
	}
}

func TestDAA(t *testing.T) {
	cpu, mem := setupCPU()

	tests := []struct {
		name     string
		a        uint8
		flags    uint8 // Initial flags
		expected uint8
		expectZ  bool
		expectH  bool // H should be cleared
		expectC  bool
		expectN  bool // Should preserve N flag
	}{
		// After addition (N=0)
		{"ADD: 0x09 + 0x08 = 0x11, no adjust", 0x11, 0x00, 0x11, false, false, false, false},
		{"ADD: 0x09 + 0x09 = 0x12 (H set), adjust +6", 0x12, FlagH, 0x18, false, false, false, false},
		{"ADD: Lower nibble >9, adjust +6", 0x1A, 0x00, 0x20, false, false, false, false},
		{"ADD: Upper nibble >9, adjust +60", 0xA3, 0x00, 0x03, false, false, true, false},
		{"ADD: 0x99 + 0x99 = 0x32 (C set), adjust +60", 0x32, FlagC, 0x92, false, false, true, false},
		{"ADD: 0x99 + 0x99 = 0x32 (C+H set), adjust +66", 0x32, FlagC | FlagH, 0x98, false, false, true, false},
		{"ADD: result 0x00 after adjust", 0x9A, FlagC, 0x00, true, false, true, false},

		// After subtraction (N=1)
		{"SUB: 0x46 - 0x08 = 0x3E, no adjust", 0x3E, FlagN, 0x3E, false, false, false, true},
		{"SUB: 0x40 - 0x09 = 0x37 (H set), adjust -6", 0x37, FlagN | FlagH, 0x31, false, false, false, true},
		{"SUB: result with C flag set", 0x37, FlagN | FlagC, 0xD7, false, false, true, true},
		{"SUB: result with C+H flags", 0x37, FlagN | FlagC | FlagH, 0xD1, false, false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up initial state
			cpu.Registers.A = tt.a
			cpu.Registers.F = tt.flags
			cpu.Registers.PC = 0x0100

			// DAA instruction
			mem.data[0x0100] = 0x27

			cpu.Step()

			if cpu.Registers.A != tt.expected {
				t.Errorf("A = 0x%02X, want 0x%02X", cpu.Registers.A, tt.expected)
			}
			if cpu.Registers.ZeroFlag() != tt.expectZ {
				t.Errorf("Z flag = %v, want %v", cpu.Registers.ZeroFlag(), tt.expectZ)
			}
			if cpu.Registers.HalfCarryFlag() != tt.expectH {
				t.Errorf("H flag = %v, want %v (should be cleared)", cpu.Registers.HalfCarryFlag(), tt.expectH)
			}
			if cpu.Registers.CarryFlag() != tt.expectC {
				t.Errorf("C flag = %v, want %v", cpu.Registers.CarryFlag(), tt.expectC)
			}
			if cpu.Registers.SubtractFlag() != tt.expectN {
				t.Errorf("N flag = %v, want %v (should preserve)", cpu.Registers.SubtractFlag(), tt.expectN)
			}
		})
	}
}

func TestConditionalJumps(t *testing.T) {
	cpu, mem := setupCPU()

	tests := []struct {
		name       string
		opcode     uint8
		offset     int8
		flags      uint8
		shouldJump bool
	}{
		// JR NZ (0x20)
		{"JR NZ with Z=0 (should jump)", 0x20, 5, 0x00, true},
		{"JR NZ with Z=1 (should not jump)", 0x20, 5, FlagZ, false},

		// JR Z (0x28)
		{"JR Z with Z=1 (should jump)", 0x28, 5, FlagZ, true},
		{"JR Z with Z=0 (should not jump)", 0x28, 5, 0x00, false},

		// JR NC (0x30)
		{"JR NC with C=0 (should jump)", 0x30, 5, 0x00, true},
		{"JR NC with C=1 (should not jump)", 0x30, 5, FlagC, false},

		// JR C (0x38)
		{"JR C with C=1 (should jump)", 0x38, 5, FlagC, true},
		{"JR C with C=0 (should not jump)", 0x38, 5, 0x00, false},

		// Test negative offset
		{"JR NZ backward (should jump)", 0x20, -3, 0x00, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu.Registers.PC = 0x0100
			cpu.Registers.F = tt.flags

			mem.data[0x0100] = tt.opcode
			mem.data[0x0101] = uint8(tt.offset) //nolint:gosec // G115: Intentional signed to unsigned conversion for test

			cycles := cpu.Step()

			expectedPC := uint16(0x0102)
			if tt.shouldJump {
				expectedPC = uint16(int32(0x0102) + int32(tt.offset)) //nolint:gosec // G115: Intentional conversion
				if cycles != 12 {
					t.Errorf("Cycles = %d, want 12 (taken)", cycles)
				}
			} else if cycles != 8 {
				t.Errorf("Cycles = %d, want 8 (not taken)", cycles)
			}

			if cpu.Registers.PC != expectedPC {
				t.Errorf("PC = 0x%04X, want 0x%04X", cpu.Registers.PC, expectedPC)
			}
		})
	}
}

func TestConditionalCalls(t *testing.T) {
	cpu, mem := setupCPU()

	tests := []struct {
		name       string
		opcode     uint8
		flags      uint8
		shouldCall bool
	}{
		// CALL NZ (0xC4)
		{"CALL NZ with Z=0 (should call)", 0xC4, 0x00, true},
		{"CALL NZ with Z=1 (should not call)", 0xC4, FlagZ, false},

		// CALL Z (0xCC)
		{"CALL Z with Z=1 (should call)", 0xCC, FlagZ, true},
		{"CALL Z with Z=0 (should not call)", 0xCC, 0x00, false},

		// CALL NC (0xD4)
		{"CALL NC with C=0 (should call)", 0xD4, 0x00, true},
		{"CALL NC with C=1 (should not call)", 0xD4, FlagC, false},

		// CALL C (0xDC)
		{"CALL C with C=1 (should call)", 0xDC, FlagC, true},
		{"CALL C with C=0 (should not call)", 0xDC, 0x00, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu.Registers.PC = 0x0100
			cpu.Registers.SP = 0xFFFE
			cpu.Registers.F = tt.flags

			mem.data[0x0100] = tt.opcode
			mem.data[0x0101] = 0x34 // Low byte of address
			mem.data[0x0102] = 0x12 // High byte of address

			cycles := cpu.Step()

			if tt.shouldCall { //nolint:nestif // Test validation complexity is acceptable
				// PC should be set to 0x1234
				if cpu.Registers.PC != 0x1234 {
					t.Errorf("PC = 0x%04X, want 0x1234", cpu.Registers.PC)
				}
				// Return address (0x0103) should be on stack
				if cpu.Registers.SP != 0xFFFC {
					t.Errorf("SP = 0x%04X, want 0xFFFC", cpu.Registers.SP)
				}
				if cycles != 24 {
					t.Errorf("Cycles = %d, want 24 (taken)", cycles)
				}
			} else {
				// PC should advance past instruction
				if cpu.Registers.PC != 0x0103 {
					t.Errorf("PC = 0x%04X, want 0x0103", cpu.Registers.PC)
				}
				// SP should not change
				if cpu.Registers.SP != 0xFFFE {
					t.Errorf("SP = 0x%04X, want 0xFFFE", cpu.Registers.SP)
				}
				if cycles != 12 {
					t.Errorf("Cycles = %d, want 12 (not taken)", cycles)
				}
			}
		})
	}
}

func TestConditionalReturns(t *testing.T) {
	cpu, mem := setupCPU()

	tests := []struct {
		name         string
		opcode       uint8
		flags        uint8
		shouldReturn bool
	}{
		// RET NZ (0xC0)
		{"RET NZ with Z=0 (should return)", 0xC0, 0x00, true},
		{"RET NZ with Z=1 (should not return)", 0xC0, FlagZ, false},

		// RET Z (0xC8)
		{"RET Z with Z=1 (should return)", 0xC8, FlagZ, true},
		{"RET Z with Z=0 (should not return)", 0xC8, 0x00, false},

		// RET NC (0xD0)
		{"RET NC with C=0 (should return)", 0xD0, 0x00, true},
		{"RET NC with C=1 (should not return)", 0xD0, FlagC, false},

		// RET C (0xD8)
		{"RET C with C=1 (should return)", 0xD8, FlagC, true},
		{"RET C with C=0 (should not return)", 0xD8, 0x00, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu.Registers.PC = 0x0100
			cpu.Registers.SP = 0xFFFC
			cpu.Registers.F = tt.flags

			// Set up return address on stack (0x1234)
			mem.data[0xFFFC] = 0x34
			mem.data[0xFFFD] = 0x12

			mem.data[0x0100] = tt.opcode

			cycles := cpu.Step()

			if tt.shouldReturn { //nolint:nestif // Test validation complexity is acceptable
				// PC should be set to return address
				if cpu.Registers.PC != 0x1234 {
					t.Errorf("PC = 0x%04X, want 0x1234", cpu.Registers.PC)
				}
				// SP should be popped
				if cpu.Registers.SP != 0xFFFE {
					t.Errorf("SP = 0x%04X, want 0xFFFE", cpu.Registers.SP)
				}
				if cycles != 20 {
					t.Errorf("Cycles = %d, want 20 (taken)", cycles)
				}
			} else {
				// PC should advance past instruction
				if cpu.Registers.PC != 0x0101 {
					t.Errorf("PC = 0x%04X, want 0x0101", cpu.Registers.PC)
				}
				// SP should not change
				if cpu.Registers.SP != 0xFFFC {
					t.Errorf("SP = 0x%04X, want 0xFFFC", cpu.Registers.SP)
				}
				if cycles != 8 {
					t.Errorf("Cycles = %d, want 8 (not taken)", cycles)
				}
			}
		})
	}
}
