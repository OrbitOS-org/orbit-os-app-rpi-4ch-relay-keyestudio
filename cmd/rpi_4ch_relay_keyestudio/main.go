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

// relay4 — Modbus TCP server + web UI for the Keyestudio KS0212
// 4-channel relay shield on Raspberry Pi.
//
// GPIO mapping (BCM, gpiochip0):
//
//	CH1 → GPIO4   CH2 → GPIO22   CH3 → GPIO6   CH4 → GPIO26
//
// Modbus TCP coils (FC01/FC05/FC0F):
//
//	coil 0 → CH1   coil 1 → CH2   coil 2 → CH3   coil 3 → CH4
package main

import (
	"context"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	orbitclient "github.com/OrbitOS-org/sdk-go/v26/client"
	"github.com/OrbitOS-org/sdk-go/v26/logger"
	"github.com/OrbitOS-org/sdk-go/v26/metadata"
)

//go:embed metadata.json
var metadataJSON []byte

//go:embed webui/index.html
var webUI []byte

//go:embed webui/favicon.svg
var faviconSVG []byte

//go:embed img/shield.png
var shieldPNG []byte

var appManifest = metadata.MustParseAppManifestJSON(metadataJSON)

const (
	logTag            = "relay4"
	httpAddr          = "127.0.0.1:9012"
	defaultGravity    = "192.168.1.51"
	defaultModbusPort = 5020
	configFile        = "relay4_config.json"
)

