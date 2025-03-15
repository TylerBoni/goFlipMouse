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
	"github.com/goFlipMouse/keymaps"
	evdev "github.com/grafov/evdev"
)

// Constants for Linux events
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

// Event processing return values
const (
	ChangedToMouse = -2
	MuteEvent      = 0
	PassThruEvent  = 1
	ChangedEvent   = 2
)

// Config holds application configuration
type Config struct {
	LogPath           string
	DebugMode         bool
	LongPressDuration time.Duration
}

// Default configuration
var defaultConfig = Config{
	LogPath:           "/cache/goFlipMouse.log",
	DebugMode:         true,
	LongPressDuration: 225 * time.Millisecond,
}

// Logger manages application logging
type Logger struct {
	*log.Logger
	debugMode bool
}

// NewLogger creates a new logger instance
func NewLogger(config Config) (*Logger, *os.File, error) {
	// Ensure directory exists
	dir := filepath.Dir(config.LogPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, nil, fmt.Errorf("failed to create log directory: %v", err)
		}
	}

	// Open or create the log file
	logFile, err := os.OpenFile(config.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, err
	}

	logger := &Logger{
		Logger:    log.New(logFile, "", log.LstdFlags),
		debugMode: config.DebugMode,
	}

	return logger, logFile, nil
}

// Debug logs a message if debug mode is enabled
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.debugMode {
		fmt.Printf(format, v...)
		l.Printf(format, v...)
	}
}

// Import the KeyMapping and KeyMappingProvider from the keymaps package

// MouseState represents the state of the mouse controller
type MouseState struct {
	VelocityX       float64
	VelocityY       float64
	ScrollVelocityX float64
	ScrollVelocityY float64
	MaxSpeed        float64
	ScrollMaxSpeed  float64
	SpeedMulti      float64
	ScrollMulti     float64
	Acceleration    float64
	Friction        float64

	MouseMode bool

	LeftBtnPressed    bool
	RightBtnPressed   bool
	DragToggleActive  bool
	UpKeyActive       bool
	DownKeyActive     bool
	LeftKeyActive     bool
	RightKeyActive    bool
	ScrollUpActive    bool
	ScrollDownActive  bool
	ScrollLeftActive  bool
	ScrollRightActive bool

	ToggleKeyDown     bool
	ToggleKeyDownTime time.Time
}

// NewMouseState creates a new mouse state with default values
func NewMouseState() *MouseState {
	return &MouseState{
		VelocityX:       0,
		VelocityY:       0,
		ScrollVelocityX: 0,
		ScrollVelocityY: 0,
		MaxSpeed:        4,
		ScrollMaxSpeed:  30,
		SpeedMulti:      1,
		ScrollMulti:     1,
		Acceleration:    0.3,
		Friction:        0.85,

		MouseMode:       false,
		LeftBtnPressed:  false,
		RightBtnPressed: false,
	}
}

// MouseController manages mouse movements and actions
type MouseController struct {
	State  *MouseState
	Mouse  uinput.Mouse
	Logger *Logger
}

// NewMouseController creates a new mouse controller
func NewMouseController(mouse uinput.Mouse, logger *Logger) *MouseController {
	return &MouseController{
		State:  NewMouseState(),
		Mouse:  mouse,
		Logger: logger,
	}
}

func NewVirtualMouse() uinput.Mouse {
	mouse, err := uinput.CreateMouse("/dev/uinput", []byte("goFlipMouse"))
	if err != nil {
		panic(err)
	}
	return mouse
}

func (mc *MouseController) AccelerateVelocity(inputX, inputY float64, maxSpeed float64, velocityX, velocityY float64) (float64, float64) {
	actualSpeed := maxSpeed

	// Apply acceleration in the input direction
	if inputX != 0 {
		velocityX += inputX * mc.State.Acceleration
	} else {
		// Apply friction when no input
		velocityX *= mc.State.Friction
	}

	if inputY != 0 {
		velocityY += inputY * mc.State.Acceleration
	} else {
		velocityY *= mc.State.Friction
	}

	// Clamp to maximum speed
	speed := math.Sqrt(velocityX*velocityX + velocityY*velocityY)
	if speed > actualSpeed {
		scale := actualSpeed / speed
		velocityX *= scale
		velocityY *= scale
	}

	// Cut off tiny movements
/*
	if math.Abs(velocityX) < 0.1 {
		velocityX = 0
		}

		if math.Abs(velocityY) < 0.1 {
			velocityY = 0
	} */

	return velocityX, velocityY
}

