// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

// Binary catbus-lifx-actuator controls Lifx bulbs via Catbus.
package main

import (
	"context"
	"strconv"
	"sync"
	"time"

	"go.eth.moe/catbus"
	"go.eth.moe/catbus-lifx/config"
	"go.eth.moe/catbus-lifx/lifx"
	"go.eth.moe/flag"
	"go.eth.moe/logger"
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

	log := logger.Background()

	config, err := config.ParseFile(configPath)
	if err != nil {
		log.AddField("config-path", configPath)
		log.WithError(err).Fatal("could not load config")
	}

	go discoverBulbs()
	go func() {
		for range time.Tick(30 * time.Second) {
			discoverBulbs()
		}
	}()

	broker := catbus.NewClient(config.BrokerURI, catbus.ClientOptions{
		ConnectHandler: func(broker catbus.Client) {
			log := logger.Background()
			log.AddField("broker-uri", config.BrokerURI)
			log.Info("connected to MQTT broker")

			for label, bulb := range config.BulbsByLabel {
				if err := broker.Subscribe(bulb.Topics.Power, setPower(label)); err != nil {
					log := log.WithError(err)
					log.AddField("topic", bulb.Topics.Power)
					log.Error("could not subscribe to power")
				}
				if err := broker.Subscribe(bulb.Topics.Hue, setHue(label)); err != nil {
					log := log.WithError(err)
					log.AddField("topic", bulb.Topics.Hue)
					log.Error("could not subscribe to hue")
				}
				if err := broker.Subscribe(bulb.Topics.Saturation, setSaturation(label)); err != nil {
					log := log.WithError(err)
					log.AddField("topic", bulb.Topics.Saturation)
					log.Error("could not subscribe to saturation")
				}
				if err := broker.Subscribe(bulb.Topics.Brightness, setBrightness(label)); err != nil {
					log := log.WithError(err)
					log.AddField("topic", bulb.Topics.Brightness)
					log.Error("could not subscribe to brightness")
				}
				if err := broker.Subscribe(bulb.Topics.Kelvin, setKelvin(label)); err != nil {
					log := log.WithError(err)
					log.AddField("topic", bulb.Topics.Kelvin)
					log.Error("could not subscribe to kelvin")
				}
			}
			log.Info("subscribed to all topics for all bulbs")
		},
		DisconnectHandler: func(_ catbus.Client, err error) {
			log := logger.Background()
			log.AddField("broker-uri", config.BrokerURI)
			log.Error("disconnected from MQTT broker")
		},
	})

	log.AddField("broker-uri", config.BrokerURI)
	log.Info("connecting to MQTT broker")
	if err := broker.Connect(); err != nil {
		log.WithError(err).Fatal("could not connect to MQTT broker")
	}
}

