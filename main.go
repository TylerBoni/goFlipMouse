package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bendahl/uinput"
	evdev "github.com/gvalkov/golang-evdev"
)

var logger *log.Logger

func setupLogging() (*os.File, error) {
	// Ensure directory exists
	dir := "/cache"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %v", err)
		}
	}

	// Open or create the log file
	logFile, err := os.OpenFile("/cache/goFlipMouse.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	// Create a logger that writes to the file
	logger = log.New(logFile, "", log.LstdFlags)

	return logFile, nil
}

// var DEBUG bool = os.Getenv("DEBUG") != ""
const DEBUG = true

func dprint(format string, v ...interface{}) {
	if DEBUG {
		fmt.Printf(format, v...)
		logger.Printf(format, v...)
	}
}

func intToBool(i int32) bool {
	return i != 0
}

var (
	velocityX    float64 = 0
	velocityY    float64 = 0
	maxSpeed     float64 = 4
	speedMulti   float64 = 1
	acceleration float64 = 0.3
	friction     float64 = 0.85
)

// accelerate adjusts velocity based on input direction

func accelerate(left *bool, right *bool, up *bool, down *bool, vertualMouse uinput.Mouse, inputX, inputY float64) {
	var actualSpeed float64 = maxSpeed * speedMulti
	// Apply acceleration in the input direction
	if inputX != 0 {
		velocityX += float64(inputX) * acceleration
	} else {
		// Apply friction when no input
		velocityX *= friction
	}

	if inputY != 0 {
		velocityY += float64(inputY) * acceleration
	} else {
		velocityY *= friction
	}

	// Clamp to maximum speed
	speed := math.Sqrt(velocityX*velocityX + velocityY*velocityY)
	if speed > actualSpeed {
		scale := actualSpeed / speed
		velocityX *= scale
		velocityY *= scale
	}

	// Cut off tiny movements
	if math.Abs(velocityX) < 0.1 {
		velocityX = 0
	}
	if math.Abs(velocityY) < 0.1 {
		velocityY = 0
	}

	// while moving, move the mouse
	if velocityX != 0 || velocityY != 0 {
		vertualMouse.Move(int32(velocityX), int32(velocityY))
	}
}

// Constants for event return values
const (
	ChangedToMouse = -2
	MuteEvent      = 0
	PassThruEvent  = 1
	ChangedEvent   = 2
)

// Event type and code constants from linux/input-event-codes.h
const (
	EvKey         = 0x01
	EvRel         = 0x02
	EvMsc         = 0x04
	EvSyn         = 0x00
	KeyPower      = 116
	KeyHelp       = 138
	KeyEnter      = 28
	KeyVolumeUp   = 115
	KeyVolumeDown = 114
	BtnLeft       = 0x110
	BtnRight      = 0x111
	RelX          = 0x00
	RelY          = 0x01
	RelWheel      = 0x08
	RelHWheel     = 0x06
	MscScan       = 0x04
	SynReport     = 0
)

// Define keyboard types
const (
	KBD_TYPE_PHONE = iota
	KBD_TYPE_LAPTOP
	KBD_TYPE_EXTERNAL
	// Add other types as needed
)

// Create a mapping structure
type KeyMapping struct {
	PowerKey       uint16
	EnterKey       uint16
	Key1           uint16
	Key3           uint16
	ToggleKey      uint16
	ClickKey       uint16
	DragKey        uint16
	VolumeUpKey    uint16
	VolumeDownKey  uint16
	UpKey          uint16
	DownKey        uint16
	LeftKey        uint16
	RightKey       uint16
	ScrollDownKey  uint16
	ScrollUpKey    uint16
	ScrollLeftKey  uint16
	ScrollRightKey uint16
	CallKey        uint16
	LeftSoftKey    uint16
	RightSoftKey   uint16
	MessagesKey    uint16
	// Add other keys as needed
}