type mqttConfig struct {
	Enabled     bool   `json:"enabled"`
	Broker      string `json:"broker"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	TopicPrefix string `json:"topic_prefix"`
	DeviceID    string `json:"device_id"`
}

type appConfig struct {
	ModbusEnabled bool       `json:"modbus_enabled"`
	ModbusPort    int        `json:"modbus_port"`
	MQTT          mqttConfig `json:"mqtt"`
}

func defaultConfig() appConfig {
	return appConfig{
		ModbusEnabled: true,
		ModbusPort:    defaultModbusPort,
		MQTT:          mqttConfig{TopicPrefix: "relay4"},
	}
}

func loadConfig() appConfig {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return defaultConfig()
	}
	cfg := defaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil || cfg.ModbusPort <= 0 {
		return defaultConfig()
	}
	if cfg.MQTT.TopicPrefix == "" {
		cfg.MQTT.TopicPrefix = "relay4"
	}
	return cfg
}

func saveConfig(cfg appConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

// relay defines a single channel: label, GPIO pin number (BCM), chip index.
type relay struct {
	label string
	gpio  int32
	chip  int32
}

var relays = [4]relay{
	{label: "Channel 1", gpio: 4, chip: 0},
	{label: "Channel 2", gpio: 22, chip: 0},
	{label: "Channel 3", gpio: 6, chip: 0},
	{label: "Channel 4", gpio: 26, chip: 0},
}

// controller holds the SDK client and relay state.
type controller struct {
	mu           sync.RWMutex
	state        [4]bool
	cfg          appConfig
	gpio         *orbitclient.GpioManager
	boardWarn    string
	appCtx       context.Context
	modbusCancel context.CancelFunc
	sseClients   map[chan [4]bool]struct{}
	mqttMgr      *mqttManager
}

func newController(gpio *orbitclient.GpioManager, boardWarn string, cfg appConfig, appCtx context.Context) *controller {
	return &controller{
		gpio:       gpio,
		boardWarn:  boardWarn,
		cfg:        cfg,
		appCtx:     appCtx,
		sseClients: make(map[chan [4]bool]struct{}),
	}
}

func (c *controller) notifySSE(s [4]bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for ch := range c.sseClients {
		select {
		case ch <- s:
		default:
		}
	}
}

func (c *controller) modbusAddr() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return fmt.Sprintf("0.0.0.0:%d", c.cfg.ModbusPort)
}

func (c *controller) startModbus() {
	c.mu.Lock()
	if c.modbusCancel != nil {
		c.modbusCancel()
	}
	ctx, cancel := context.WithCancel(c.appCtx)
	c.modbusCancel = cancel
	c.mu.Unlock()
	go c.runModbus(ctx)
}

func (c *controller) stopModbus() {
	c.mu.Lock()
	if c.modbusCancel != nil {
		c.modbusCancel()
		c.modbusCancel = nil
	}
	c.mu.Unlock()
	logger.Infof(logTag, "Modbus TCP stopped")
}

// set drives a relay ON (true) or OFF (false).
func (c *controller) set(ch int, on bool) error {
	r := relays[ch]
	pin := &orbitclient.GpioPin{Number: r.gpio, ChipNumber: r.chip}

	if err := c.gpio.SetDirection(pin, orbitclient.GPIO_DIR_OUT); err != nil {
		logger.Warnf(logTag, "SetDirection CH%d GPIO%d: %v", ch+1, r.gpio, err)
	}
	lvl := orbitclient.GPIO_LEVEL_LOW
	if on {
		lvl = orbitclient.GPIO_LEVEL_HIGH
	}
	if err := c.gpio.SetLevel(pin, lvl); err != nil {
		return err
	}
	c.mu.Lock()
	c.state[ch] = on
	snap := c.state
	c.mu.Unlock()
	logger.Infof(logTag, "CH%d GPIO%d → %v", ch+1, r.gpio, on)
	go c.notifySSE(snap)
	if c.mqttMgr != nil {
		c.mqttMgr.publishState(ch, on)
	}
	return nil
}

// getAll returns current state snapshot.
func (c *controller) getAll() [4]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// ── board check ───────────────────────────────────────────────────────────────

// rpiModelPrefixes matches known Raspberry Pi board model strings
// returned by Gravity (e.g. "4-model-b", "raspberry pi 4", "rpi4", "3-model-b-plus").
var rpiModelPrefixes = []string{
	"5-model", "4-model", "3-model", "zero",
}

func isRaspberryPi(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	for _, p := range rpiModelPrefixes {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func checkRaspberryPi(sys *orbitclient.SystemManager) string {
	if model, err := sys.GetBoardModel(); err == nil {
		if isRaspberryPi(model) {
			logger.Infof(logTag, "board: %s — OK", model)
			return ""
		}
		msg := "board model «" + model + "» does not look like a Raspberry Pi — GPIO pins may differ"
		logger.Warnf(logTag, "%s", msg)
		return msg
	}
	if vendor, err := sys.GetBoardVendor(); err == nil && isRaspberryPi(vendor) {
		return ""
	}
	msg := "could not confirm Raspberry Pi board — GPIO pins may differ"
	logger.Warnf(logTag, "%s", msg)
	return msg
}

// ── HTTP handlers ─────────────────────────────────────────────────────────────

func (c *controller) registerHTTP(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(webUI)
	})

	mux.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(faviconSVG)
	})

	mux.HandleFunc("/img/shield.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(shieldPNG)
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c.mu.RLock()
		modbusActive := c.modbusCancel != nil
		mqttActive := c.mqttMgr != nil && c.mqttMgr.client != nil && c.mqttMgr.client.IsConnected()
		c.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"modbus": modbusActive,
			"mqtt":   mqttActive,
		})
	})

	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"modbus_addr": c.modbusAddr(),
			"board_warn":  c.boardWarn,
		})
	})

	mux.HandleFunc("/api/relays", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		st := c.getAll()
		bools := [4]bool(st)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"relays": bools})
	})

	// POST /api/relays/{0-3}  body: {"on": true}
	mux.HandleFunc("/api/relays/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		seg := strings.TrimPrefix(r.URL.Path, "/api/relays/")
		ch, err := strconv.Atoi(strings.Trim(seg, "/"))
		if err != nil || ch < 0 || ch > 3 {
			http.Error(w, "channel must be 0-3", http.StatusBadRequest)
			return
		}
		var body struct {
			On bool `json:"on"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := c.set(ch, body.On); err != nil {
			logger.Errorf(logTag, "set CH%d: %v", ch+1, err)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch := make(chan [4]bool, 4)
		c.mu.Lock()
		c.sseClients[ch] = struct{}{}
		c.mu.Unlock()
		defer func() {
			c.mu.Lock()
			delete(c.sseClients, ch)
			c.mu.Unlock()
		}()

		// send current state immediately
		st := c.getAll()
		data, _ := json.Marshal(map[string]any{"relays": st})
		fmt.Fprintf(w, "data: %s\n\n", data)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		for {
			select {
			case st := <-ch:
				data, _ := json.Marshal(map[string]any{"relays": st})
				fmt.Fprintf(w, "data: %s\n\n", data)
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			case <-r.Context().Done():
				return
			}
		}
	})

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			c.mu.RLock()
			cfg := c.cfg
			c.mu.RUnlock()
			_ = json.NewEncoder(w).Encode(cfg)
		case http.MethodPost:
			var body appConfig
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid JSON", http.StatusBadRequest)
				return
			}
			if body.ModbusPort < 1 || body.ModbusPort > 65535 {
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "port must be 1-65535"})
				return
			}
			c.mu.Lock()
			c.cfg = body
			c.mu.Unlock()
			if err := saveConfig(body); err != nil {
				logger.Errorf(logTag, "saveConfig: %v", err)
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
				return
			}
			logger.Infof(logTag, "config saved: modbus_enabled=%v port=%d mqtt_enabled=%v", body.ModbusEnabled, body.ModbusPort, body.MQTT.Enabled)
			// Modbus
			if body.ModbusEnabled {
				c.startModbus()
			} else {
				c.stopModbus()
			}
			// MQTT
			if c.mqttMgr != nil {
				c.mqttMgr.stop()
			}
			if body.MQTT.Enabled {
				c.mqttMgr = newMQTTManager(body.MQTT, c)
				c.mqttMgr.start()
			} else {
				c.mqttMgr = nil
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// ── Modbus TCP server ─────────────────────────────────────────────────────────
//
// Implements FC01 (Read Coils), FC05 (Write Single Coil), FC0F (Write Multiple Coils).
// Coil address = channel index (0-3).

const (
	mbFCReadCoils            = 0x01
	mbFCReadHoldingRegisters = 0x03
	mbFCWriteSingleCoil      = 0x05
	mbFCWriteSingleRegister  = 0x06
	mbFCWriteMultipleCoils   = 0x0F
	mbExcIllegalFunction     = 0x01
	mbExcIllegalDataAddr     = 0x02
	mbExcIllegalDataValue    = 0x03
)

func (c *controller) runModbus(ctx context.Context) {
	addr := c.modbusAddr()
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Errorf(logTag, "Modbus TCP listen %s: %v", addr, err)
		return
	}
	logger.Infof(logTag, "Modbus TCP listening on %s", addr)
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
			default:
				logger.Errorf(logTag, "Modbus accept: %v", err)
			}
			return
		}
		go c.handleModbusConn(conn)
	}
}

