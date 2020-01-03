package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"github.com/2tvenom/golifx"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	configPath = flag.String("config-path", "", "path to config.json")
)

type (
	Light struct {
		Name            string `json:"name"`
		BulbLabel       string `json:"bulb_label"`
		TopicPower      string `json:"topic_power"`
		TopicKelvin     string `json:"topic_kelvin"`
		TopicSaturation string `json:"topic_saturation"`
		TopicBrightness string `json:"topic_brightness"`
		TopicHue        string `json:"topic_hue"`
	}
	Config struct {
		BrokerHost string `json:"broker_host"`
		BrokerPort uint   `json:"broker_port"`

		Lights []Light `json:"lights"`
	}
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

		mqttClient.Subscribe(light.TopicPower, 1, setBulbPower(light.Name, bulb))
		mqttClient.Subscribe(light.TopicHue, 1, setBulbHue(light.Name, bulb))
		mqttClient.Subscribe(light.TopicSaturation, 1, setBulbSaturation(light.Name, bulb))
		mqttClient.Subscribe(light.TopicBrightness, 1, setBulbBrightness(light.Name, bulb))
		mqttClient.Subscribe(light.TopicKelvin, 1, setBulbKelvin(light.Name, bulb))
	}

	for {
	}
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
		hueDegrees, err := strconv.ParseUint(state, 10, 16)
		if err != nil || hueDegrees > 360 {
			log.Printf("%v: invalid hue value: %w", err)
			return
		}
		hue := hueDegrees * 182
		err = bulb.SetHue(uint16(hue))
		if err != nil {
			log.Printf("%v: failed to set hue value: %w", err)
			return
		}
	}
}
func setBulbSaturation(name string, bulb *Bulb) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		state := string(msg.Payload())
		saturationPercent, err := strconv.ParseUint(state, 10, 8)
		if err != nil || saturationPercent > 100 {
			log.Printf("%v: invalid saturation value: %w", err)
			return
		}
		saturation := saturationPercent * 655
		err = bulb.SetSaturation(uint16(saturation))
		if err != nil {
			log.Printf("%v: failed to set saturation value: %w", err)
			return
		}
	}
}
func setBulbBrightness(name string, bulb *Bulb) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		state := string(msg.Payload())
		brightnessPercent, err := strconv.ParseUint(state, 10, 8)
		if err != nil || brightnessPercent > 100 {
			log.Printf("%v: invalid brightness value: %w", err)
			return
		}
		brightness := brightnessPercent * 655
		err = bulb.SetBrightness(uint16(brightness))
		if err != nil {
			log.Printf("%v: failed to set brightness value: %w", err)
			return
		}
	}
}
func setBulbKelvin(name string, bulb *Bulb) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		state := string(msg.Payload())
		k, err := strconv.ParseUint(state, 10, 16)
		if err != nil {
			log.Printf("%v: invalid kelvin value: %w", err)
			return
		}
		err = bulb.SetKelvin(uint16(k))
		if err != nil {
			log.Printf("%v: failed to set kelvin value: %w", err)
			return
		}
	}
}

func NewMQTTClient(brokerURI string) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURI)
	opts.SetConnectRetry(true)
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

func (c *Config) LightForBulb(label string) (Light, bool) {
	for _, light := range c.Lights {
		if light.BulbLabel == label {
			return light, true
		}
	}
	return Light{}, false
}

func loadConfig(path string) (*Config, error) {
	src, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	config := &Config{}
	if err := json.Unmarshal(src, config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return config, nil
}
