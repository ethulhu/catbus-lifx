# MQTT-Lifx Bridge

Control [Lifx](https://www.lifx.com/) bulbs using MQTT.

## MQTT Topics

The control of each parameter of the bulb is split into its own topic:

- power, either `on` or `off`.
- hue, in degrees between 0 and 360.
- saturation, as a percentage.
- brightness, as a percentage.
- kelvin, the color temperature.

## Configuration

The bridge is configured with a JSON file, containing:

- the broker host & port.
- one or more lights, where a light defines:
 - its name.
 - its Lifx bulb label.
 - its topics for each of power, hue, saturation, brightness, and kelvin.

For example,

```json
{
	"broker_host": "home-server.local",
	"broker_port": 1883,
	"lights": [
		"name":             "bedroom bedside lamp",
		"bulb_label":       "Bedside Lamp",

		"topic_power":      "home/bedroom/bedside/power",
		"topic_hue":        "home/bedroom/bedside/hue_degrees",
		"topic_saturation": "home/bedroom/bedside/saturation_percent",
		"topic_brightness": "home/bedroom/bedside/saturation_percent",
		"topic_kelvin":     "home/bedroom/bedside/kelvin"
	]
}
```
