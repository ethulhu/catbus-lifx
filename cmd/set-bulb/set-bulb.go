// Binary set-bulb sets color properties for named Lifx bulbs.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/ethulhu/mqtt-lifx-bridge/lifx"
)

var (
	bulbLabel = flag.String("bulb", "", "bulb to change")

	power      = flag.String("power", "", "on or off")
	hue        = flag.Int("hue", -1, "0 – 359°")
	saturation = flag.Int("saturation", -1, "0 – 100%")
	brightness = flag.Int("brightness", -1, "0 – 100%")
	kelvin     = flag.Int("kelvin", -1, "2500K – 9000K")

	timeout  = flag.Duration("timeout", 10*time.Second, "how long to wait for bulbs to respond")
	duration = flag.Duration("duration", 500*time.Millisecond, "how long to smooth transitions over")
)

func main() {
	flag.Parse()

	if *bulbLabel == "" {
		log.Fatal("must set --bulb")
	}

	if *power != "" && !(*power == "on" || *power == "off") {
		log.Fatalf("power must be on or off, found %v", *power)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	bulbs, err := lifx.Discover(ctx)
	if err != nil {
		log.Fatalf("failed to discover bulbs: %v", err)
	}

	var bulb lifx.Bulb
	var state lifx.State
	for b := range bulbs {
		ctx, _ := context.WithTimeout(context.Background(), *timeout)
		s, err := b.State(ctx)
		if err != nil {
			continue
		}
		if s.Label == *bulbLabel {
			bulb = b
			state = s
			break
		}
	}
	cancel()
	if bulb == nil {
		log.Fatalf("could not find bulb %q", *bulbLabel)
	}

	colorChange := *hue != -1 || *saturation != 0 || *brightness != 0 || *kelvin != 0

	color := state.Color
	if *hue != -1 {
		color.Hue = *hue
	}
	if *saturation != -1 {
		color.Saturation = *saturation
	}
	if *brightness != -1 {
		color.Brightness = *brightness
	}
	if *kelvin != -1 {
		color.Kelvin = *kelvin
	}

	// if color && power=on, do color first.
	if colorChange && *power == "on" {
		ctx, _ = context.WithTimeout(context.Background(), *timeout)
		if err := bulb.SetColor(ctx, color, 0); err != nil {
			log.Fatalf("failed to set color: %v", err)
		}
		ctx, _ = context.WithTimeout(context.Background(), *timeout)
		if err := bulb.SetPower(ctx, lifx.On, *duration); err != nil {
			log.Fatalf("failed to set power: %v", err)
		}
	} else {
		if *power == "on" {
			ctx, _ = context.WithTimeout(context.Background(), *timeout)
			if err := bulb.SetPower(ctx, lifx.On, *duration); err != nil {
				log.Fatalf("failed to set power: %v", err)
			}
		}
		if *power == "off" {
			ctx, _ = context.WithTimeout(context.Background(), *timeout)
			if err := bulb.SetPower(ctx, lifx.Off, *duration); err != nil {
				log.Fatalf("failed to set power: %v", err)
			}
		}

		if colorChange {
			ctx, _ = context.WithTimeout(context.Background(), *timeout)
			if err := bulb.SetColor(ctx, color, *duration); err != nil {
				log.Fatalf("failed to set color: %v", err)
			}
		}
	}
}
