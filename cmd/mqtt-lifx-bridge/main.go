package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/ethulhu/mqtt-lifx-bridge/lifx"
	"github.com/ethulhu/mqtt-lifx-bridge/mqtt"
)

var (
	configPath = flag.String("config-path", "", "path to config.json")
)

func discoverBulbs(bulbs *sync.Map) {
	log.Print("discovering Lifx bulbs")

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	bulbHandles, err := lifx.Discover(ctx)
	if err != nil {
		log.Printf("failed to refresh bulb handles: %v", err)
	}
	if len(bulbHandles) == 0 {
		log.Print("found no bulbs")
	}

	for _, bulb := range bulbHandles {
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
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
		maybeBulb, ok := bulbs.Load(light.BulbLabel)
		if !ok {
			continue
		}
		bulb := maybeBulb.(lifx.Bulb)

		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		state, err := bulb.State(ctx)
		if err != nil {
			log.Printf("%v: failed to read bulb state: %v", light.BulbLabel, err)
			continue
		}

		publishState(broker, light, state)
	}
}

func main() {
	flag.Parse()

	if *configPath == "" {
		log.Fatal("must set --config-path")
	}

	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Printf("connecting to MQTT broker %v:%v", config.BrokerHost, config.BrokerPort)
	broker := mqtt.NewClient(config.BrokerHost, config.BrokerPort)

	var bulbs sync.Map

	discoverBulbs(&bulbs)
	go func(c <-chan time.Time) {
		for _ = range c {
			discoverBulbs(&bulbs)
		}
	}(time.Tick(5 * time.Minute))

	publishBulbStates(config, broker, &bulbs)
	go func(c <-chan time.Time) {
		for _ = range c {
			publishBulbStates(config, broker, &bulbs)
		}
	}(time.Tick(30 * time.Second))

	for _, light := range config.Lights {
		subscribe(broker, &bulbs, light, light.TopicPower, func(ctx context.Context, bulb lifx.Bulb, payload string) error {
			var power lifx.Power
			switch payload {
			case "on":
				power = lifx.On
			case "off":
				power = lifx.Off
			default:
				return fmt.Errorf("power must be one of {on, off}, found %v", payload)
			}

			if err := bulb.SetPower(ctx, power, 500*time.Millisecond); err != nil {
				return fmt.Errorf("failed to set power: %v", err)
			}
			return nil
		})
		subscribe(broker, &bulbs, light, light.TopicHue, func(ctx context.Context, bulb lifx.Bulb, payload string) error {
			hue, err := parseNumber(payload)
			if err != nil {
				return fmt.Errorf("hue must be a number, found %v", payload)
			}

			state, err := bulb.State(ctx)
			if err != nil {
				return fmt.Errorf("failed to get bulb state: %w", err)
			}

			color := state.Color
			color.Hue = hue
			if err := bulb.SetColor(ctx, color, 100*time.Millisecond); err != nil {
				return fmt.Errorf("failed to set hue: %w", err)
			}
			return nil
		})
		subscribe(broker, &bulbs, light, light.TopicSaturation, func(ctx context.Context, bulb lifx.Bulb, payload string) error {
			saturation, err := parseNumber(payload)
			if err != nil {
				return fmt.Errorf("saturation must be a number, found %v", payload)
			}

			state, err := bulb.State(ctx)
			if err != nil {
				return fmt.Errorf("failed to get bulb state: %w", err)
			}

			color := state.Color
			color.Saturation = saturation
			if err := bulb.SetColor(ctx, color, 100*time.Millisecond); err != nil {
				return fmt.Errorf("failed to set saturation: %w", err)
			}
			return nil
		})
		subscribe(broker, &bulbs, light, light.TopicBrightness, func(ctx context.Context, bulb lifx.Bulb, payload string) error {
			brightness, err := parseNumber(payload)
			if err != nil {
				return fmt.Errorf("brightness must be a number, found %v", payload)
			}

			state, err := bulb.State(ctx)
			if err != nil {
				return fmt.Errorf("failed to get bulb state: %w", err)
			}

			color := state.Color
			color.Brightness = brightness
			if err := bulb.SetColor(ctx, color, 100*time.Millisecond); err != nil {
				return fmt.Errorf("failed to set brightness: %w", err)
			}
			return nil
		})
		subscribe(broker, &bulbs, light, light.TopicKelvin, func(ctx context.Context, bulb lifx.Bulb, payload string) error {
			kelvin, err := parseNumber(payload)
			if err != nil {
				return fmt.Errorf("kelvin must be a number, found %v", payload)
			}

			state, err := bulb.State(ctx)
			if err != nil {
				return fmt.Errorf("failed to get bulb state: %w", err)
			}

			color := state.Color
			color.Kelvin = kelvin
			if err := bulb.SetColor(ctx, color, 100*time.Millisecond); err != nil {
				return fmt.Errorf("failed to set kelvin: %w", err)
			}
			return nil
		})
	}

	// block forever.
	select {}
}

func parseNumber(raw string) (int, error) {
	float, err := strconv.ParseFloat(raw, 64)
	return int(float), err
}

func publishState(broker mqtt.Client, light Light, state lifx.State) {
	broker.Publish(light.TopicPower, mqtt.AtLeastOnce, mqtt.Retain, state.Power.String())
	broker.Publish(light.TopicHue, mqtt.AtLeastOnce, mqtt.Retain, strconv.Itoa(state.Color.Hue))
	broker.Publish(light.TopicSaturation, mqtt.AtLeastOnce, mqtt.Retain, strconv.Itoa(state.Color.Saturation))
	broker.Publish(light.TopicBrightness, mqtt.AtLeastOnce, mqtt.Retain, strconv.Itoa(state.Color.Brightness))
	broker.Publish(light.TopicKelvin, mqtt.AtLeastOnce, mqtt.Retain, strconv.Itoa(state.Color.Kelvin))
}

func subscribe(broker mqtt.Client, bulbs *sync.Map, light Light, topic string, f func(context.Context, lifx.Bulb, string) error) {
	broker.Subscribe(topic, mqtt.AtLeastOnce, func(broker mqtt.Client, msg mqtt.Message) {
		payload := string(msg.Payload())

		maybeBulb, ok := bulbs.Load(light.BulbLabel)
		if !ok {
			log.Printf("%v: could not find bulb handle", light.BulbLabel)
			return
		}
		bulb := maybeBulb.(lifx.Bulb)

		ctx, _ := context.WithTimeout(context.Background(), 15*time.Second)
		if err := f(ctx, bulb, payload); err != nil {

			// If the error is invalid input, attempt to correct the bus.
			var invalidColor *lifx.ErrInvalidColor
			if errors.As(err, &invalidColor) {
				ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
				state, err := bulb.State(ctx)
				if err != nil {
					log.Printf("%v: failed to get bulb state", light.BulbLabel)
					return
				}
				publishState(broker, light, state)
				return
			}

			log.Printf("%v: %v", light.BulbLabel, err)
		}
	})
}