func (c *controller) handleModbusConn(conn net.Conn) {
	remote := conn.RemoteAddr().String()
	logger.Infof(logTag, "Modbus TCP: new connection from %s", remote)
	defer func() {
		logger.Infof(logTag, "Modbus TCP: connection closed %s", remote)
		conn.Close()
	}()
	buf := make([]byte, 256)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		// MBAP header: 6 bytes + PDU
		if _, err := readFull(conn, buf[:6]); err != nil {
			logger.Warnf(logTag, "Modbus [%s] MBAP read: %v", remote, err)
			return
		}
		txID := binary.BigEndian.Uint16(buf[0:2])
		protoID := binary.BigEndian.Uint16(buf[2:4])
		length := binary.BigEndian.Uint16(buf[4:6])
		logger.Infof(logTag, "Modbus [%s] MBAP txID=%d proto=%d length=%d", remote, txID, protoID, length)
		if length < 2 || length > 250 {
			logger.Warnf(logTag, "Modbus [%s] invalid length %d — closing", remote, length)
			return
		}
		if _, err := readFull(conn, buf[6:6+length]); err != nil {
			logger.Warnf(logTag, "Modbus [%s] PDU read: %v", remote, err)
			return
		}
		unitID := buf[6]
		fc := buf[7]
		pdu := buf[8 : 6+length]
		logger.Infof(logTag, "Modbus [%s] unitID=%d FC=0x%02X pdu=%X", remote, unitID, fc, pdu)

		resp := c.processPDU(fc, pdu)
		logger.Infof(logTag, "Modbus [%s] response=%X", remote, resp)

		// build response MBAP
		out := make([]byte, 6+1+len(resp)) // MBAP + unit_id + PDU
		binary.BigEndian.PutUint16(out[0:], txID)
		binary.BigEndian.PutUint16(out[2:], 0x0000) // protocol
		binary.BigEndian.PutUint16(out[4:], uint16(1+len(resp)))
		out[6] = unitID
		copy(out[7:], resp)
		_, _ = conn.Write(out)
	}
}