// AccelerateAndMove calculates acceleration and applies movement to the mouse
func (mc *MouseController) AccelerateAndMove(inputX, inputY float64) {
	mc.State.VelocityX, mc.State.VelocityY = mc.AccelerateVelocity(inputX, inputY, mc.State.MaxSpeed, mc.State.VelocityX, mc.State.VelocityY)
	// Move the mouse if there's any velocity
	if mc.State.VelocityX != 0 || mc.State.VelocityY != 0 {
		mc.Mouse.Move(int32(mc.State.VelocityX*mc.State.SpeedMulti), int32(mc.State.VelocityY*mc.State.SpeedMulti))
	}
}

// AccelerateAndScroll can be used for scrolling with acceleration physics
func (mc *MouseController) AccelerateAndScroll(inputX, inputY float64) {
	// We'll use the input for the Y direction only
	mc.State.ScrollVelocityX, mc.State.ScrollVelocityY = mc.AccelerateVelocity(inputX, inputY, mc.State.ScrollMaxSpeed, mc.State.ScrollVelocityX, mc.State.ScrollVelocityY)
	// Scroll if there's any velocity (only vertical)
	if mc.State.ScrollVelocityY != 0 {
		mc.Mouse.Wheel(false, int32(mc.State.ScrollVelocityY*mc.State.ScrollMulti))
	}
	if mc.State.ScrollVelocityX != 0 {
		mc.Mouse.Wheel(true, int32(mc.State.ScrollVelocityX*mc.State.ScrollMulti))
	}
}

// IncreaseSpeed increases the mouse movement speed
func (mc *MouseController) IncreaseSpeed() {
	mc.State.MaxSpeed++
	fmt.Printf("Mouse speed increased to %.1f\n", mc.State.MaxSpeed)
}

// DecreaseSpeed decreases the mouse movement speed
func (mc *MouseController) DecreaseSpeed() {
	mc.State.MaxSpeed--
	if mc.State.MaxSpeed < 1 {
		mc.State.MaxSpeed = 1
	}
	fmt.Printf("Mouse speed decreased to %.1f\n", mc.State.MaxSpeed)
}

// ToggleMouseMode toggles mouse mode on/off
func (mc *MouseController) ToggleMouseMode() {
	mc.State.MouseMode = !mc.State.MouseMode

	// Wiggle mouse to show it's active
	if mc.State.MouseMode {
mc.Mouse = NewVirtualMouse()
		mc.Mouse.Move(int32(mc.State.MaxSpeed), 0)
		time.Sleep(50 * time.Millisecond)
		mc.Mouse.Move(int32(-mc.State.MaxSpeed), 0)
	}

	// Reset button states when toggling
	if !mc.State.MouseMode {
		mc.ResetButtons()
mc.Mouse.Close()
	}
}

// ResetButtons resets button states and releases any pressed buttons
func (mc *MouseController) ResetButtons() {
	if mc.State.LeftBtnPressed {
		mc.Mouse.LeftRelease()
		mc.State.LeftBtnPressed = false
	}

	if mc.State.RightBtnPressed {
		mc.Mouse.RightRelease()
		mc.State.RightBtnPressed = false
	}

	mc.State.DragToggleActive = false
}

// ToggleDragMode toggles drag mode on/off
func (mc *MouseController) ToggleDragMode() {
	mc.State.DragToggleActive = !mc.State.DragToggleActive

	if mc.State.DragToggleActive {
		mc.Mouse.LeftPress()
		mc.State.LeftBtnPressed = true
		fmt.Println("Drag mode activated")
	} else {
		mc.Mouse.LeftRelease()
		mc.State.LeftBtnPressed = false
		fmt.Println("Drag mode deactivated")
	}
}

// ToggleLeftButton toggles left button press/release
func (mc *MouseController) ToggleLeftButton() {
	if !mc.State.LeftBtnPressed {
		mc.Mouse.LeftPress()
		mc.State.LeftBtnPressed = true
		fmt.Println("Left button pressed")
	} else {
		mc.Mouse.LeftRelease()
		mc.State.LeftBtnPressed = false
		fmt.Println("Left button released")
	}
}

// InputDevice represents a physical input device
type InputDevice struct {
	Device       *evdev.InputDevice
	Name         string
	Path         string
	KeyboardType int // Refers to keymaps.KBD_TYPE_*
}