// Create mappings for each keyboard type
var keyMappings = map[int]KeyMapping{
	KBD_TYPE_PHONE: {
		PowerKey:       116, // End call key
		EnterKey:       28,
		Key1:           2,
		Key3:           4,
		ToggleKey:      138, // star key
		ClickKey:       28,  // Enter
		DragKey:        4,   // 3 key
		VolumeUpKey:    115, // Up arrow
		VolumeDownKey:  114, // Down arrow
		UpKey:          103, // Up arrow
		DownKey:        108, // Down arrow
		LeftKey:        105, // Left arrow
		RightKey:       106, // Right arrow
		ScrollDownKey:  522, // Page down
		ScrollUpKey:    8,   // 7 key
		ScrollLeftKey:  5,   // 4 jey
		ScrollRightKey: 7,   // 6 key
		CallKey:        231,
		LeftSoftKey:    139,
		RightSoftKey:   48,
		MessagesKey:    30,
	},
	KBD_TYPE_LAPTOP: {
		PowerKey:       1, // Esc
		EnterKey:       28,
		ToggleKey:      29, // Ctrl
		Key1:           2,
		Key3:           4,
		ClickKey:       57,  // Space
		DragKey:        32,  // D key
		VolumeUpKey:    13,  // = key
		VolumeDownKey:  12,  // - key
		UpKey:          103, // up arrow
		DownKey:        108, // down arrow
		LeftKey:        105, // left arrow
		RightKey:       106, // right arrow
		ScrollUpKey:    17,  // w ke
		ScrollDownKey:  31,  // s key
		ScrollLeftKey:  30,  // a key
		ScrollRightKey: 32,  // d key
	},
	// Add other keyboard types
}

// Global state
var (
	mouseMode = false
	// Fix for drag issue - track left button state
	toggleKeyDownTime time.Time
	leftBtnPressed    = false
	rightBtnPressed   = false
	dragToggleActive  = false
	upKeyActive       = false
	downKeyActive     = false
	leftKeyActive     = false
	rightKeyActive    = false
)

// InputDevice represents a physical input device
type InputDevice struct {
	device       *evdev.InputDevice
	name         string
	path         string
	keyboardType int
}

// Detect keyboard type during device discovery
func determineKeyboardType(dev *evdev.InputDevice) int {
	if dev.Name == "AT Translated Set 2 keyboard" {
		return KBD_TYPE_LAPTOP
	}
	// Default to standard
	return KBD_TYPE_PHONE
}

// FindInputDevices locates keyboard devices
func FindInputDevices() ([]*InputDevice, error) {
	var devices []*InputDevice
	// wantedDevs := []string{"mtk-kpd", "matrix-keypad"}
	wantedDevs := []string{"mtk-kpd", "matrix-keypad", "AT Translated Set 2 keyboard"}

	// Find all input devices
	devFiles, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		return nil, fmt.Errorf("failed to list input devices: %v", err)
	}

	for _, path := range devFiles {
		dev, err := evdev.Open(path)
		if err != nil {
			continue
		}

		// Check if it's a device we want
		for _, wanted := range wantedDevs {
			if dev.Name == wanted {
				devices = append(devices, &InputDevice{
					device:       dev,
					name:         dev.Name,
					path:         path,
					keyboardType: determineKeyboardType(dev),
				})
				break
			}
		}
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("no suitable input devices found")
	}

	return devices, nil
}

