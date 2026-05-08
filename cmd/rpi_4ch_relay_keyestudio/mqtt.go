// Copyright (C) 2026 Orbit OS <orbit-os.org>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/OrbitOS-org/sdk-go/v26/logger"
)

type mqttManager struct {
	client mqtt.Client
	cfg    mqttConfig
	ctrl   *controller
}

func newMQTTManager(cfg mqttConfig, ctrl *controller) *mqttManager {
	return &mqttManager{cfg: cfg, ctrl: ctrl}
}

func (m *mqttManager) deviceID() string {
	if m.cfg.DeviceID != "" {
		return m.cfg.DeviceID
	}
	h, _ := os.Hostname()
	if h == "" {
		h = "relay4"
	}
	return h
}

func (m *mqttManager) prefix() string {
	if m.cfg.TopicPrefix != "" {
		return m.cfg.TopicPrefix
	}
	return "relay4"
}

func (m *mqttManager) stateTopic(ch int) string {
	return fmt.Sprintf("%s/%s/%d/state", m.prefix(), m.deviceID(), ch+1)
}

func (m *mqttManager) commandTopic(ch int) string {
	return fmt.Sprintf("%s/%s/%d/set", m.prefix(), m.deviceID(), ch+1)
}

func (m *mqttManager) availabilityTopic() string {
	return fmt.Sprintf("%s/%s/availability", m.prefix(), m.deviceID())
}

func (m *mqttManager) discoveryTopic(ch int) string {
	return fmt.Sprintf("homeassistant/switch/%s/relay4_%d/config", m.deviceID(), ch+1)
}

func (m *mqttManager) start() {
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://" + m.cfg.Broker)
	opts.SetClientID("relay4-" + m.deviceID())
	if m.cfg.Username != "" {
		opts.SetUsername(m.cfg.Username)
		opts.SetPassword(m.cfg.Password)
	}
	opts.SetWill(m.availabilityTopic(), "offline", 1, true)
	opts.SetAutoReconnect(true)
	opts.SetOnConnectHandler(m.onConnect)
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		logger.Warnf(logTag, "MQTT connection lost: %v", err)
	})

	m.client = mqtt.NewClient(opts)
	go m.connect()
}

func (m *mqttManager) connect() {
	for {
		token := m.client.Connect()
		token.Wait()
		if err := token.Error(); err != nil {
			logger.Warnf(logTag, "MQTT connect %s: %v — retrying in 10s", m.cfg.Broker, err)
			time.Sleep(10 * time.Second)
			continue
		}
		return
	}
}

func (m *mqttManager) onConnect(c mqtt.Client) {
	logger.Infof(logTag, "MQTT connected to %s", m.cfg.Broker)

	c.Publish(m.availabilityTopic(), 1, true, "online")

	for i := 0; i < 4; i++ {
		m.publishDiscovery(i)
	}

	st := m.ctrl.getAll()
	for i := 0; i < 4; i++ {
		val := "OFF"
		if st[i] {
			val = "ON"
		}
		c.Publish(m.stateTopic(i), 1, true, val)
	}

	for i := 0; i < 4; i++ {
		ch := i
		c.Subscribe(m.commandTopic(ch), 1, func(_ mqtt.Client, msg mqtt.Message) {
			on := string(msg.Payload()) == "ON"
			if err := m.ctrl.set(ch, on); err != nil {
				logger.Errorf(logTag, "MQTT set CH%d: %v", ch+1, err)
			}
		})
	}

	logger.Infof(logTag, "MQTT: discovery published, subscribed to command topics")
}

func (m *mqttManager) publishDiscovery(ch int) {
	devID := m.deviceID()
	prefix := m.prefix()

	payload := map[string]any{
		"name":                  fmt.Sprintf("Relay %d", ch+1),
		"unique_id":             fmt.Sprintf("%s_relay4_%d", devID, ch+1),
		"command_topic":         fmt.Sprintf("%s/%s/%d/set", prefix, devID, ch+1),
		"state_topic":           fmt.Sprintf("%s/%s/%d/state", prefix, devID, ch+1),
		"payload_on":            "ON",
		"payload_off":           "OFF",
		"retain":                true,
		"availability_topic":    m.availabilityTopic(),
		"payload_available":     "online",
		"payload_not_available": "offline",
		"device": map[string]any{
			"identifiers":  []string{devID + "_relay4"},
			"name":         "KS0212 4-Channel Relay",
			"model":        "KS0212",
			"manufacturer": "Keyestudio",
		},
	}

	data, _ := json.Marshal(payload)
	m.client.Publish(m.discoveryTopic(ch), 1, true, string(data))
}

func (m *mqttManager) publishState(ch int, on bool) {
	if m.client == nil || !m.client.IsConnected() {
		return
	}
	val := "OFF"
	if on {
		val = "ON"
	}
	m.client.Publish(m.stateTopic(ch), 1, true, val)
}

func (m *mqttManager) stop() {
	if m.client != nil && m.client.IsConnected() {
		m.client.Publish(m.availabilityTopic(), 1, true, "offline")
		m.client.Disconnect(500)
		logger.Infof(logTag, "MQTT disconnected")
	}
}
