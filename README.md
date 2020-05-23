<!--
SPDX-FileCopyrightText: 2020 Ethel Morgan

SPDX-License-Identifier: MIT
-->

# Catbus Lifx

Control [Lifx](https://www.lifx.com/) bulbs using [Catbus](https://ethulhu.co.uk/catbus), a home automation framework built around MQTT.

## MQTT Topics

The control of each parameter of the bulb is split into its own topic:

- power, either `on` or `off`.
- hue, in degrees, from 0 to 359.
- saturation, as a percentage, from 0 to 100.
- brightness, as a percentage, from 0 to 100.
- kelvin, the color temperature, from 2500 to 9000.

## Configuration

The bridge is configured with a JSON file, containing:

- the broker host & port.
- one or more lights, where a light defines:
 - its Lifx bulb label.
 - its topics for each of power, hue, saturation, brightness, and kelvin.

For example,

```json
{
	"broker_host": "home-server.local",
	"broker_port": 1883,
	"lights": [
		{
			"bulb_label":       "Bedside Lamp",

			"topic_power":      "home/bedroom/bedside/power",
			"topic_hue":        "home/bedroom/bedside/hue_degrees",
			"topic_saturation": "home/bedroom/bedside/saturation_percent",
			"topic_brightness": "home/bedroom/bedside/saturation_percent",
			"topic_kelvin":     "home/bedroom/bedside/kelvin"
		}
	]
}
```