// Process a single input event
func ProcessEvent(event *evdev.InputEvent, virtualMouse uinput.Mouse, km KeyMapping, devName string) int {
	if event.Type == EvMsc {
		return PassThruEvent
	}
	dprint("Device: %s\n", devName)
	dprint("Event: %+v\n", event)
	dprint("\n")
	// Convert the event time

	// Static variables (using global vars in Go)
	static := struct {
		start    time.Time
		slowdown uint
	}{
		// These values persist but are encapsulated
	}

	// Handle key events
	if event.Type == EvKey {
		// Power key handling
		if event.Code == km.PowerKey {
			dprint("Power key pressed\n")
			mouseMode = false
			// Release any stuck buttons when exiting mouse mode
			leftBtnPressed = false
			rightBtnPressed = false
			virtualMouse.LeftRelease()
			virtualMouse.RightRelease()
			return PassThruEvent
		}

		// Help key for mouse mode toggle
		if event.Code == km.ToggleKey {
			dprint("Toggle key pressed\n")
			// Record start time on key press
			if event.Value == 1 {
				toggleKeyDownTime = time.Now()
				return MuteEvent
			}

			// Check for long press
			diff := time.Since(toggleKeyDownTime)
			toggleKeyDownTime = time.Time{}
			if diff > 225*time.Millisecond {
				// Long press - enter mouse mode
				dprint("Long press detected\n")
				mouseMode = !mouseMode
				// Wiggle mouse to show it's active
				if mouseMode {
					virtualMouse.Move(1, 0)
					virtualMouse.Move(0, -2)
				}
				return MuteEvent
			} else {
				// Short press - pass through normal key event
				// This is tricky in Go - we'd need to forward the raw event
				return PassThruEvent
			}
		}
	}

	// If not in mouse mode, just pass through
	if !mouseMode {
		return PassThruEvent
	}

	// Handle mouse mode key events
	dprint("Handling event in mouse mode\n")
	switch event.Code {
	case km.EnterKey:
		// Convert Enter key to left mouse button
		print("Enter key pressed\n")
		if event.Value == 1 {
			virtualMouse.LeftPress()
			leftBtnPressed = true
		} else {
			virtualMouse.LeftRelease()
			leftBtnPressed = false
		}
		return MuteEvent
	case km.VolumeUpKey:
		if event.Value == 1 {
			maxSpeed = maxSpeed + 1
			fmt.Printf("Mouse speed increased to %d\n", maxSpeed)
		}
		return MuteEvent
	case km.VolumeDownKey:
		if event.Value == 1 {
			maxSpeed = maxSpeed - 1
			if maxSpeed < 1 {
				maxSpeed = 1
			}
			fmt.Printf("Mouse speed decreased to %d\n", maxSpeed)
		}
		return MuteEvent

	// Add KEY_3 as drag toggle (assuming it's defined)
	case km.Key3: // KEY_3 (adjust if different)
		if event.Value == 1 { // Only on press, not release
			dragToggleActive = !dragToggleActive
			if dragToggleActive {
				virtualMouse.LeftPress()
				leftBtnPressed = true
				fmt.Println("Drag mode activated")
			} else {
				virtualMouse.LeftRelease()
				leftBtnPressed = false
				fmt.Println("Drag mode deactivated")
			}
		}
		return MuteEvent
	case km.UpKey:
		upKeyActive = intToBool(event.Value)
		return MuteEvent

	case km.DownKey:
		downKeyActive = intToBool(event.Value)
		return MuteEvent

	case km.LeftKey:
		leftKeyActive = intToBool(event.Value)
		return MuteEvent

	case km.RightKey:
		rightKeyActive = intToBool(event.Value)
		return MuteEvent

	case km.EnterKey, km.Key1:
		// Toggle drag mode instead of just press
		if !leftBtnPressed {
			virtualMouse.LeftPress()
			leftBtnPressed = true
			fmt.Println("Left button pressed")
		} else {
			virtualMouse.LeftRelease()
			leftBtnPressed = false
			fmt.Println("Left button released")
		}
		return MuteEvent

	case km.ScrollUpKey: //0, 1: // Scroll wheel up
		dprint("Scroll wheel up\n")
		static.slowdown++
		if static.slowdown%5 != 0 {
			return MuteEvent
		}
		// Need wheel support - might require custom implementation
		return MuteEvent

	case km.ScrollDownKey: //18, 16: // Scroll wheel down
		static.slowdown++
		if static.slowdown%5 != 0 {
			return MuteEvent
		}
		// Need wheel support
		return MuteEvent

	case km.ScrollRightKey: // Horizontal scroll right
		static.slowdown++
		if static.slowdown%5 != 0 {
			return MuteEvent
		}
		// Need wheel support
		return MuteEvent

	case km.ScrollLeftKey: // Horizontal scroll left
		static.slowdown++
		if static.slowdown%5 != 0 {
			return MuteEvent
		}
		return MuteEvent
	default:
		return PassThruEvent
	}

}

