// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"flag"
	"log"
	"strconv"
	"sync"
	"time"

	"go.eth.moe/catbus"
	"go.eth.moe/catbus-lifx/config"
	"go.eth.moe/catbus-lifx/lifx"
)

var (
	configPath = flag.String("config-path", "", "path to config.json")
)

func main() {
	flag.Parse()

	if *configPath == "" {
		log.Fatal("must set --config-path")
	}

	config, err := config.ParseFile(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	var bulbs sync.Map

	go discoverBulbs(&bulbs)
	go func(c <-chan time.Time) {
		for _ = range c {
			discoverBulbs(&bulbs)
		}
	}(time.Tick(5 * time.Minute))

	broker := catbus.NewClient(config.BrokerURI, catbus.ClientOptions{
		ConnectHandler: func(broker catbus.Client) {
			log.Printf("connected to MQTT broker %s", config.BrokerURI)

			for label, bulb := range config.BulbsByLabel {
				if err := broker.Subscribe(bulb.Topics.Power, setPower(&bulbs, label)); err != nil {
					log.Printf("could not subscribe to power: %v", err)
				}
				if err := broker.Subscribe(bulb.Topics.Hue, setHue(&bulbs, label)); err != nil {
					log.Printf("could not subscribe to hue: %v", err)
				}
				if err := broker.Subscribe(bulb.Topics.Saturation, setSaturation(&bulbs, label)); err != nil {
					log.Printf("could not subscribe to saturation: %v", err)
				}
				if err := broker.Subscribe(bulb.Topics.Brightness, setBrightness(&bulbs, label)); err != nil {
					log.Printf("could not subscribe to brightness: %v", err)
				}
				if err := broker.Subscribe(bulb.Topics.Kelvin, setKelvin(&bulbs, label)); err != nil {
					log.Printf("could not subscribe to kelvin: %v", err)
				}
			}
		},
		DisconnectHandler: func(_ catbus.Client, err error) {
			log.Printf("disconnected from MQTT broker %s: %v", config.BrokerURI, err)
		},
	})

	go publishBulbStates(config, broker, &bulbs)
	go func(c <-chan time.Time) {
		for _ = range c {
			publishBulbStates(config, broker, &bulbs)
		}
	}(time.Tick(30 * time.Second))

	log.Printf("connecting to MQTT broker %v", config.BrokerURI)
	if err := broker.Connect(); err != nil {
		log.Fatalf("could not connect to MQTT broker: %v", err)
	}
}

func findBulb(bulbs *sync.Map, label string) (lifx.Bulb, bool) {
	maybeBulb, ok := bulbs.Load(label)
	if !ok {
		return nil, false
	}
	return maybeBulb.(lifx.Bulb), true
}

func discoverBulbs(bulbs *sync.Map) {
	log.Print("discovering Lifx bulbs")

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	bulbHandles, err := lifx.Discover(ctx)
	if err != nil {
		log.Printf("failed to refresh bulb handles: %v", err)
		return
	}

	for bulb := range bulbHandles {
		state, err := bulb.State(ctx)
		if err != nil {
			log.Printf("failed to read bulb state during discovery: %v", err)
			continue
		}
		log.Printf("found bulb: %v", state.Label)

		bulbs.Store(state.Label, bulb)
	}
}

func publishBulbStates(config *config.Config, broker catbus.Client, bulbs *sync.Map) {
	for label, bulbConfig := range config.BulbsByLabel {
		bulb, ok := findBulb(bulbs, label)
		if !ok {
			continue
		}

		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		state, err := bulb.State(ctx)
		if err != nil {
			log.Printf("%v: failed to read bulb state: %v", label, err)
			continue
		}

		if err := broker.Publish(bulbConfig.Topics.Power, catbus.Retain, state.Power.String()); err != nil {
			log.Printf("%v: could not publish power: %v", label, err)
		}
		if err := broker.Publish(bulbConfig.Topics.Hue, catbus.Retain, strconv.Itoa(state.Color.Hue)); err != nil {
			log.Printf("%v: could not publish hue: %v", label, err)
		}
		if err := broker.Publish(bulbConfig.Topics.Saturation, catbus.Retain, strconv.Itoa(state.Color.Saturation)); err != nil {
			log.Printf("%v: could not publish saturation: %v", label, err)
		}
		if err := broker.Publish(bulbConfig.Topics.Brightness, catbus.Retain, strconv.Itoa(state.Color.Brightness)); err != nil {
			log.Printf("%v: could not publish brightness: %v", label, err)
		}
		if err := broker.Publish(bulbConfig.Topics.Kelvin, catbus.Retain, strconv.Itoa(state.Color.Kelvin)); err != nil {
			log.Printf("%v: could not publish kelvin: %v", label, err)
		}
	}
}

func parseNumber(raw string) (int, error) {
	float, err := strconv.ParseFloat(raw, 64)
	return int(float), err
}

func setPower(bulbs *sync.Map, label string) catbus.MessageHandler {
	return func(_ catbus.Client, msg catbus.Message) {
		bulb, ok := findBulb(bulbs, label)
		if !ok {
			return
		}

		var power lifx.Power
		switch msg.Payload {
		case "on":
			power = lifx.On
		case "off":
			power = lifx.Off
		default:
			return
		}

		ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
		if err := bulb.SetPower(ctx, power, 500*time.Millisecond); err != nil {
			log.Printf("%s: failed to set power: %v", label, err)
		}
	}
}
func setHue(bulbs *sync.Map, label string) catbus.MessageHandler {
	return func(_ catbus.Client, msg catbus.Message) {
		bulb, ok := findBulb(bulbs, label)
		if !ok {
			return
		}

		hue, err := parseNumber(msg.Payload)
		if err != nil {
			return
		}
		for hue < 0 {
			hue += 360
		}
		if hue > 359 {
			hue = hue % 360
		}

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		state, err := bulb.State(ctx)
		if err != nil {
			log.Printf("%s: failed to get bulb state: %v", label, err)
			return
		}

		color := state.Color
		color.Hue = hue

		if err := bulb.SetColor(ctx, color, 100*time.Millisecond); err != nil {
			log.Printf("%s: failed to set hue: %v", label, err)
		}
	}
}
func setSaturation(bulbs *sync.Map, label string) catbus.MessageHandler {
	return func(_ catbus.Client, msg catbus.Message) {
		bulb, ok := findBulb(bulbs, label)
		if !ok {
			return
		}

		saturation, err := parseNumber(msg.Payload)
		if err != nil {
			return
		}
		if saturation < 0 {
			saturation = 0
		}
		if saturation > 100 {
			saturation = 100
		}

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		state, err := bulb.State(ctx)
		if err != nil {
			log.Printf("%s: failed to get bulb state: %v", label, err)
			return
		}

		color := state.Color
		color.Saturation = saturation

		if err := bulb.SetColor(ctx, color, 100*time.Millisecond); err != nil {
			log.Printf("%s: failed to set saturation: %v", label, err)
		}
	}
}
func setBrightness(bulbs *sync.Map, label string) catbus.MessageHandler {
	return func(_ catbus.Client, msg catbus.Message) {
		bulb, ok := findBulb(bulbs, label)
		if !ok {
			return
		}

		brightness, err := parseNumber(msg.Payload)
		if err != nil {
			return
		}
		if brightness < 0 {
			brightness = 0
		}
		if brightness > 100 {
			brightness = 100
		}

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		state, err := bulb.State(ctx)
		if err != nil {
			log.Printf("%s: failed to get bulb state: %v", label, err)
			return
		}

		color := state.Color
		color.Brightness = brightness

		if err := bulb.SetColor(ctx, color, 100*time.Millisecond); err != nil {
			log.Printf("%s: failed to set brightness: %v", label, err)
		}
	}
}
func setKelvin(bulbs *sync.Map, label string) catbus.MessageHandler {
	return func(_ catbus.Client, msg catbus.Message) {
		bulb, ok := findBulb(bulbs, label)
		if !ok {
			return
		}

		kelvin, err := parseNumber(msg.Payload)
		if err != nil {
			return
		}
		if kelvin < 2500 {
			kelvin = 2500
		}
		if kelvin > 9000 {
			kelvin = 9000
		}

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		state, err := bulb.State(ctx)
		if err != nil {
			log.Printf("%s: failed to get bulb state: %v", label, err)
			return
		}

		color := state.Color
		color.Kelvin = kelvin

		if err := bulb.SetColor(ctx, color, 100*time.Millisecond); err != nil {
			log.Printf("%s: failed to set Kelvin: %v", label, err)
		}
	}
}
