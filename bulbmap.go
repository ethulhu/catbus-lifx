package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/2tvenom/golifx"
	"github.com/ethulhu/mqtt-lifx-bridge/lifx"
)

type bulbMap struct {
	sync.Mutex

	bulbs map[string]*lifx.Bulb
}

func newBulbMap() *bulbMap {
	return &bulbMap{
		bulbs: map[string]*lifx.Bulb{},
	}
}

func (m *bulbMap) refresh() error {
	m.Lock()
	defer m.Unlock()

	bulbs, err := golifx.LookupBulbs()
	if err != nil {
		return fmt.Errorf("failed to look up bulbs: %w", err)
	}
	for _, bulb := range bulbs {
		bulb := lifx.NewBulb(bulb)
		state, err := bulb.State()
		if err != nil {
			return fmt.Errorf("failed to read bulb state: %v", err)
		}
		m.bulbs[state.Label] = bulb
		log.Printf("found bulb %v", state.Label)
	}
	return nil
}

func (m *bulbMap) bulb(label string) (*lifx.Bulb, bool) {
	m.Lock()
	defer m.Unlock()

	bulb, ok := m.bulbs[label]
	return bulb, ok
}