// EventProcessor processes input events
type EventProcessor struct {
	MouseController    *MouseController
	Config             Config
	KeyMappingProvider *keymaps.KeyMappingProvider
	Logger             *Logger
	VirtualKeyboard    uinput.Keyboard
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(
	mouseController *MouseController,
	config Config,
	keyMappingProvider *keymaps.KeyMappingProvider,
	logger *Logger,
	virtualKeyboard uinput.Keyboard,
) *EventProcessor {
	return &EventProcessor{
		MouseController:    mouseController,
		Config:             config,
		KeyMappingProvider: keyMappingProvider,
		Logger:             logger,
		VirtualKeyboard:    virtualKeyboard,
	}
}

// ProcessEvent processes a single input event
func (ep *EventProcessor) ProcessEvent(event *evdev.InputEvent, device *InputDevice) int {
	if event.Type != EvKey {
		ep.Logger.Debug("Event: %+v\n", event)
	}

	// Get the key mapping for this device
	km := ep.KeyMappingProvider.GetMapping(device.KeyboardType)
	mouseState := ep.MouseController.State

	// Handle key events
	if event.Type == EvKey {
		// Power key handling - exit mouse mode
		if event.Code == km.ExitKey {
			ep.Logger.Debug("Power key pressed\n")
			mouseState.MouseMode = false
			ep.MouseController.ResetButtons()
			return PassThruEvent
		}

		// Toggle key for mouse mode
		if event.Code == km.ToggleMouseKey {
			ep.Logger.Debug("Toggle key pressed\n")
			if event.Value == 2 {
				return MuteEvent
			}

			// Record start time on key press
			if event.Value == 1 {
				mouseState.ToggleKeyDownTime = time.Now()
				mouseState.ToggleKeyDown = true
				return MuteEvent
			}

			// Check for long press
			diff := time.Since(mouseState.ToggleKeyDownTime)
			mouseState.ToggleKeyDownTime = time.Time{}
			mouseState.ToggleKeyDown = false

			if diff > ep.Config.LongPressDuration {
				// Long press - toggle mouse mode
				ep.Logger.Debug("Long press detected\n")
				ep.MouseController.ToggleMouseMode()
				return MuteEvent
			} else {
				// Short press - pass through normal key event
				return PassThruEvent
			}
		}
	}

	// If not in mouse mode, just pass through
	if !mouseState.MouseMode {
		return PassThruEvent
	}

	// Handle mouse mode key events
	ep.Logger.Debug("Handling event in mouse mode\n")

	switch event.Code {
	case km.EnterKey:
		// Convert Enter key to left mouse button
		if event.Value == 1 {
			ep.MouseController.Mouse.LeftPress()
			mouseState.LeftBtnPressed = true
		} else {
			ep.MouseController.Mouse.LeftRelease()
			mouseState.LeftBtnPressed = false
		}
		return MuteEvent

	case km.FasterKey:
		if event.Value == 1 {
			ep.MouseController.IncreaseSpeed()
		}
		return MuteEvent

	case km.SlowerKey:
		if event.Value == 1 {
			ep.MouseController.DecreaseSpeed()
		}
		return MuteEvent

	case km.DragKey:
		if event.Value == 1 {
			ep.MouseController.ToggleDragMode()
		}
		return MuteEvent

	case km.UpKey:
		mouseState.UpKeyActive = (event.Value != 0)
		return MuteEvent

	case km.DownKey:
		mouseState.DownKeyActive = (event.Value != 0)
		return MuteEvent

	case km.LeftKey:
		mouseState.LeftKeyActive = (event.Value != 0)
		return MuteEvent

	case km.RightKey:
		mouseState.RightKeyActive = (event.Value != 0)
		return MuteEvent

	case km.ScrollUpKey:
		// Wheel scrolling functionality
		mouseState.ScrollUpActive = (event.Value != 0)
		return MuteEvent

	case km.ScrollDownKey:
		// Wheel scrolling functionality
		mouseState.ScrollDownActive = (event.Value != 0)
		return MuteEvent

	case km.ScrollRightKey:
		// Horizontal wheel scrolling
		mouseState.ScrollRightActive = (event.Value != 0)
		return MuteEvent

	case km.ScrollLeftKey:
		// Horizontal wheel scrolling
		mouseState.ScrollLeftActive = (event.Value != 0)
		return MuteEvent
	}
return PassThruEvent
}

// DeviceManager manages input devices
type DeviceManager struct {
	Devices         []*InputDevice
	EventProcessor  *EventProcessor
	MouseController *MouseController
	Logger          *Logger
}

// NewDeviceManager creates a new device manager
func NewDeviceManager(
	eventProcessor *EventProcessor,
	mouseController *MouseController,
	logger *Logger,
) *DeviceManager {
	return &DeviceManager{
		Devices:         []*InputDevice{},
		EventProcessor:  eventProcessor,
		MouseController: mouseController,
		Logger:          logger,
	}
}

// FindInputDevices locates and initializes input devices
func (dm *DeviceManager) FindInputDevices() error {
	// Define devices we're looking for
	wantedDevs := []string{"mtk-kpd", "matrix-keypad", "AT Translated Set 2 keyboard"}

	// Find all input devices
	devFiles, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		return fmt.Errorf("failed to list input devices: %v", err)
	}

	for _, path := range devFiles {
		dev, err := evdev.Open(path)
		if err != nil {
			continue
		}

		// Check if it's a device we want
		for _, wanted := range wantedDevs {
			if dev.Name == wanted {
				keyboardType := keymaps.GetKeyboardType(dev.Name)

				dm.Devices = append(dm.Devices, &InputDevice{
					Device:       dev,
					Name:         dev.Name,
					Path:         path,
					KeyboardType: keyboardType,
				})
				break
			}
		}
	}

	if len(dm.Devices) == 0 {
		return fmt.Errorf("no suitable input devices found")
	}
	return nil
}

