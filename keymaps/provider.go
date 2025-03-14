package keymaps

// CreateDefaultKeyMappingProvider creates and returns a provider with all default mappings
func CreateDefaultKeyMappingProvider() *KeyMappingProvider {
	provider := NewKeyMappingProvider()

	// Register all available mappings
	RegisterPhoneKeyMapping(provider)
	RegisterLaptopKeyMapping(provider)

	return provider
}

// GetKeyboardType determines the keyboard type based on device name
func GetKeyboardType(deviceName string) int {
	switch deviceName {
	case "AT Translated Set 2 keyboard":
		return KBD_TYPE_LAPTOP
	case "USB-HID Keyboard":
		return KBD_TYPE_EXTERNAL
	default:
		return KBD_TYPE_PHONE
	}
}
