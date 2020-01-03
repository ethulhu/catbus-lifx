package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/2tvenom/golifx"
)

type Bulb struct {
	*golifx.Bulb
	sync.Mutex

	state       *golifx.BulbState
	lastUpdated time.Time
}

func NewBulb(bulb *golifx.Bulb) *Bulb {
	return &Bulb{
		Bulb: bulb,
	}
}

func (b *Bulb) State() (*golifx.BulbState, error) {
	b.Lock()
	defer b.Unlock()
	if b.state == nil || b.lastUpdated.Before(time.Now().Add(-5*time.Second)) {
		state, err := b.GetColorState()
		if err != nil {
			return nil, err
		}
		b.state = state
	}
	return b.state, nil
}
func (b *Bulb) SetColor(hsbk *golifx.HSBK, duration uint32) error {
	b.Lock()
	defer b.Unlock()
	state, err := b.SetColorStateWithResponse(hsbk, duration)
	if err == nil {
		b.state = state
	}
	return err
}

func (b *Bulb) SetHue(h uint16) error {
	state, err := b.State()
	if err != nil {
		return fmt.Errorf("failed to get bulb state: %w", err)
	}

	hsbk := state.Color
	hsbk.Hue = h
	return b.SetColorState(hsbk, 500)
}
func (b *Bulb) SetSaturation(s uint16) error {
	state, err := b.State()
	if err != nil {
		return fmt.Errorf("failed to get bulb state: %w", err)
	}

	hsbk := state.Color
	hsbk.Saturation = s
	return b.SetColorState(hsbk, 500)
}
func (b *Bulb) SetBrightness(br uint16) error {
	state, err := b.State()
	if err != nil {
		return fmt.Errorf("failed to get bulb state: %w", err)
	}

	hsbk := state.Color
	hsbk.Brightness = br
	return b.SetColorState(hsbk, 500)
}
func (b *Bulb) SetKelvin(k uint16) error {
	state, err := b.State()
	if err != nil {
		return fmt.Errorf("failed to get bulb state: %w", err)
	}

	hsbk := state.Color
	hsbk.Kelvin = k
	return b.SetColorState(hsbk, 500)
}
