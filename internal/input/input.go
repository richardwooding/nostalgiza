// Package input implements Game Boy joypad input handling.
package input

// Joypad represents the Game Boy joypad state and P1/JOYP register.
type Joypad struct {
	// Selection bits (written by CPU)
	selectAction    bool // P15 (0=select action buttons)
	selectDirection bool // P14 (0=select direction buttons)

	// Button states (true = pressed)
	buttonA      bool
	buttonB      bool
	buttonStart  bool
	buttonSelect bool
	buttonUp     bool
	buttonDown   bool
	buttonLeft   bool
	buttonRight  bool

	// Interrupt callback
	requestInterrupt func(uint8)
}

// New creates a new Joypad instance.
func New(requestInterrupt func(uint8)) *Joypad {
	return &Joypad{
		selectAction:     true, // Not selected (1)
		selectDirection:  true, // Not selected (1)
		requestInterrupt: requestInterrupt,
	}
}

// Read returns the P1/JOYP register value (0xFF00).
func (j *Joypad) Read() uint8 {
	result := uint8(0xC0) // Upper 2 bits always 1

	// Set selection bits
	if j.selectAction {
		result |= 0x20 // P15
	}
	if j.selectDirection {
		result |= 0x10 // P14
	}

	// Initialize button bits as all released (1)
	buttonBits := uint8(0x0F)

	// If action buttons selected (P15=0)
	if !j.selectAction {
		if j.buttonStart {
			buttonBits &^= 0x08 // Bit 3
		}
		if j.buttonSelect {
			buttonBits &^= 0x04 // Bit 2
		}
		if j.buttonB {
			buttonBits &^= 0x02 // Bit 1
		}
		if j.buttonA {
			buttonBits &^= 0x01 // Bit 0
		}
	}

	// If direction buttons selected (P14=0)
	if !j.selectDirection {
		if j.buttonDown {
			buttonBits &^= 0x08 // Bit 3
		}
		if j.buttonUp {
			buttonBits &^= 0x04 // Bit 2
		}
		if j.buttonLeft {
			buttonBits &^= 0x02 // Bit 1
		}
		if j.buttonRight {
			buttonBits &^= 0x01 // Bit 0
		}
	}

	result |= buttonBits
	return result
}

// Write updates the P1/JOYP register (only bits 4-5 are writable).
func (j *Joypad) Write(value uint8) {
	j.selectAction = (value & 0x20) != 0
	j.selectDirection = (value & 0x10) != 0
}

// PressButton sets a button as pressed and requests joypad interrupt on state change.
// Only triggers interrupt when button transitions from released to pressed.
func (j *Joypad) PressButton(button string) {
	// Check current state before update
	wasPressed := false

	switch button {
	case "A":
		wasPressed = j.buttonA
		j.buttonA = true
	case "B":
		wasPressed = j.buttonB
		j.buttonB = true
	case "Start":
		wasPressed = j.buttonStart
		j.buttonStart = true
	case "Select":
		wasPressed = j.buttonSelect
		j.buttonSelect = true
	case "Up":
		wasPressed = j.buttonUp
		if !j.buttonDown { // Block opposite directions
			j.buttonUp = true
		}
	case "Down":
		wasPressed = j.buttonDown
		if !j.buttonUp { // Block opposite directions
			j.buttonDown = true
		}
	case "Left":
		wasPressed = j.buttonLeft
		if !j.buttonRight { // Block opposite directions
			j.buttonLeft = true
		}
	case "Right":
		wasPressed = j.buttonRight
		if !j.buttonLeft { // Block opposite directions
			j.buttonRight = true
		}
	}

	// Only request interrupt on state transition (released -> pressed)
	if !wasPressed && j.requestInterrupt != nil {
		j.requestInterrupt(4)
	}
}

// ReleaseButton sets a button as released.
func (j *Joypad) ReleaseButton(button string) {
	switch button {
	case "A":
		j.buttonA = false
	case "B":
		j.buttonB = false
	case "Start":
		j.buttonStart = false
	case "Select":
		j.buttonSelect = false
	case "Up":
		j.buttonUp = false
	case "Down":
		j.buttonDown = false
	case "Left":
		j.buttonLeft = false
	case "Right":
		j.buttonRight = false
	}
}
