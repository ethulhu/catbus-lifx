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

	"github.com/ethulhu/catbus-lifx/lifx"
	"github.com/ethulhu/catbus-lifx/mqtt"
)

var (
	configPath = flag.String("config-path", "", "path to config.json")
)

func main() {
	flag.Parse()

	if *configPath == "" {
		log.Fatal("must set --config-path")
	}

	config, err := loadConfig(*configPath)
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

	brokerURI := mqtt.URI(config.BrokerHost, config.BrokerPort)
	brokerOptions := mqtt.NewClientOptions()
	brokerOptions.AddBroker(brokerURI)
	brokerOptions.SetAutoReconnect(true)
	brokerOptions.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		log.Printf("disconnected from MQTT broker %s: %v", brokerURI, err)
	})
	brokerOptions.SetOnConnectHandler(func(broker mqtt.Client) {
		log.Printf("connected to MQTT broker %s", brokerURI)

		for _, light := range config.Lights {
			broker.Subscribe(light.TopicPower, mqtt.AtLeastOnce, setPower(&bulbs, light.BulbLabel))
			broker.Subscribe(light.TopicHue, mqtt.AtLeastOnce, setHue(&bulbs, light.BulbLabel))
			broker.Subscribe(light.TopicSaturation, mqtt.AtLeastOnce, setSaturation(&bulbs, light.BulbLabel))
			broker.Subscribe(light.TopicBrightness, mqtt.AtLeastOnce, setBrightness(&bulbs, light.BulbLabel))
			broker.Subscribe(light.TopicKelvin, mqtt.AtLeastOnce, setKelvin(&bulbs, light.BulbLabel))
		}
	})

	log.Printf("connecting to MQTT broker %v", brokerURI)
	broker := mqtt.NewClient(brokerOptions)
	_ = broker.Connect()

	go publishBulbStates(config, broker, &bulbs)
	go func(c <-chan time.Time) {
		for _ = range c {
			publishBulbStates(config, broker, &bulbs)
		}
	}(time.Tick(30 * time.Second))

	// block forever.
	select {}
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

func publishBulbStates(config *Config, broker mqtt.Client, bulbs *sync.Map) {
	for _, light := range config.Lights {
		bulb, ok := findBulb(bulbs, light.BulbLabel)
		if !ok {
			continue
		}

		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		state, err := bulb.State(ctx)
		if err != nil {
			log.Printf("%v: failed to read bulb state: %v", light.BulbLabel, err)
			continue
		}

		broker.Publish(light.TopicPower, mqtt.AtLeastOnce, mqtt.Retain, state.Power.String())
		broker.Publish(light.TopicHue, mqtt.AtLeastOnce, mqtt.Retain, strconv.Itoa(state.Color.Hue))
		broker.Publish(light.TopicSaturation, mqtt.AtLeastOnce, mqtt.Retain, strconv.Itoa(state.Color.Saturation))
		broker.Publish(light.TopicBrightness, mqtt.AtLeastOnce, mqtt.Retain, strconv.Itoa(state.Color.Brightness))
		broker.Publish(light.TopicKelvin, mqtt.AtLeastOnce, mqtt.Retain, strconv.Itoa(state.Color.Kelvin))
	}
}

func parseNumber(raw string) (int, error) {
	float, err := strconv.ParseFloat(raw, 64)
	return int(float), err
}

func setPower(bulbs *sync.Map, label string) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		bulb, ok := findBulb(bulbs, label)
		if !ok {
			return
		}

		var power lifx.Power
		switch string(msg.Payload()) {
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
func setHue(bulbs *sync.Map, label string) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		bulb, ok := findBulb(bulbs, label)
		if !ok {
			return
		}

		hue, err := parseNumber(string(msg.Payload()))
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
func setSaturation(bulbs *sync.Map, label string) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		bulb, ok := findBulb(bulbs, label)
		if !ok {
			return
		}

		saturation, err := parseNumber(string(msg.Payload()))
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
func setBrightness(bulbs *sync.Map, label string) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		bulb, ok := findBulb(bulbs, label)
		if !ok {
			return
		}

		brightness, err := parseNumber(string(msg.Payload()))
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
func setKelvin(bulbs *sync.Map, label string) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		bulb, ok := findBulb(bulbs, label)
		if !ok {
			return
		}

		kelvin, err := parseNumber(string(msg.Payload()))
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