func discoverBulbs() {
	log, ctx := logger.FromContext(context.Background())

	log.Info("discovering bulbs")
	discoverCtx, _ := context.WithTimeout(ctx, 10*time.Second)
	bulbs, err := lifx.Discover(discoverCtx)
	if err != nil {
		log.WithError(err).Error("could not discover bulbs")
		return
	}
	log.Info("found bulbs")

	for bulb := range bulbs {
		bulb := bulb
		go func() {
			state, err := bulb.State(ctx)
			if err != nil {
				log.WithError(err).Error("could not read bulb state")
				return
			}
			log.AddField("bulb", state.Label)
			log.Info("found bulb")

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
		log, ctx := logger.FromContext(context.Background())
		log.AddField("bulb", label)
		log.AddField("payload", msg.Payload)

		bulb, ok := findBulb(label)
		if !ok {
			log.Error("could not find bulb")
			return
		}

		var power lifx.Power
		switch msg.Payload {
		case "on":
			power = lifx.On
		case "off":
			power = lifx.Off
		default:
			log.Warning("invalid power state")
			return
		}

		ctx, _ = context.WithTimeout(ctx, 2*time.Second)
		if err := bulb.SetPower(ctx, power, 500*time.Millisecond); err != nil {
			log.WithError(err).Error("could not set power")
			return
		}
		log.Info("set power")
	}
}
func setHue(label string) catbus.MessageHandler {
	return func(_ catbus.Client, msg catbus.Message) {
		log, ctx := logger.FromContext(context.Background())
		log.AddField("bulb", label)
		log.AddField("payload", msg.Payload)

		bulb, ok := findBulb(label)
		if !ok {
			log.Error("could not find bulb")
			return
		}

		hue, err := parseNumber(msg.Payload)
		if err != nil {
			log.Warning("invalid hue")
			return
		}
		for hue < 0 {
			hue += 360
		}
		if hue > 359 {
			hue = hue % 360
		}
		log.AddField("hue", hue)

		ctx, _ = context.WithTimeout(ctx, 5*time.Second)
		state, err := bulb.State(ctx)
		if err != nil {
			log.WithError(err).Error("could not get bulb state")
			return
		}

		color := state.Color
		color.Hue = hue

		if err := bulb.SetColor(ctx, color, 100*time.Millisecond); err != nil {
			log.WithError(err).Error("could not set hue")
			return
		}
		log.Info("set hue")
	}
}
func setSaturation(label string) catbus.MessageHandler {
	return func(_ catbus.Client, msg catbus.Message) {
		log, ctx := logger.FromContext(context.Background())
		log.AddField("bulb", label)
		log.AddField("payload", msg.Payload)

		bulb, ok := findBulb(label)
		if !ok {
			log.Error("could not find bulb")
			return
		}

		saturation, err := parseNumber(msg.Payload)
		if err != nil {
			log.Warning("invalid saturation")
			return
		}
		if saturation < 0 {
			saturation = 0
		}
		if saturation > 100 {
			saturation = 100
		}
		log.AddField("saturation", saturation)

		ctx, _ = context.WithTimeout(ctx, 5*time.Second)
		state, err := bulb.State(ctx)
		if err != nil {
			log.WithError(err).Error("could not get bulb state")
			return
		}

		color := state.Color
		color.Saturation = saturation

		if err := bulb.SetColor(ctx, color, 100*time.Millisecond); err != nil {
			log.WithError(err).Error("could not set saturation")
			return
		}
		log.Info("set saturation")
	}
}
func setBrightness(label string) catbus.MessageHandler {
	return func(_ catbus.Client, msg catbus.Message) {
		log, ctx := logger.FromContext(context.Background())
		log.AddField("bulb", label)
		log.AddField("payload", msg.Payload)

		bulb, ok := findBulb(label)
		if !ok {
			log.Error("could not find bulb")
			return
		}

		brightness, err := parseNumber(msg.Payload)
		if err != nil {
			log.Warning("invalid brightness")
			return
		}
		if brightness < 0 {
			brightness = 0
		}
		if brightness > 100 {
			brightness = 100
		}
		log.AddField("brightness", brightness)

		ctx, _ = context.WithTimeout(ctx, 5*time.Second)
		state, err := bulb.State(ctx)
		if err != nil {
			log.WithError(err).Error("could not get bulb state")
			return
		}

		color := state.Color
		color.Brightness = brightness

		if err := bulb.SetColor(ctx, color, 100*time.Millisecond); err != nil {
			log.WithError(err).Error("could not set brightness")
			return
		}
		log.Info("set brightness")
	}
}
func setKelvin(label string) catbus.MessageHandler {
	return func(_ catbus.Client, msg catbus.Message) {
		log, ctx := logger.FromContext(context.Background())
		log.AddField("bulb", label)
		log.AddField("payload", msg.Payload)

		bulb, ok := findBulb(label)
		if !ok {
			log.Error("could not find bulb")
			return
		}

		kelvin, err := parseNumber(msg.Payload)
		if err != nil {
			log.Warning("invalid kelvin")
			return
		}
		if kelvin < 2500 {
			kelvin = 2500
		}
		if kelvin > 9000 {
			kelvin = 9000
		}
		log.AddField("kelvin", kelvin)

		ctx, _ = context.WithTimeout(ctx, 5*time.Second)
		state, err := bulb.State(ctx)
		if err != nil {
			log.WithError(err).Error("could not get bulb state")
			return
		}

		color := state.Color
		color.Kelvin = kelvin

		if err := bulb.SetColor(ctx, color, 100*time.Millisecond); err != nil {
			log.WithError(err).Error("could not set kelvin")
			return
		}
		log.Info("set kelvin")
	}
}
