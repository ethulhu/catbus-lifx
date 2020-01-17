// Package lifx controls Lifx bulbs using the Lifx LAN protocol.
package lifx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	MinHue        = 0
	MaxHue        = 359
	MinSaturation = 0
	MaxSaturation = 100
	MinBrightness = 0
	MaxBrightness = 100
	MinKelvin     = 2500
	MaxKelvin     = 9000

	maxUint16       = int(^uint16(0))
	hueScale        = maxUint16 / MaxHue
	saturationScale = maxUint16 / MaxSaturation
	brightnessScale = maxUint16 / MaxBrightness
)

type (
	// HSBK is a Lifx color.
	// All four parts can change the color at once.
	HSBK struct {
		// Hue ranges from 0° to 359°.
		Hue int
		// Saturation ranges from 0 to 100.
		Saturation int
		// Brightness ranges from 0 to 100.
		Brightness int
		// Kelvin ranges from 2500K to 9000K.
		Kelvin int
	}

	// State is the state of a given bulb at a given time.
	State struct {
		Label string
		Power Power
		Color HSBK
	}

	// Bulb is a Lifx bulb.
	Bulb interface {
		// State returns the current State of the bulb.
		State(context.Context) (State, error)
		// SetPower sets the power, with a duration to smooth the change over.
		SetPower(context.Context, Power, time.Duration) error
		// SetColor sets the color, with a duration to smooth the change over.
		SetColor(context.Context, HSBK, time.Duration) error
	}
)

var ErrNoResponse = errors.New("no response from bulb")

type ErrInvalidColor struct {
	hue        int
	saturation int
	brightness int
	kelvin     int
}

func (e *ErrInvalidColor) Error() string {
	var parts []string
	if e.hue != 0 {
		parts = append(parts, fmt.Sprintf("hue must be within [0,359], found %v", e.hue))
	}
	if e.saturation != 0 {
		parts = append(parts, fmt.Sprintf("saturation must be within [0,100], found %v", e.saturation))
	}
	if e.brightness != 0 {
		parts = append(parts, fmt.Sprintf("brightness must be within [0,100], found %v", e.brightness))
	}
	if e.kelvin != 0 {
		parts = append(parts, fmt.Sprintf("kelvin must be within [2500,9000], found %v", e.kelvin))
	}
	return fmt.Sprintf("invalid color: %v", strings.Join(parts, "; "))
}
func (e *ErrInvalidColor) ok() bool {
	return e.hue == 0 && e.saturation == 0 && e.brightness == 0 && e.kelvin == 0
}

func prettyState(s *state) State {
	return State{
		Label: string(bytes.Trim(s.Label[:], "\x00")),
		Power: Power(s.Power),
		Color: HSBK{
			Hue:        int(s.Color.Hue) / hueScale,
			Saturation: int(s.Color.Saturation) / saturationScale,
			Brightness: int(s.Color.Brightness) / brightnessScale,
			Kelvin:     int(s.Color.Kelvin),
		},
	}
}
func uglyHSBK(color HSBK) (hsbk, error) {
	err := &ErrInvalidColor{}
	if !(MinHue <= color.Hue && color.Hue <= MaxHue) {
		err.hue = color.Hue
	}
	if !(MinSaturation <= color.Saturation && color.Saturation <= MaxSaturation) {
		err.saturation = color.Saturation
	}
	if !(MinBrightness <= color.Brightness && color.Brightness <= MaxBrightness) {
		err.brightness = color.Brightness
	}
	if !(MinKelvin <= color.Kelvin && color.Kelvin <= MaxKelvin) {
		err.kelvin = color.Kelvin
	}
	if !err.ok() {
		return hsbk{}, err
	}

	return hsbk{
		Hue:        uint16(color.Hue * hueScale),
		Saturation: uint16(color.Saturation * saturationScale),
		Brightness: uint16(color.Brightness * brightnessScale),
		Kelvin:     uint16(color.Kelvin),
	}, nil
}