func (c *controller) processPDU(fc byte, data []byte) []byte {
	excResp := func(code byte) []byte { return []byte{fc | 0x80, code} }

	switch fc {
	case mbFCReadCoils:
		if len(data) < 4 {
			return excResp(mbExcIllegalDataValue)
		}
		addr := binary.BigEndian.Uint16(data[0:2])
		count := binary.BigEndian.Uint16(data[2:4])
		if count == 0 || count > 2000 {
			return excResp(mbExcIllegalDataAddr)
		}
		// Accept any addr/count — coils beyond index 3 are always 0.
		byteCount := (count + 7) / 8
		resp := make([]byte, 2+byteCount)
		resp[0] = fc
		resp[1] = byte(byteCount)
		st := c.getAll()
		for i := uint16(0); i < count; i++ {
			if addr+i < 4 && st[addr+i] {
				resp[2+i/8] |= 1 << (i % 8)
			}
		}
		return resp

	case mbFCReadHoldingRegisters:
		if len(data) < 4 {
			return excResp(mbExcIllegalDataValue)
		}
		addr := binary.BigEndian.Uint16(data[0:2])
		count := binary.BigEndian.Uint16(data[2:4])
		if count == 0 || count > 125 {
			return excResp(mbExcIllegalDataAddr)
		}
		// Each register = 2 bytes; relay ON → 0xFF00, OFF → 0x0000.
		byteCount := count * 2
		resp := make([]byte, 2+byteCount)
		resp[0] = fc
		resp[1] = byte(byteCount)
		st := c.getAll()
		for i := uint16(0); i < count; i++ {
			if addr+i < 4 && st[addr+i] {
				resp[2+i*2] = 0xFF
				resp[3+i*2] = 0x00
			}
		}
		return resp

	case mbFCWriteSingleRegister:
		if len(data) < 4 {
			return excResp(mbExcIllegalDataValue)
		}
		addr := binary.BigEndian.Uint16(data[0:2])
		val := binary.BigEndian.Uint16(data[2:4])
		if addr > 3 {
			// out-of-range write — echo and ignore
			return []byte{fc, data[0], data[1], data[2], data[3]}
		}
		on := val != 0x0000
		if err := c.set(int(addr), on); err != nil {
			return excResp(mbExcIllegalDataValue)
		}
		return []byte{fc, data[0], data[1], data[2], data[3]}

	case mbFCWriteSingleCoil:
		if len(data) < 4 {
			return excResp(mbExcIllegalDataValue)
		}
		addr := binary.BigEndian.Uint16(data[0:2])
		val := binary.BigEndian.Uint16(data[2:4])
		if addr > 3 {
			return excResp(mbExcIllegalDataAddr)
		}
		if val != 0xFF00 && val != 0x0000 {
			return excResp(mbExcIllegalDataValue)
		}
		on := val == 0xFF00
		if err := c.set(int(addr), on); err != nil {
			return excResp(mbExcIllegalDataValue)
		}
		// echo request
		return []byte{fc, data[0], data[1], data[2], data[3]}

	case mbFCWriteMultipleCoils:
		if len(data) < 5 {
			return excResp(mbExcIllegalDataValue)
		}
		addr := binary.BigEndian.Uint16(data[0:2])
		count := binary.BigEndian.Uint16(data[2:4])
		if addr > 3 || count == 0 || count > 2000 {
			return excResp(mbExcIllegalDataAddr)
		}
		byteCount := (count + 7) / 8
		if len(data) < 5+int(byteCount) {
			return excResp(mbExcIllegalDataValue)
		}
		coilBytes := data[5:]
		for i := uint16(0); i < count; i++ {
			if addr+i > 3 {
				break
			}
			on := (coilBytes[i/8]>>(i%8))&1 == 1
			if err := c.set(int(addr+i), on); err != nil {
				return excResp(mbExcIllegalDataValue)
			}
		}
		return []byte{fc, data[0], data[1], data[2], data[3]}

	default:
		return excResp(mbExcIllegalFunction)
	}
}

func readFull(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := conn.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	meta := metadata.Build(appManifest)
	logger.Init(meta.Name, "INFO", true)
	logger.Infof(logTag, "Starting %s", meta.Name)
	appManifest.PrintInfo()

	client, err := orbitclient.NewClientAuto(defaultGravity)
	if err != nil {
		logger.Fatalf(logTag, "Gravity (SDK): %v", err)
		os.Exit(1)
	}
	defer client.Close()

	boardWarn := checkRaspberryPi(client.SystemManager)

	cfg := loadConfig()
	logger.Infof(logTag, "Modbus port: %d (config: %s)", cfg.ModbusPort, configFile)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	ctrl := newController(client.GpioManager, boardWarn, cfg, ctx)

	if cfg.MQTT.Enabled {
		ctrl.mqttMgr = newMQTTManager(cfg.MQTT, ctrl)
		ctrl.mqttMgr.start()
	}

	// Modbus TCP
	if cfg.ModbusEnabled {
		ctrl.startModbus()
	} else {
		logger.Infof(logTag, "Modbus TCP disabled by config")
	}

	// HTTP
	mux := http.NewServeMux()
	ctrl.registerHTTP(mux)

	srv := &http.Server{
		Addr:              httpAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Register with AppHub after the HTTP server is ready.
	go func() {
		time.Sleep(500 * time.Millisecond)
		if err := client.AppHubManager.RegisterWebUI(httpAddr, "/rpi-4ch"); err != nil {
			logger.Warnf(logTag, "AppHub RegisterWebUI: %v (portal tile unavailable)", err)
		} else {
			logger.Infof(logTag, "AppHub: registered at /rpi-4ch → %s", httpAddr)
		}
	}()

	go func() {
		<-ctx.Done()
		if ctrl.mqttMgr != nil {
			ctrl.mqttMgr.stop()
		}
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
		_ = client.AppHubManager.UnregisterService()
	}()

	logger.Infof(logTag, "Web UI http://%s", httpAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf(logTag, "HTTP server: %v", err)
	}
}
