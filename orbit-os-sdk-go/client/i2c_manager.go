package client

import (
	"context"
	"fmt"

	common "github.com/OrbitOS-org/sdk-go/v26/api/common"
	i2cv26 "github.com/OrbitOS-org/sdk-go/v26/api/i2c_service/v26"
)

// ─── Types ────────────────────────────────────────────────────────────────────

// I2CConfig holds the configuration of an I2C bus.
type I2CConfig struct {
	Bus             uint32
	ClockHz         uint32
	TenBitAddr      bool
	ClockStretching bool
}

// ─── I2CManager ───────────────────────────────────────────────────────────────

// I2CManager provides gRPC-based access to I2C buses on the device.
type I2CManager struct {
	client i2cv26.I2CServiceClient
	ctx    context.Context
}

// NewI2CManager creates a new I2CManager backed by the given gRPC client.
func NewI2CManager(client i2cv26.I2CServiceClient, ctx context.Context) *I2CManager {
	return &I2CManager{client: client, ctx: ctx}
}

// ListBuses returns the bus numbers of all available I2C adapters on the device.
func (m *I2CManager) ListBuses() ([]uint32, error) {
	resp, err := m.client.ListI2CBuses(m.ctx, &common.Void{})
	if err != nil {
		return nil, fmt.Errorf("ListI2CBuses: %w", err)
	}
	if resp.Error != nil && resp.Error.Code != 0 {
		return nil, fmt.Errorf("ListI2CBuses: %s", resp.Error.Message)
	}
	return resp.Buses, nil
}

// Open configures the given bus and returns a handle for all subsequent operations.
func (m *I2CManager) Open(bus uint32, clockHz uint32, tenBitAddr bool, clockStretching bool) (*I2CBus, error) {
	resp, err := m.client.SetI2CConfig(m.ctx, &i2cv26.I2CConfigRequest{
		Bus:             bus,
		ClockHz:         clockHz,
		TenBitAddr:      tenBitAddr,
		ClockStretching: clockStretching,
	})
	if err != nil {
		return nil, fmt.Errorf("SetI2CConfig bus %d: %w", bus, err)
	}
	if resp.Error != nil && resp.Error.Code != 0 {
		return nil, fmt.Errorf("SetI2CConfig bus %d: %s", bus, resp.Error.Message)
	}
	return &I2CBus{client: m.client, ctx: m.ctx, bus: bus}, nil
}

// ─── I2CBus ───────────────────────────────────────────────────────────────────

// I2CBus is a handle to a specific I2C bus.
type I2CBus struct {
	client i2cv26.I2CServiceClient
	ctx    context.Context
	bus    uint32
}

// Scan probes all 7-bit addresses (0x03–0x77) and returns those that respond.
func (b *I2CBus) Scan() ([]uint32, error) {
	resp, err := b.client.ScanI2CBus(b.ctx, &i2cv26.I2CBusRequest{Bus: b.bus})
	if err != nil {
		return nil, fmt.Errorf("ScanI2CBus bus %d: %w", b.bus, err)
	}
	if resp.Error != nil && resp.Error.Code != 0 {
		return nil, fmt.Errorf("ScanI2CBus bus %d: %s", b.bus, resp.Error.Message)
	}
	return resp.Addresses, nil
}

// GetConfig returns the current configuration of the bus.
func (b *I2CBus) GetConfig() (I2CConfig, error) {
	resp, err := b.client.GetI2CConfig(b.ctx, &i2cv26.I2CBusRequest{Bus: b.bus})
	if err != nil {
		return I2CConfig{}, fmt.Errorf("GetI2CConfig bus %d: %w", b.bus, err)
	}
	if resp.Error != nil && resp.Error.Code != 0 {
		return I2CConfig{}, fmt.Errorf("GetI2CConfig bus %d: %s", b.bus, resp.Error.Message)
	}
	return I2CConfig{
		Bus:             resp.Bus,
		ClockHz:         resp.ClockHz,
		TenBitAddr:      resp.TenBitAddr,
		ClockStretching: resp.ClockStretching,
	}, nil
}

// Transfer performs an I2C operation on the device at addr:
//   - Write:             Transfer(addr, data, 0, 0)
//   - Read:              Transfer(addr, nil, n, 0)
//   - Write-then-read:   Transfer(addr, data, n, 0)
//
// flags is passed directly as i2c_msg flags (0 for most uses).
// Returns the received bytes (nil for write-only operations).
func (b *I2CBus) Transfer(addr uint32, data []byte, readLen uint32, flags uint32) ([]byte, error) {
	resp, err := b.client.I2CTransfer(b.ctx, &i2cv26.I2CTransferRequest{
		Bus:        b.bus,
		Address:    addr,
		Data:       data,
		ReadLength: readLen,
		Flags:      flags,
	})
	if err != nil {
		return nil, fmt.Errorf("I2CTransfer bus %d addr 0x%02X: %w", b.bus, addr, err)
	}
	if resp.Error != nil && resp.Error.Code != 0 {
		return nil, fmt.Errorf("I2CTransfer bus %d addr 0x%02X: %s", b.bus, addr, resp.Error.Message)
	}
	return resp.Data, nil
}
