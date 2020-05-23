// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type (
	Light struct {
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