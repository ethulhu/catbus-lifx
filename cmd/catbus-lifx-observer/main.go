// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

// Binary catbus-lifx-observer observes Lifx bulbs for Catbus.
package main

import (
	"context"
	"strconv"
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

func main() {
	flag.Parse()

	configPath := (*configPath).(string)

	log := logger.Background()

	config, err := config.ParseFile(configPath)
	if err != nil {
		log.AddField("config-path", configPath)
		log.WithError(err).Fatal("could not load config")
	}

	log.AddField("broker-uri", config.BrokerURI)
	broker := catbus.NewClient(config.BrokerURI, catbus.ClientOptions{
		ConnectHandler: func(_ catbus.Client) {
			log.Info("connected to MQTT broker")
		},
		DisconnectHandler: func(_ catbus.Client, err error) {
			log.WithError(err)
			log.Error("disconnected from MQTT broker")
		},
	})

	go func() {
		log.Info("connecting to MQTT broker")
		if err := broker.Connect(); err != nil {
			log.WithError(err).Fatal("could not connect to MQTT broker")
		}
	}()

	publishBulbStates(config, broker)
	for range time.Tick(30 * time.Second) {
		publishBulbStates(config, broker)
	}
}

func publishBulbStates(config *config.Config, broker catbus.Client) {
	log, ctx := logger.FromContext(context.Background())

	log.Info("discovering bulbs")
	discoverCtx, _ := context.WithTimeout(ctx, 10*time.Second)
	bulbs, err := lifx.Discover(discoverCtx)
	if err != nil {
		log.WithError(err).Error("could not discover bulbs")
		return
	}
	log.Info("discovered bulbs")

	ctx, _ = context.WithTimeout(ctx, 5*time.Second)
	for _, bulb := range bulbs {
		bulb := bulb
		go func() {
			state, err := bulb.State(ctx)
			if err != nil {
				log.WithError(err).Error("could not read bulb state")
				return
			}
			log.AddField("bulb", state.Label)
			log.Info("found bulb")

			bulbConfig, ok := config.BulbsByLabel[state.Label]
			if !ok {
				log.Warning("discovered bulb with no config")
				return
			}

			if err := broker.Publish(bulbConfig.Topics.Power, catbus.Retain, state.Power.String()); err != nil {
				log.WithError(err).Error("could not publish power")
			}
			if err := broker.Publish(bulbConfig.Topics.Hue, catbus.Retain, strconv.Itoa(state.Color.Hue)); err != nil {
				log.WithError(err).Error("could not publish hue")
			}
			if err := broker.Publish(bulbConfig.Topics.Saturation, catbus.Retain, strconv.Itoa(state.Color.Saturation)); err != nil {
				log.WithError(err).Error("could not publish saturation")
			}
			if err := broker.Publish(bulbConfig.Topics.Brightness, catbus.Retain, strconv.Itoa(state.Color.Brightness)); err != nil {
				log.WithError(err).Error("could not publish brightness")
			}
			if err := broker.Publish(bulbConfig.Topics.Kelvin, catbus.Retain, strconv.Itoa(state.Color.Kelvin)); err != nil {
				log.WithError(err).Error("could not publish kelvin")
			}
			log.Info("published bulb status")
		}()
	}
}
