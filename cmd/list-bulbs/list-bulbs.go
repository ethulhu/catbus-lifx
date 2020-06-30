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
	timeout = flag.String("timeout", "10s", "how long to wait for bulbs to respond")
)

func main() {
	flag.Parse()

	d, err := time.ParseDuration(*timeout)
	if err != nil {
		log.Fatalf("invalid duration %q for --timeout: %v", *timeout, err)
	}

	ctx, _ := context.WithTimeout(context.Background(), d)
	bulbs, err := lifx.Discover(ctx)
	if err != nil {
		log.Fatalf("failed to discover bulbs: %v", err)
	}

	var stats []string
	for bulb := range bulbs {
		ctx, _ := context.WithTimeout(context.Background(), d)
		state, err := bulb.State(ctx)
		if err != nil {
			log.Printf("a bulb was discovered but we failed to query it: %v", err)
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
