package keymaps

// GetLaptopKeyMapping returns key mappings for laptop-type keyboards
func GetLaptopKeyMapping() KeyMapping {
	n := KeyMapping{}
	n.ExitKey = 1 // Esc
	n.EnterKey = 28
	n.ToggleMouseKey = 29
	n.DragKey = 32        // D key
	n.FasterKey = 13      // = key
	n.SlowerKey = 12      // - key
	n.UpKey = 103         // up arrow
	n.DownKey = 108       // down arrow
	n.LeftKey = 105       // left arrow
	n.RightKey = 106      // right arrow
	n.ScrollUpKey = 17    // w key
	n.ScrollDownKey = 31  // s key
	n.ScrollLeftKey = 30  // a key
	n.ScrollRightKey = 32 // d key
	return n
}

// RegisterLaptopKeyMapping registers laptop keyboard mapping with the provider
func RegisterLaptopKeyMapping(provider *KeyMappingProvider) {
	provider.RegisterMapping(KBD_TYPE_LAPTOP, GetLaptopKeyMapping())
}
