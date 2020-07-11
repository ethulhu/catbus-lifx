// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

// Binary list-bulbs lists Lifx bulbs on the local network.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"go.eth.moe/catbus-lifx/lifx"
)

var (
	timeout = flag.Duration("timeout", 10*time.Second, "how long to wait for bulbs to respond")
)

func main() {
	flag.Parse()

	timeout := *timeout

	ctx, _ := context.WithTimeout(context.Background(), timeout)
	bulbs, err := lifx.Discover(ctx)
	if err != nil {
		log.Fatalf("could not discover bulbs: %v", err)
	}

	var stats []string
	for _, bulb := range bulbs {
		ctx, _ := context.WithTimeout(context.Background(), timeout)
		state, err := bulb.State(ctx)
		if err != nil {
			log.Printf("a bulb was discovered but we could not query it: %v", err)
			continue
		}

		stats = append(stats, fmt.Sprintf(`%s:
	power:      %v
	hue:        %v°
	saturation: %v%%
	brightness: %v%%
	kelvin:     %vK`, state.Label, state.Power, state.Color.Hue, state.Color.Saturation, state.Color.Brightness, state.Color.Kelvin))
	}

	sort.Strings(stats)
	fmt.Println(strings.Join(stats, "\n\n"))
}
