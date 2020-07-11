// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

// Binary catbus-lifx-observer observes Lifx bulbs for Catbus.
package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"go.eth.moe/catbus"
	"go.eth.moe/catbus-lifx/config"
	"go.eth.moe/catbus-lifx/lifx"
	"go.eth.moe/flag"
)

var (
	configPath = flag.Custom("config-path", "", "path to config.json", flag.RequiredString)
)

func main() {
	flag.Parse()

	configPath := (*configPath).(string)

	config, err := config.ParseFile(configPath)
	if err != nil {
		log.Fatalf("could not load config: %v", err)
	}

	broker := catbus.NewClient(config.BrokerURI, catbus.ClientOptions{
		ConnectHandler: func(_ catbus.Client) {
			log.Printf("connected to MQTT broker %v", config.BrokerURI)
		},
		DisconnectHandler: func(_ catbus.Client, err error) {
			log.Printf("disconnected from MQTT broker %v: %v", config.BrokerURI, err)
		},
	})

	go func() {
		log.Print("connecting to MQTT broker %v", config.BrokerURI)
		if err := broker.Connect(); err != nil {
			log.Fatalf("could not connect to MQTT broker %v: %v", config.BrokerURI, err)
		}
	}()

	publishBulbStates(config, broker)
	for range time.Tick(30 * time.Second) {
		publishBulbStates(config, broker)
	}
}

func publishBulbStates(config *config.Config, broker catbus.Client) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	bulbs, err := lifx.Discover(ctx)
	if err != nil {
		log.Printf("could not discover bulbs: %v", err)
		return
	}
	log.Print("found bulbs")

	for _, bulb := range bulbs {
		ctx := context.Background()
		state, err := bulb.State(ctx)
		if err != nil {
			log.Printf("could not read bulb state: %v", err)
			continue
		}
		log.Printf("found bulb: %v", state.Label)

		bulbConfig, ok := config.BulbsByLabel[state.Label]
		if !ok {
			log.Printf("found bulb with no config: %v", state.Label)
			continue
		}

		if err := broker.Publish(bulbConfig.Topics.Power, catbus.Retain, state.Power.String()); err != nil {
			log.Printf("%v: could not publish power: %v", state.Label, err)
		}
		if err := broker.Publish(bulbConfig.Topics.Hue, catbus.Retain, strconv.Itoa(state.Color.Hue)); err != nil {
			log.Printf("%v: could not publish hue: %v", state.Label, err)
		}
		if err := broker.Publish(bulbConfig.Topics.Saturation, catbus.Retain, strconv.Itoa(state.Color.Saturation)); err != nil {
			log.Printf("%v: could not publish saturation: %v", state.Label, err)
		}
		if err := broker.Publish(bulbConfig.Topics.Brightness, catbus.Retain, strconv.Itoa(state.Color.Brightness)); err != nil {
			log.Printf("%v: could not publish brightness: %v", state.Label, err)
		}
		if err := broker.Publish(bulbConfig.Topics.Kelvin, catbus.Retain, strconv.Itoa(state.Color.Kelvin)); err != nil {
			log.Printf("%v: could not publish kelvin: %v", state.Label, err)
		}
		log.Printf("published status: %v", state.Label)
	}
}
