package keymaps

// GetPhoneKeyMapping returns key mappings for phone-type keyboards
func GetPhoneKeyMapping() KeyMapping {
	type keyAddresses struct {
		Key1        uint16
		Key2        uint16
		Key3        uint16
		Key4        uint16
		Key5        uint16
		Key6        uint16
		Key7        uint16
		Key8        uint16
		Key9        uint16
		Key0        uint16
		AsteriskKey uint16
		HashKey     uint16

		StarKey      uint16
		MailKey      uint16
		SoftLeftKey  uint16
		SoftRightKey uint16

		CallKey    uint16
		EndCallKey uint16

		VolumeUpKey   uint16
		VolumeDownKey uint16
		EnterKey      uint16
		UpKey         uint16
		DownKey       uint16
		LeftKey       uint16
		RightKey      uint16
	}
	ka := keyAddresses{
		// Numberpad
		Key1:        2,
		Key2:        3,
		Key3:        4,
		Key4:        5,
		Key5:        6,
		Key6:        7,
		Key7:        8,
		Key8:        9,
		Key9:        10,
		Key0:        11,
		AsteriskKey: 522,
		HashKey:     523,

		// Shortcuts
		StarKey:      138,
		MailKey:      30,
		SoftLeftKey:  139,
		SoftRightKey: 48,

		// Call Keys
		CallKey:    231,
		EndCallKey: 116,

		VolumeUpKey:   115,
		VolumeDownKey: 114,
		EnterKey:      28,
		UpKey:         103,
		DownKey:       108,
		LeftKey:       105,
		RightKey:      106,
	}
	n := KeyMapping{}
	n.ExitKey = ka.EndCallKey
	n.EnterKey = ka.EnterKey
	n.ToggleMouseKey = ka.StarKey
	n.ClickKey = ka.EnterKey
	n.DragKey = ka.SoftRightKey
	n.FasterKey = ka.VolumeDownKey
	n.SlowerKey = ka.VolumeUpKey
	n.UpKey = ka.UpKey
	n.DownKey = ka.DownKey
	n.LeftKey = ka.LeftKey
	n.RightKey = ka.RightKey
	n.ScrollUpKey = ka.SoftLeftKey
	n.ScrollDownKey = ka.CallKey
	// Disabled
	n.ScrollLeftKey = 0
	n.ScrollRightKey = 0

	return n
}

// RegisterPhoneKeyMapping registers phone keyboard mapping with the provider
func RegisterPhoneKeyMapping(provider *KeyMappingProvider) {
	provider.RegisterMapping(KBD_TYPE_PHONE, GetPhoneKeyMapping())
}
