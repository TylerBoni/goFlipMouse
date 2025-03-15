package keymaps

// KeyMapping defines keyboard key mappings
type KeyMapping struct {
	ExitKey        uint16
	EnterKey       uint16
	ToggleMouseKey uint16
	ClickKey       uint16
	DragKey        uint16
	FasterKey      uint16
	SlowerKey      uint16
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
}

// KeyMappingProvider provides key mappings for different keyboard types
type KeyMappingProvider struct {
	mappings map[int]KeyMapping
}

// NewKeyMappingProvider creates a new mapping provider with default mappings
func NewKeyMappingProvider() *KeyMappingProvider {
	return &KeyMappingProvider{
		mappings: map[int]KeyMapping{},
	}
}

// GetMapping returns the key mapping for the specified keyboard type
func (p *KeyMappingProvider) GetMapping(keyboardType int) KeyMapping {
	mapping, exists := p.mappings[keyboardType]
	if !exists {
		// Default to phone mapping if type not found
		return p.mappings[KBD_TYPE_PHONE]
	}
	return mapping
}

// RegisterMapping registers a new key mapping for a specific keyboard type
func (p *KeyMappingProvider) RegisterMapping(keyboardType int, mapping KeyMapping) {
	p.mappings[keyboardType] = mapping
}