func main() {
	fmt.Println("Starting virtual mouse service...")
	logFile, err := setupLogging()
	if err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}
	defer logFile.Close()

	// Create a virtual mouse
	mouse, err := uinput.CreateMouse("/dev/uinput", []byte("goFlipMouse"))
	if err != nil {
		log.Fatalf("Failed to create virtual mouse: %v", err)
	}
	defer mouse.Close()

	// Create a pass-through keyboard for non-muted events
	keyboard, err := uinput.CreateKeyboard("/dev/uinput", []byte("goFlipKeyboard"))
	if err != nil {
		log.Fatalf("Failed to create virtual keyboard: %v", err)
	}
	defer keyboard.Close()

	// Find physical input devices
	devices, err := FindInputDevices()
	if err != nil {
		log.Fatalf("Error finding input devices: %v", err)
	}
	fmt.Printf("Found %d input devices\n", len(devices))

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nShutting down...")
		// Release buttons in case they're stuck
		mouse.LeftRelease()
		mouse.RightRelease()
		for _, dev := range devices {
			dev.device.File.Close()
		}
		mouse.Close()

		os.Exit(0)
	}()

	// Create event channels for each device
	for i, dev := range devices {
		fmt.Printf("Monitoring device %d: %s\n", i, dev.name)
		dprint("Debugging enabled\n")

		err := dev.device.Grab()
		if err != nil {
			log.Printf("Failed to grab device %s: %v", dev.name, err)
			continue
		}

		// Start a goroutine for each device
		go func(d *InputDevice) {
			for {
				var mapping KeyMapping = keyMappings[d.keyboardType]
				// Read the next event
				event, err := d.device.ReadOne()
				if err != nil {
					log.Printf("Error reading from %s: %v", d.name, err)
					continue
				}

				// Process the event
				result := ProcessEvent(event, mouse, mapping, d.name)
				if result == PassThruEvent {
					switch event.Code {
					}
					if event.Value == 1 {
						keyboard.KeyDown(int(event.Code))
					}
					if event.Value == 0 {
						keyboard.KeyUp(int(event.Code))
					}
				} else {
					dprint("Intercepted event. Result: %d\n", result)
				}

				// Debug output if needed
				if result != MuteEvent && event.Type != EvSyn {
					// dprint("Event: type=%d, code=%d, value=%d, result=%d\n",
					// event.Type, event.Code, event.Value, result)
				}
			}
		}(dev)

		go func() {
			ticker := time.NewTicker(16 * time.Millisecond) // ~60fps
			defer ticker.Stop()

			for range ticker.C {
				if !mouseMode {
					// Reset velocities when not in mouse mode
					velocityX = 0
					velocityY = 0
					continue
				}

				// Calculate input direction
				inputX := float64(0)
				inputY := float64(0)

				if leftKeyActive {
					inputX -= maxSpeed
				}
				if rightKeyActive {
					inputX += maxSpeed
				}
				if upKeyActive {
					inputY -= maxSpeed
				}
				if downKeyActive {
					inputY += maxSpeed
				}

				// Call accelerate function with current direction state
				accelerate(&leftKeyActive, &rightKeyActive, &upKeyActive, &downKeyActive, mouse, inputX, inputY)
			}
		}()
	}

	// Keep the program running
	fmt.Println("Virtual mouse active. Press Ctrl+C to exit.")
	select {} // Block forever
}
