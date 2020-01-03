# MQTT-Lifx Bridge

Control [Lifx](https://www.lifx.com/) bulbs using MQTT.

## Topics

The control of each parameter of the bulb is split into its own topic:

- power, either `on` or `off`.
- hue, in degrees between 0 and 360.
- saturation, as a percentage.
- brightness, as a percentage.
- kelvin, the color temperature.
