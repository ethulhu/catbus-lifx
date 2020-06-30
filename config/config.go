// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package config

import (
	"encoding/json"
	"io/ioutil"
)

type (
	Bulb struct {
		Label  string
		Topics struct {
			Power      string
			Hue        string
			Saturation string
			Brightness string
			Kelvin     string
		}
	}

	Config struct {
		BrokerURI string

		BulbsByLabel map[string]Bulb
	}

	config struct {
		MQTTBroker string `json:"mqttBroker"`
		Bulbs      map[string]struct {
			Label  string `json:"label"`
			Topics struct {
				Power      string `json:"power"`
				Hue        string `json:"hue"`
				Saturation string `json:"saturation"`
				Brightness string `json:"brightness"`
				Kelvin     string `json:"kelvin"`
			} `json:"topics"`
		} `json:"bulbs"`
	}
)

func ParseFile(path string) (*Config, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	raw := config{}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return nil, err
	}

	return configFromConfig(raw), nil
}

func configFromConfig(raw config) *Config {
	c := &Config{
		BrokerURI:    raw.MQTTBroker,
		BulbsByLabel: map[string]Bulb{},
	}

	for k, v := range raw.Bulbs {
		label := k
		if v.Label != "" {
			label = v.Label
		}

		b := Bulb(v)
		b.Label = label

		c.BulbsByLabel[label] = b
	}

	return c
}