// StartDeviceMonitoring starts monitoring all devices
func (dm *DeviceManager) StartDeviceMonitoring() error {
	for i, dev := range dm.Devices {
		fmt.Printf("Monitoring device %d: %s\n - %s\n", i, dev.Name, dev.Path)

		err := dev.Device.Grab()
		if err != nil {
			return fmt.Errorf("failed to grab device %s: %v", dev.Name, err)
		}

		// Start a goroutine for each device to handle input events
		go dm.processDeviceEvents(dev)
	}

	// Start the movement goroutine
	go dm.processMovement()
	go dm.processScroll()

	return nil
}

// processDeviceEvents continuously processes events from a device
func (dm *DeviceManager) processDeviceEvents(device *InputDevice) {
	for {
		// Read the next event
		event, err := device.Device.ReadOne()
		if err != nil {
			dm.Logger.Printf("Error reading from %s: %v", device.Name, err)
			continue
		}

		// Process the event
		result := dm.EventProcessor.ProcessEvent(event, device)

		// Handle event result
		if result == PassThruEvent {
			dm.EventProcessor.VirtualKeyboard.SendEvent(event.Time, event.Type, event.Code, event.Value)
		} else {
			dm.Logger.Debug("Intercepted event. Result: %d\n", result)
		}
	}
}

// processMovement handles continuous mouse movement based on key states
func (dm *DeviceManager) processMovement() {
	ticker := time.NewTicker((1000 / 60) * time.Millisecond) // ~60fps
	defer ticker.Stop()

	for range ticker.C {
		mouseState := dm.MouseController.State

		if !mouseState.MouseMode {
			// Reset velocities when not in mouse mode
			mouseState.VelocityX = 0
			mouseState.VelocityY = 0
			continue
		}

		// Calculate input direction
		moveInputX := float64(0)
		moveInputY := float64(0)

		if mouseState.LeftKeyActive {
			moveInputX -= mouseState.MaxSpeed
		}
		if mouseState.RightKeyActive {
			moveInputX += mouseState.MaxSpeed
		}
		if mouseState.UpKeyActive {
			moveInputY -= mouseState.MaxSpeed
		}
		if mouseState.DownKeyActive {
			moveInputY += mouseState.MaxSpeed
		}

		dm.MouseController.AccelerateAndMove(moveInputX, moveInputY)
	}
}

