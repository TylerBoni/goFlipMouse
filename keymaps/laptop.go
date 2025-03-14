package keymaps

// GetLaptopKeyMapping returns key mappings for laptop-type keyboards
func GetLaptopKeyMapping() KeyMapping {
	return KeyMapping{
		ExitKey:         1, // Esc
		EnterKey:        28,
		ToggleMouseKey:  29,
		ToggleScrollKey: 4,
		ClickKey:        57,  // Space
		DragKey:         32,  // D key
		FasterKey:       13,  // = key
		SlowerKey:       12,  // - key
		UpKey:           103, // up arrow
		DownKey:         108, // down arrow
		LeftKey:         105, // left arrow
		RightKey:        106, // right arrow
		ScrollUpKey:     17,  // w key
		ScrollDownKey:   31,  // s key
		ScrollLeftKey:   30,  // a key
		ScrollRightKey:  32,  // d key
	}
}

// RegisterLaptopKeyMapping registers laptop keyboard mapping with the provider
func RegisterLaptopKeyMapping(provider *KeyMappingProvider) {
	provider.RegisterMapping(KBD_TYPE_LAPTOP, GetLaptopKeyMapping())
}
