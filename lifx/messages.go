// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package lifx

type Power uint16

const (
	On  = Power(0xFFFF)
	Off = Power(0x0000)
)

func (p Power) String() string {
	if p == On {
		return "on"
	}
	if p == Off {
		return "off"
	}
	return "invalid Power value"
}

type hsbk struct {
	// Hue is 360° scaled from 0 to 65535.
	Hue uint16
	// Saturation is percentage scaled from 0 to 65535.
	Saturation uint16
	// Brightness is percentage scaled from 0 to 65535.
	Brightness uint16
	// Kelvin ranges from 2500 (warm) to 9000 (cool).
	Kelvin uint16
}

type getService struct{}

type stateService struct {
	// Service is always 1, for "UDP".
	Service uint8
	// Port is the preferred listening port.
	Port    uint32
}

type acknowledgement struct{}

type get struct{}

type setColor struct {
	Reserved1 uint8
	Color     hsbk
	// Duration is the transition time in milliseconds.
	Duration uint32
}

type state struct {
	Color     hsbk
	Reserved1 int16
	// Power must be either 0x0000 (off) or 0xFFFF (on).
	Power     uint16
	// Label is the bulb's human-readable label.
	Label     [32]byte
	Reserved2 uint64
}

type setPower struct {
	// Power must be either 0x0000 (off) or 0xFFFF (on).
	Power uint16
	// Duration is the transition time in milliseconds.
	Duration uint32
}

type statePower struct {
	// Level must be either 0x0000 (off) or 0xFFFF (on).
	Level uint16
}

func typeForMessage(message interface{}) uint16 {
	switch message.(type) {
	case *getService:
		return 2
	case *stateService:
		return 3
	case *acknowledgement:
		return 45
	case *get:
		return 101
	case *setColor:
		return 102
	case *state:
		return 107
	case *setPower:
		return 117
	case *statePower:
		return 118
	default:
		panic("unknown Lifx message type")
	}
}

func messageForType(typ uint16) interface{} {
	switch typ {
	case 2:
		return &getService{}
	case 3:
		return &stateService{}
	case 45:
		return &acknowledgement{}
	case 101:
		return &get{}
	case 102:
		return &setColor{}
	case 107:
		return &state{}
	case 117:
		return &setPower{}
	case 118:
		return &statePower{}
	default:
		return nil
	}
}