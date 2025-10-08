// Package timer implements the Game Boy timer system.
//
// The timer system consists of:
//   - DIV: Divider register (increments at 16384 Hz)
//   - TIMA: Timer counter (increments at configurable rate)
//   - TMA: Timer modulo (value to reload into TIMA on overflow)
//   - TAC: Timer control (enable and clock select)
//
// The timer uses falling edge detection on specific bits of the internal
// DIV counter to increment TIMA at the selected frequency.
package timer

// InterruptCallback is the function type for timer interrupt requests.
type InterruptCallback func()

// Timer represents the Game Boy timer system.
type Timer struct {
	divCounter uint16 // Internal 16-bit counter (DIV is upper 8 bits)
	tima       uint8  // Timer counter ($FF05)
	tma        uint8  // Timer modulo ($FF06)
	tac        uint8  // Timer control ($FF07)

	enabled     bool  // Timer enable bit (TAC bit 2)
	clockSelect uint8 // Clock select bits (TAC bits 1-0)

	// Callback for timer interrupt
	requestInterrupt InterruptCallback
}

// Register addresses.
const (
	DIV  = 0xFF04
	TIMA = 0xFF05
	TMA  = 0xFF06
	TAC  = 0xFF07
)

// TAC register bits.
const (
	tacEnableBit = 0x04 // Bit 2: Timer enable
	tacClockMask = 0x03 // Bits 1-0: Clock select
)

// New creates a new Timer with the given interrupt callback.
func New(requestInterrupt InterruptCallback) *Timer {
	return &Timer{
		requestInterrupt: requestInterrupt,
	}
}

// Read reads a timer register.
func (t *Timer) Read(addr uint16) uint8 {
	switch addr {
	case DIV:
		return uint8(t.divCounter >> 8) //nolint:gosec // DIV is upper 8 bits
	case TIMA:
		return t.tima
	case TMA:
		return t.tma
	case TAC:
		return t.tac | 0xF8 // Upper 5 bits read as 1
	}
	return 0xFF
}

// Write writes to a timer register.
func (t *Timer) Write(addr uint16, value uint8) {
	switch addr {
	case DIV:
		// Any write resets DIV to 0
		// Check for falling edge before resetting (can trigger TIMA increment)
		if t.enabled {
			t.checkFallingEdge(t.divCounter, 0)
		}
		t.divCounter = 0

	case TIMA:
		t.tima = value

	case TMA:
		t.tma = value

	case TAC:
		oldTAC := t.tac
		t.tac = value & 0x07 // Only lower 3 bits are writable

		oldEnabled := t.enabled
		oldClockSelect := t.clockSelect

		t.enabled = (t.tac & tacEnableBit) != 0
		t.clockSelect = t.tac & tacClockMask

		// Check for falling edge when TAC changes
		if oldEnabled || t.enabled {
			// If enable state or clock select changed, check for falling edge
			if oldEnabled != t.enabled || oldClockSelect != t.clockSelect || oldTAC != t.tac {
				oldBit := t.getTimerBit(t.divCounter, oldEnabled, oldClockSelect)
				newBit := t.getTimerBit(t.divCounter, t.enabled, t.clockSelect)

				if oldBit && !newBit {
					t.incrementTIMA()
				}
			}
		}
	}
}

// Update advances the timer by the given number of CPU cycles.
func (t *Timer) Update(cycles uint16) {
	if !t.enabled {
		// Timer disabled, only update DIV
		t.divCounter += cycles
		return
	}

	// When timer is enabled, we need to detect all falling edges
	// Calculate falling edges mathematically instead of iterating
	startCounter := t.divCounter
	endCounter := t.divCounter + cycles

	// Count falling edges on the timer bit between start and end
	fallingEdges := t.countFallingEdges(startCounter, endCounter)

	// Update divCounter
	t.divCounter = endCounter

	// Increment TIMA for each falling edge
	for i := uint16(0); i < fallingEdges; i++ {
		t.incrementTIMA()
	}
}

// countFallingEdges counts the number of falling edges (1->0 transitions)
// on the timer bit as the counter increments from startCounter to endCounter.
func (t *Timer) countFallingEdges(startCounter, endCounter uint16) uint16 {
	if startCounter >= endCounter {
		return 0
	}

	// Get the bit position for the current clock select
	var bitPosition uint
	switch t.clockSelect {
	case 0: // 4096 Hz
		bitPosition = 9
	case 1: // 262144 Hz
		bitPosition = 3
	case 2: // 65536 Hz
		bitPosition = 5
	case 3: // 16384 Hz
		bitPosition = 7
	}

	// A falling edge occurs when the specified bit transitions from 1 to 0.
	// For bit N, the pattern repeats every 2^(N+1) cycles:
	// - Cycles [0, 2^N): bit = 0
	// - Cycles [2^N, 2^(N+1)): bit = 1
	// - At cycle 2^(N+1): bit = 0 again (falling edge)
	//
	// So falling edges occur at multiples of 2^(N+1).

	period := uint16(1 << (bitPosition + 1))

	// Find the first falling edge >= startCounter + 1 (since we increment from startCounter)
	// Falling edges are at 0, period, 2*period, 3*period, ...
	// We need the first multiple of period that is > startCounter

	firstEdge := ((startCounter / period) + 1) * period

	// Count how many multiples of period are in the range (startCounter, endCounter]
	if firstEdge > endCounter {
		return 0
	}

	// Count edges: firstEdge, firstEdge+period, firstEdge+2*period, ..., up to endCounter
	return (endCounter-firstEdge)/period + 1
}

// checkFallingEdge checks if a falling edge occurred on the selected timer bit.
func (t *Timer) checkFallingEdge(oldDiv, newDiv uint16) {
	oldBit := t.getTimerBit(oldDiv, t.enabled, t.clockSelect)
	newBit := t.getTimerBit(newDiv, t.enabled, t.clockSelect)

	if oldBit && !newBit {
		t.incrementTIMA()
	}
}

// getTimerBit returns the value of the timer bit for the given counter value.
func (t *Timer) getTimerBit(counter uint16, enabled bool, clockSelect uint8) bool {
	if !enabled {
		return false
	}

	// Determine which bit to check based on clock select
	var bitPosition uint
	switch clockSelect {
	case 0: // 4096 Hz
		bitPosition = 9
	case 1: // 262144 Hz
		bitPosition = 3
	case 2: // 65536 Hz
		bitPosition = 5
	case 3: // 16384 Hz
		bitPosition = 7
	}

	return (counter & (1 << bitPosition)) != 0
}

// incrementTIMA increments the timer counter and handles overflow.
func (t *Timer) incrementTIMA() {
	t.tima++

	if t.tima == 0 {
		// Overflow occurred
		t.tima = t.tma
		if t.requestInterrupt != nil {
			t.requestInterrupt()
		}
	}
}

// Reset resets the timer to initial state.
func (t *Timer) Reset() {
	t.divCounter = 0
	t.tima = 0
	t.tma = 0
	t.tac = 0
	t.enabled = false
	t.clockSelect = 0
}
