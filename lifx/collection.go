package lifx

import (
	"fmt"
	"log"
	"sync"

	"github.com/2tvenom/golifx"
)

type Collection struct {
	mu sync.Mutex

	bulbs map[string]*Bulb
}

func NewCollection() *Collection {
	return &Collection{
		bulbs: map[string]*Bulb{},
	}
}

func (c *Collection) Refresh() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	bulbs, err := golifx.LookupBulbs()
	if err != nil {
		return fmt.Errorf("failed to look up bulbs: %w", err)
	}
	for _, bulb := range bulbs {
		bulb := NewBulb(bulb)
		state, err := bulb.State()
		if err != nil {
			return fmt.Errorf("failed to read bulb state: %v", err)
		}
		c.bulbs[state.Label] = bulb
		log.Printf("found bulb %v", state.Label)
	}
	return nil
}

func (c *Collection) Bulb(label string) (*Bulb, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bulb, ok := c.bulbs[label]
	return bulb, ok
}
