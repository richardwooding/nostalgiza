package input

import "testing"

func TestJoypadRead_NoButtonsPressed(t *testing.T) {
	j := New(nil)

	// Default state: nothing selected, no buttons pressed
	result := j.Read()

	// Upper 2 bits should be 1, selection bits should be 1, button bits should be 1
	expected := uint8(0xFF)
	if result != expected {
		t.Errorf("Expected 0x%02X, got 0x%02X", expected, result)
	}
}

func TestJoypadRead_ActionButtonsSelected(t *testing.T) {
	j := New(nil)

	// Select action buttons (P15=0)
	j.Write(0xDF) // 11011111 - P15=0, P14=1

	// Press A button
	j.buttonA = true

	result := j.Read()

	// Expected: 11011110 (P15=0, P14=1, A pressed=bit0 clear)
	expected := uint8(0xDE)
	if result != expected {
		t.Errorf("Expected 0x%02X, got 0x%02X", expected, result)
	}
}

func TestJoypadRead_DirectionButtonsSelected(t *testing.T) {
	j := New(nil)

	// Select direction buttons (P14=0)
	j.Write(0xEF) // 11101111 - P15=1, P14=0

	// Press Up button
	j.buttonUp = true

	result := j.Read()

	// Expected: 11101011 (P15=1, P14=0, Up pressed=bit2 clear)
	expected := uint8(0xEB)
	if result != expected {
		t.Errorf("Expected 0x%02X, got 0x%02X", expected, result)
	}
}

func TestJoypadRead_MultipleActionButtons(t *testing.T) {
	j := New(nil)

	// Select action buttons
	j.Write(0xDF)

	// Press A, B, and Start
	j.buttonA = true
	j.buttonB = true
	j.buttonStart = true

	result := j.Read()

	// Expected: 11010100 (bits 0,1,3 clear for A,B,Start)
	expected := uint8(0xD4)
	if result != expected {
		t.Errorf("Expected 0x%02X, got 0x%02X", expected, result)
	}
}

func TestJoypadRead_MultipleDirectionButtons(t *testing.T) {
	j := New(nil)

	// Select direction buttons
	j.Write(0xEF)

	// Press Up and Right (valid combination)
	j.buttonUp = true
	j.buttonRight = true

	result := j.Read()

	// Expected: 11101010 (bits 0,2 clear for Right,Up)
	expected := uint8(0xEA)
	if result != expected {
		t.Errorf("Expected 0x%02X, got 0x%02X", expected, result)
	}
}

func TestJoypadRead_NoSelectionBits(t *testing.T) {
	j := New(nil)

	// Select neither action nor direction (both P15 and P14 = 0)
	j.Write(0xCF)

	// Press some buttons
	j.buttonA = true
	j.buttonUp = true

	result := j.Read()

	// When both are selected, both sets of buttons should be readable
	// Expected: 11001010 (bits 0,2 clear from both sets)
	expected := uint8(0xCA)
	if result != expected {
		t.Errorf("Expected 0x%02X, got 0x%02X", expected, result)
	}
}

func TestJoypadWrite_SelectionBits(t *testing.T) {
	j := New(nil)

	// Write to select action buttons only
	j.Write(0xDF) // P15=0, P14=1

	if j.selectAction {
		t.Error("Expected selectAction to be false (bit cleared)")
	}
	if !j.selectDirection {
		t.Error("Expected selectDirection to be true (bit set)")
	}

	// Write to select direction buttons only
	j.Write(0xEF) // P15=1, P14=0

	if !j.selectAction {
		t.Error("Expected selectAction to be true (bit set)")
	}
	if j.selectDirection {
		t.Error("Expected selectDirection to be false (bit cleared)")
	}
}

func TestOppositeDirectionBlocking_UpDown(t *testing.T) {
	j := New(nil)

	// Press Down first
	j.PressButton("Down")
	if !j.buttonDown {
		t.Error("Down should be pressed")
	}

	// Try to press Up (should be blocked)
	j.PressButton("Up")
	if j.buttonUp {
		t.Error("Up should be blocked when Down is pressed")
	}

	// Release Down, then press Up
	j.ReleaseButton("Down")
	j.PressButton("Up")
	if !j.buttonUp {
		t.Error("Up should be pressed after Down is released")
	}

	// Try to press Down (should be blocked)
	j.PressButton("Down")
	if j.buttonDown {
		t.Error("Down should be blocked when Up is pressed")
	}
}

func TestOppositeDirectionBlocking_LeftRight(t *testing.T) {
	j := New(nil)

	// Press Right first
	j.PressButton("Right")
	if !j.buttonRight {
		t.Error("Right should be pressed")
	}

	// Try to press Left (should be blocked)
	j.PressButton("Left")
	if j.buttonLeft {
		t.Error("Left should be blocked when Right is pressed")
	}

	// Release Right, then press Left
	j.ReleaseButton("Right")
	j.PressButton("Left")
	if !j.buttonLeft {
		t.Error("Left should be pressed after Right is released")
	}

	// Try to press Right (should be blocked)
	j.PressButton("Right")
	if j.buttonRight {
		t.Error("Right should be blocked when Left is pressed")
	}
}