func (dm *DeviceManager) processScroll() {
	ticker := time.NewTicker((1000 / 10) * time.Millisecond) // ~10fps
	defer ticker.Stop()

	for range ticker.C {
		// check if ticker even or odd
		mouseState := dm.MouseController.State

		if !mouseState.MouseMode {
			// Reset velocities when not in mouse mode
			mouseState.ScrollVelocityX = 0
			mouseState.ScrollVelocityY = 0
			continue
		}

		// Calculate input direction
		scrollInputX := float64(0)
		scrollInputY := float64(0)
		if mouseState.ScrollLeftActive {
			scrollInputX += mouseState.ScrollMaxSpeed
		}
		if mouseState.ScrollRightActive {
			scrollInputX -= mouseState.ScrollMaxSpeed
		}
		if mouseState.ScrollUpActive {
			scrollInputY += mouseState.ScrollMaxSpeed
		}
		if mouseState.ScrollDownActive {
			scrollInputY -= mouseState.ScrollMaxSpeed
		}

		// Currently too fast, not fine enough input
		// dm.MouseController.AccelerateAndScroll(scrollInputX, scrollInputY)

		dm.MouseController.Mouse.Wheel(false, int32(scrollInputY*mouseState.ScrollMulti))
		dm.MouseController.Mouse.Wheel(true, int32(scrollInputX*mouseState.ScrollMulti))
	}
}

// Application is the main application structure
type Application struct {
	Config          Config
	Logger          *Logger
	MouseController *MouseController
	EventProcessor  *EventProcessor
	DeviceManager   *DeviceManager
	VirtualMouse    uinput.Mouse
	VirtualKeyboard uinput.Keyboard
	LogFile         *os.File
}

// NewApplication creates and initializes the application
func NewApplication() (*Application, error) {
	// Use default config
	config := defaultConfig

	// Initialize logger
	logger, logFile, err := NewLogger(config)
	if err != nil {
		return nil, fmt.Errorf("failed to setup logging: %v", err)
	}

	// Create virtual devices
	virtualMouse, err := uinput.CreateMouse("/dev/uinput", []byte("goFlipMouse"))
	if err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to create virtual mouse: %v", err)
	}

	virtualKeyboard, err := uinput.CreateKeyboard("/dev/uinput", []byte("goFlipKeyboard"))
	if err != nil {
		virtualMouse.Close()
		logFile.Close()
		return nil, fmt.Errorf("failed to create virtual keyboard: %v", err)
	}

	// Create components
	mouseController := NewMouseController(virtualMouse, logger)
	keyMappingProvider := keymaps.CreateDefaultKeyMappingProvider()

	eventProcessor := NewEventProcessor(
		mouseController,
		config,
		keyMappingProvider,
		logger,
		virtualKeyboard,
	)

	deviceManager := NewDeviceManager(
		eventProcessor,
		mouseController,
		logger,
	)

	return &Application{
		Config:          config,
		Logger:          logger,
		MouseController: mouseController,
		EventProcessor:  eventProcessor,
		DeviceManager:   deviceManager,
		VirtualMouse:    virtualMouse,
		VirtualKeyboard: virtualKeyboard,
		LogFile:         logFile,
	}, nil
}

// Setup initializes the application
func (app *Application) Setup() error {
	// Find input devices
	if err := app.DeviceManager.FindInputDevices(); err != nil {
		return err
	}

	fmt.Printf("Found %d input devices\n", len(app.DeviceManager.Devices))

	// Set up signal handling for graceful shutdown
	app.setupSignalHandling()

	return nil
}

// Run starts the application
func (app *Application) Run() error {
	// Start monitoring devices
	if err := app.DeviceManager.StartDeviceMonitoring(); err != nil {
		return err
	}

	fmt.Println("Virtual mouse active. Press Ctrl+C to exit.")

	// Block forever
	select {}
}

// setupSignalHandling sets up handlers for OS signals
func (app *Application) setupSignalHandling() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\nShutting down...")
		app.Cleanup()
		os.Exit(0)
	}()
}

// Cleanup releases resources when the application exits
func (app *Application) Cleanup() {
	// Release buttons in case they're stuck
	app.VirtualMouse.LeftRelease()
	app.VirtualMouse.RightRelease()

	// Close all devices
	for _, dev := range app.DeviceManager.Devices {
		dev.Device.File.Close()
	}

	app.VirtualMouse.Close()
	app.VirtualKeyboard.Close()
	app.LogFile.Close()
}

func main() {
	fmt.Println("Starting virtual mouse service...")

	// Create and initialize the application
	app, err := NewApplication()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer app.Cleanup()

	// Setup the application
	if err := app.Setup(); err != nil {
		log.Fatalf("Failed to setup application: %v", err)
	}

	// Run the application
	if err := app.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
