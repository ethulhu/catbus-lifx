package main

import (
	"flag"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/ethulhu/mqtt-lifx-bridge/lifx"
	"github.com/ethulhu/mqtt-lifx-bridge/mqtt"
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

	log.Printf("connecting to MQTT broker %v:%v", config.BrokerHost, config.BrokerPort)
	broker := mqtt.NewClient(config.BrokerHost, config.BrokerPort)

	log.Print("looking up Lifx bulbs")
	bulbs := lifx.NewCollection()
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case _ = <-ticker.C:
				if err := bulbs.Refresh(); err != nil {
					log.Printf("failed to look up bulbs: %v", err)
				}

				for _, light := range config.Lights {
					bulb, ok := bulbs.Bulb(light.BulbLabel)
					if !ok {
						log.Printf("%v: failed to find bulb", light.Name)
						continue
					}
					state, err := bulb.State()
					if err != nil {
						log.Printf("%v: failed to get state: %v", light.Name, err)
						continue
					}

					power := "off"
					if state.Power {
						power = "on"
					}
					broker.Publish(light.TopicPower, mqtt.AtLeastOnce, true, power)
					hueDegrees := rescale(MaxUint16, 360, float64(state.Color.Hue))
					broker.Publish(light.TopicHue, mqtt.AtLeastOnce, true, strconv.FormatFloat(hueDegrees, 'f', -1, 64))
					saturationPercent := rescale(MaxUint16, 100, float64(state.Color.Saturation))
					broker.Publish(light.TopicSaturation, mqtt.AtLeastOnce, true, strconv.FormatFloat(saturationPercent, 'f', -1, 64))
					brightnessPercent := rescale(MaxUint16, 100, float64(state.Color.Brightness))
					broker.Publish(light.TopicBrightness, mqtt.AtLeastOnce, true, strconv.FormatFloat(brightnessPercent, 'f', -1, 64))
					broker.Publish(light.TopicKelvin, mqtt.AtLeastOnce, true, strconv.FormatUint(uint64(state.Color.Kelvin), 10))
				}
			}
		}
	}()

	for _, light := range config.Lights {
		broker.Subscribe(light.TopicPower, mqtt.AtLeastOnce, setBulbPower(bulbs, light.Name, light.BulbLabel))
		broker.Subscribe(light.TopicHue, mqtt.AtLeastOnce, setBulbHue(bulbs, light.Name, light.BulbLabel))
		broker.Subscribe(light.TopicSaturation, mqtt.AtLeastOnce, setBulbSaturation(bulbs, light.Name, light.BulbLabel))
		broker.Subscribe(light.TopicBrightness, mqtt.AtLeastOnce, setBulbBrightness(bulbs, light.Name, light.BulbLabel))
		broker.Subscribe(light.TopicKelvin, mqtt.AtLeastOnce, setBulbKelvin(bulbs, light.Name, light.BulbLabel))
	}

	// TODO: have a timer periodically check for out-of-band state changes?

	// block forever.
	select {}
}

const MaxUint16 = float64(^uint16(0))

func rescale(before, after, value float64) float64 {
	return (after / before) * value
}

func setBulbPower(bulbs *lifx.Collection, name, label string) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		state := string(msg.Payload())
		on := false
		switch state {
		case "on":
			on = true
		case "off":
			on = false
		default:
			log.Printf("%v: invalid power value %q", name, state)
			return
		}

		bulb, ok := bulbs.Bulb(label)
		if !ok {
			log.Printf("%v: could not find bulb", name)
			return
		}

		err := bulb.SetPowerState(on)
		for err == io.EOF {
			log.Printf("%v: retrying to set power", name)
			err = bulb.SetPowerState(on)
		}
		if err != nil {
			log.Printf("%v: failed to set state: %v", name, err)
		}
	}
}
func setBulbHue(bulbs *lifx.Collection, name, label string) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		state := string(msg.Payload())
		hueDegrees, err := strconv.ParseFloat(state, 64)
		if err != nil || hueDegrees > 360 {
			log.Printf("%v: invalid hue value %q", name, state)
			return
		}

		bulb, ok := bulbs.Bulb(label)
		if !ok {
			log.Printf("%v: could not find bulb", name)
			return
		}

		hue := rescale(360, MaxUint16, hueDegrees)
		err = bulb.SetHue(uint16(hue))
		for err == io.EOF {
			log.Printf("%v: retrying to set hue", name)
			err = bulb.SetHue(uint16(hue))
		}
		if err != nil {
			log.Printf("%v: failed to set hue value: %v", name, err)
			return
		}
	}
}
func setBulbSaturation(bulbs *lifx.Collection, name, label string) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		state := string(msg.Payload())
		saturationPercent, err := strconv.ParseFloat(state, 64)
		if err != nil || saturationPercent > 100 {
			log.Printf("%v: invalid saturation value %q", name, state)
			return
		}

		bulb, ok := bulbs.Bulb(label)
		if !ok {
			log.Printf("%v: could not find bulb", name)
			return
		}

		saturation := rescale(100, MaxUint16, saturationPercent)
		err = bulb.SetSaturation(uint16(saturation))
		for err == io.EOF {
			log.Printf("%v: retrying to set saturation", name)
			err = bulb.SetSaturation(uint16(saturation))
		}
		if err != nil {
			log.Printf("%v: failed to set saturation value: %v", name, err)
			return
		}
	}
}
func setBulbBrightness(bulbs *lifx.Collection, name, label string) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		state := string(msg.Payload())
		brightnessPercent, err := strconv.ParseFloat(state, 64)
		if err != nil || brightnessPercent > 100 {
			log.Printf("%v: invalid brightness value %q", name, state)
			return
		}

		bulb, ok := bulbs.Bulb(label)
		if !ok {
			log.Printf("%v: could not find bulb", name)
			return
		}

		brightness := rescale(100, MaxUint16, brightnessPercent)
		err = bulb.SetBrightness(uint16(brightness))
		for err == io.EOF {
			log.Printf("%v: retrying to set saturation", name)
			err = bulb.SetBrightness(uint16(brightness))
		}
		if err != nil {
			log.Printf("%v: failed to set brightness value: %v", name, err)
			return
		}
	}
}
func setBulbKelvin(bulbs *lifx.Collection, name, label string) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		state := string(msg.Payload())
		kelvin, err := strconv.ParseUint(state, 10, 16)
		if err != nil {
			log.Printf("%v: invalid kelvin value %q", name, state)
			return
		}

		bulb, ok := bulbs.Bulb(label)
		if !ok {
			log.Printf("%v: could not find bulb", name)
			return
		}

		err = bulb.SetKelvin(uint16(kelvin))
		for err == io.EOF {
			log.Printf("%v: retrying to set saturation", name)
			err = bulb.SetKelvin(uint16(kelvin))
		}
		if err != nil {
			log.Printf("%v: failed to set kelvin value: %v", name, err)
			return
		}
	}
}