func TestJoypadInterrupt(t *testing.T) {
	interruptCalled := false
	var interruptBit uint8

	j := New(func(bit uint8) {
		interruptCalled = true
		interruptBit = bit
	})

	// Press a button
	j.PressButton("A")

	// Verify interrupt was triggered
	if !interruptCalled {
		t.Error("Interrupt should be called when button is pressed")
	}

	if interruptBit != 4 {
		t.Errorf("Expected interrupt bit 4 (joypad), got %d", interruptBit)
	}
}

func TestJoypadInterrupt_OnlyOnPress(t *testing.T) {
	callCount := 0

	j := New(func(_ uint8) {
		callCount++
	})

	// First press should trigger interrupt
	j.PressButton("A")
	if callCount != 1 {
		t.Errorf("Expected 1 interrupt call, got %d", callCount)
	}

	// Pressing again while already pressed should NOT trigger another interrupt
	j.PressButton("A")
	if callCount != 1 {
		t.Errorf("Expected 1 interrupt call (no spam), got %d", callCount)
	}

	// Release and press again should trigger another interrupt
	j.ReleaseButton("A")
	j.PressButton("A")
	if callCount != 2 {
		t.Errorf("Expected 2 interrupt calls (after release), got %d", callCount)
	}
}

func TestReleaseButton(t *testing.T) {
	j := New(nil)

	// Press and release each button
	buttons := []string{"A", "B", "Start", "Select", "Up", "Down", "Left", "Right"}

	for _, button := range buttons {
		// Press
		j.PressButton(button)

		// Release
		j.ReleaseButton(button)

		// Verify all buttons are released
		if j.buttonA || j.buttonB || j.buttonStart || j.buttonSelect ||
			j.buttonUp || j.buttonDown || j.buttonLeft || j.buttonRight {
			t.Errorf("Button %s was not properly released", button)
		}
	}
}

func TestPressButton_AllButtons(t *testing.T) {
	j := New(nil)

	tests := []struct {
		button string
		check  func() bool
	}{
		{"A", func() bool { return j.buttonA }},
		{"B", func() bool { return j.buttonB }},
		{"Start", func() bool { return j.buttonStart }},
		{"Select", func() bool { return j.buttonSelect }},
		{"Up", func() bool { return j.buttonUp }},
		{"Down", func() bool { return j.buttonDown }},
		{"Left", func() bool { return j.buttonLeft }},
		{"Right", func() bool { return j.buttonRight }},
	}

	for _, tt := range tests {
		// Reset joypad
		j = New(nil)

		// Press button
		j.PressButton(tt.button)

		// Check if pressed
		if !tt.check() {
			t.Errorf("Button %s was not pressed", tt.button)
		}
	}
}

func TestJoypadRead_ButtonMapping(t *testing.T) {
	tests := []struct {
		name           string
		selectValue    uint8
		pressedButtons []string
		expectedBits   uint8 // The low 4 bits of the result
	}{
		{
			name:           "Action: A pressed",
			selectValue:    0xDF, // P15=0 (select action)
			pressedButtons: []string{"A"},
			expectedBits:   0x0E, // 1110 (bit 0 clear)
		},
		{
			name:           "Action: B pressed",
			selectValue:    0xDF,
			pressedButtons: []string{"B"},
			expectedBits:   0x0D, // 1101 (bit 1 clear)
		},
		{
			name:           "Action: Select pressed",
			selectValue:    0xDF,
			pressedButtons: []string{"Select"},
			expectedBits:   0x0B, // 1011 (bit 2 clear)
		},
		{
			name:           "Action: Start pressed",
			selectValue:    0xDF,
			pressedButtons: []string{"Start"},
			expectedBits:   0x07, // 0111 (bit 3 clear)
		},
		{
			name:           "Direction: Right pressed",
			selectValue:    0xEF, // P14=0 (select direction)
			pressedButtons: []string{"Right"},
			expectedBits:   0x0E, // 1110 (bit 0 clear)
		},
		{
			name:           "Direction: Left pressed",
			selectValue:    0xEF,
			pressedButtons: []string{"Left"},
			expectedBits:   0x0D, // 1101 (bit 1 clear)
		},
		{
			name:           "Direction: Up pressed",
			selectValue:    0xEF,
			pressedButtons: []string{"Up"},
			expectedBits:   0x0B, // 1011 (bit 2 clear)
		},
		{
			name:           "Direction: Down pressed",
			selectValue:    0xEF,
			pressedButtons: []string{"Down"},
			expectedBits:   0x07, // 0111 (bit 3 clear)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := New(nil)
			j.Write(tt.selectValue)

			for _, button := range tt.pressedButtons {
				j.PressButton(button)
			}

			result := j.Read()
			actualBits := result & 0x0F

			if actualBits != tt.expectedBits {
				t.Errorf("Expected low 4 bits = 0x%X, got 0x%X (full result: 0x%02X)",
					tt.expectedBits, actualBits, result)
			}
		})
	}
}
