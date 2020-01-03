package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"time"

	"github.com/2tvenom/golifx"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	configPath = flag.String("config-path", "", "path to config.json")
)

type (
	Light struct {
		Name       string `json:"name"`
		BulbLabel  string `json:"bulb_label"`
		TopicPower string `json:"topic_power"`
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
		bulb := bulb

		state, err := bulb.GetColorState()
		if err != nil {
			log.Printf("failed to look up bulb: %v", err)
		}

		light, ok := config.LightForBulb(state.Label)
		if !ok {
			log.Printf("found unknown bulb: %v", state.Label)
			continue
		}

		mqttClient.Subscribe(light.TopicPower, 1, setBulbPower(light.Name, bulb))
	}

	for {
	}
}

func setBulbPower(name string, bulb *golifx.Bulb) mqtt.MessageHandler {
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
