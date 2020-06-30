// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

// Binary catbus-lifx-actuator controls Lifx bulbs via Catbus.
package main

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"go.eth.moe/catbus"
	"go.eth.moe/catbus-lifx/config"
	"go.eth.moe/catbus-lifx/lifx"
	"go.eth.moe/flag"
)

var (
	configPath = flag.Custom("config-path", "", "path to config.json", flag.RequiredString)
)

var (
	bulbsByLabel   = map[string]lifx.Bulb{}
	bulbsByLabelMu sync.Mutex
)

func main() {
	flag.Parse()

	configPath := (*configPath).(string)

	config, err := config.ParseFile(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	go discoverBulbs()
	go func() {
		for range time.Tick(30 * time.Second) {
			discoverBulbs()
		}
	}()

	broker := catbus.NewClient(config.BrokerURI, catbus.ClientOptions{
		ConnectHandler: func(broker catbus.Client) {
			log.Printf("connected to MQTT broker %v", config.BrokerURI)

			for label, bulb := range config.BulbsByLabel {
				if err := broker.Subscribe(bulb.Topics.Power, setPower(label)); err != nil {
					log.Printf("could not subscribe to power: %v", err)
				}
				if err := broker.Subscribe(bulb.Topics.Hue, setHue(label)); err != nil {
					log.Printf("could not subscribe to hue: %v", err)
				}
				if err := broker.Subscribe(bulb.Topics.Saturation, setSaturation(label)); err != nil {
					log.Printf("could not subscribe to saturation: %v", err)
				}
				if err := broker.Subscribe(bulb.Topics.Brightness, setBrightness(label)); err != nil {
					log.Printf("could not subscribe to brightness: %v", err)
				}
				if err := broker.Subscribe(bulb.Topics.Kelvin, setKelvin(label)); err != nil {
					log.Printf("could not subscribe to kelvin: %v", err)
				}
			}
		},
		DisconnectHandler: func(_ catbus.Client, err error) {
			log.Printf("disconnected from MQTT broker %v: %v", config.BrokerURI, err)
		},
	})

	log.Print("connecting to MQTT broker %v", config.BrokerURI)
	if err := broker.Connect(); err != nil {
		log.Fatalf("could not connect to MQTT broker %v: %v", config.BrokerURI, err)
	}
}

func discoverBulbs() {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	bulbs, err := lifx.Discover(ctx)
	if err != nil {
		log.Printf("could not discover bulbs: %v", err)
		return
	}
	log.Print("found bulbs")

	for bulb := range bulbs {
		bulb := bulb
		go func() {
			ctx := context.Background()
			state, err := bulb.State(ctx)
			if err != nil {
				log.Printf("could not read bulb state: %v", err)
				return
			}
			log.Printf("found bulb: %v", state.Label)

			bulbsByLabelMu.Lock()
			defer bulbsByLabelMu.Unlock()
			bulbsByLabel[state.Label] = bulb
		}()
	}
}
func findBulb(label string) (lifx.Bulb, bool) {
	bulbsByLabelMu.Lock()
	defer bulbsByLabelMu.Unlock()
	bulb, ok := bulbsByLabel[label]
	return bulb, ok
}

func parseNumber(raw string) (int, error) {
	float, err := strconv.ParseFloat(raw, 64)
	return int(float), err
}

func setPower(label string) catbus.MessageHandler {
	return func(_ catbus.Client, msg catbus.Message) {
		bulb, ok := findBulb(label)
		if !ok {
			log.Printf("could not find bulb: %v", label)
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
func setHue(label string) catbus.MessageHandler {
	return func(_ catbus.Client, msg catbus.Message) {
		bulb, ok := findBulb(label)
		if !ok {
			log.Printf("could not find bulb: %v", label)
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
func setSaturation(label string) catbus.MessageHandler {
	return func(_ catbus.Client, msg catbus.Message) {
		bulb, ok := findBulb(label)
		if !ok {
			log.Printf("could not find bulb: %v", label)
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
func setBrightness(label string) catbus.MessageHandler {
	return func(_ catbus.Client, msg catbus.Message) {
		bulb, ok := findBulb(label)
		if !ok {
			log.Printf("could not find bulb: %v", label)
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
func setKelvin(label string) catbus.MessageHandler {
	return func(_ catbus.Client, msg catbus.Message) {
		bulb, ok := findBulb(label)
		if !ok {
			log.Printf("could not find bulb: %v", label)
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
