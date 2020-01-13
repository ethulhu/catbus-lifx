package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/2tvenom/golifx"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	configPath = flag.String("config-path", "", "path to config.json")
)

const (
	AtMostOnce byte = iota
	AtLeastOnce
	ExactlyOnce
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

	brokerURI := fmt.Sprintf("tcp://%v:%v", config.BrokerHost, config.BrokerPort)
	log.Printf("connecting to MQTT broker %v", brokerURI)
	mqttClient := NewMQTTClient(brokerURI)

	log.Print("looking up Lifx bulbs")
	bulbs, err := golifx.LookupBulbs()
	if err != nil {
		log.Fatalf("failed to look up bulbs: %v", err)
	}
	log.Print("found bulbs")

	for _, bulb := range bulbs {
		bulb := NewBulb(bulb)
		state, err := bulb.State()
		if err != nil {
			log.Printf("o no: %v", err)
			continue
		}

		light, ok := config.LightForBulb(state.Label)
		if !ok {
			log.Printf("found unknown bulb: %v", state.Label)
			continue
		}

		power := "off"
		if state.Power {
			power = "on"
		}
		mqttClient.Publish(light.TopicPower, AtLeastOnce, true, power)
		hueDegrees := rescale(MaxUint16, 360, float64(state.Color.Hue))
		mqttClient.Publish(light.TopicHue, AtLeastOnce, true, strconv.FormatFloat(hueDegrees, 'f', -1, 64))
		saturationPercent := rescale(MaxUint16, 100, float64(state.Color.Saturation))
		mqttClient.Publish(light.TopicSaturation, AtLeastOnce, true, strconv.FormatFloat(saturationPercent, 'f', -1, 64))
		brightnessPercent := rescale(MaxUint16, 100, float64(state.Color.Brightness))
		mqttClient.Publish(light.TopicBrightness, AtLeastOnce, true, strconv.FormatFloat(brightnessPercent, 'f', -1, 64))
		mqttClient.Publish(light.TopicKelvin, AtLeastOnce, true, strconv.FormatUint(uint64(state.Color.Kelvin), 10))

		mqttClient.Subscribe(light.TopicPower, AtLeastOnce, setBulbPower(light.Name, bulb))
		mqttClient.Subscribe(light.TopicHue, AtLeastOnce, setBulbHue(light.Name, bulb))
		mqttClient.Subscribe(light.TopicSaturation, AtLeastOnce, setBulbSaturation(light.Name, bulb))
		mqttClient.Subscribe(light.TopicBrightness, AtLeastOnce, setBulbBrightness(light.Name, bulb))
		mqttClient.Subscribe(light.TopicKelvin, AtLeastOnce, setBulbKelvin(light.Name, bulb))
	}

	// TODO: have a timer periodically check for out-of-band state changes?

	// block forever.
	select {}
}

const MaxUint16 = float64(^uint16(0))

func rescale(before, after, value float64) float64 {
	return (after / before) * value
}

func setBulbPower(name string, bulb *Bulb) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		state := string(msg.Payload())
		on := false
		switch state {
		case "on":
			on = true
		case "off":
			on = false
		default:
			log.Printf("%v: unknown state: %s", name, state)
			return
		}
		err := bulb.SetPowerState(on)
		for err == io.EOF {
			log.Printf("%v: oopsie", name)
			err = bulb.SetPowerState(on)
		}
		if err != nil {
			log.Printf("%v: failed to set state: %v", name, err)
		}
	}
}
func setBulbHue(name string, bulb *Bulb) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		state := string(msg.Payload())
		hueDegrees, err := strconv.ParseFloat(state, 64)
		if err != nil || hueDegrees > 360 {
			log.Printf("%v: invalid hue value: %w", state, err)
			return
		}
		hue := rescale(360, MaxUint16, hueDegrees)
		err = bulb.SetHue(uint16(hue))
		if err != nil {
			log.Printf("%v: failed to set hue value: %w", hue, err)
			return
		}
	}
}
func setBulbSaturation(name string, bulb *Bulb) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		state := string(msg.Payload())
		saturationPercent, err := strconv.ParseFloat(state, 64)
		if err != nil || saturationPercent > 100 {
			log.Printf("%v: invalid saturation value: %w", state, err)
			return
		}
		saturation := rescale(100, MaxUint16, saturationPercent)
		err = bulb.SetSaturation(uint16(saturation))
		if err != nil {
			log.Printf("%v: failed to set saturation value: %w", saturation, err)
			return
		}
	}
}
func setBulbBrightness(name string, bulb *Bulb) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		state := string(msg.Payload())
		brightnessPercent, err := strconv.ParseFloat(state, 64)
		if err != nil || brightnessPercent > 100 {
			log.Printf("%v: invalid brightness value: %w", state, err)
			return
		}
		brightness := rescale(100, MaxUint16, brightnessPercent)
		err = bulb.SetBrightness(uint16(brightness))
		if err != nil {
			log.Printf("%v: failed to set brightness value: %w", brightness, err)
			return
		}
	}
}
func setBulbKelvin(name string, bulb *Bulb) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		state := string(msg.Payload())
		kelvin, err := strconv.ParseUint(state, 10, 16)
		if err != nil {
			log.Printf("%v: invalid kelvin value: %w", state, err)
			return
		}
		err = bulb.SetKelvin(uint16(kelvin))
		if err != nil {
			log.Printf("%v: failed to set kelvin value: %w", kelvin, err)
			return
		}
	}
}

func NewMQTTClient(brokerURI string) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURI)
	opts.SetAutoReconnect(true)
	opts.SetOnConnectHandler(func(_ mqtt.Client) {
		log.Printf("connected to MQTT broker %s", brokerURI)
	})
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		log.Printf("disconnected from MQTT broker %s: %v", brokerURI, err)
	})

	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	return client
}
