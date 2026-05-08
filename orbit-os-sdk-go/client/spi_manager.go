package client

import (
	"context"
	"fmt"

	"github.com/OrbitOS-org/sdk-go/v26/api/common"
	spiv26 "github.com/OrbitOS-org/sdk-go/v26/api/spi_service/v26"
)

// ─── Types ────────────────────────────────────────────────────────────────────

// SpiConfig holds the configuration of a SPI device.
type SpiConfig struct {
	Bus         uint32
	ChipSelect  uint32
	MaxSpeedHz  uint32
	BitsPerWord uint32
	Mode        int
	LSBFirst    bool
}

// ─── SpiManager ───────────────────────────────────────────────────────────────

// SpiManager is the SDK client for SpiService.
type SpiManager struct {
	client spiv26.SpiServiceClient
	ctx    context.Context
}

// NewSpiManager creates a new SpiManager.
func NewSpiManager(client spiv26.SpiServiceClient, ctx context.Context) *SpiManager {
	return &SpiManager{client: client, ctx: ctx}
}

// ListDevices returns available SPI device names (e.g. "spidev0.0").
func (m *SpiManager) ListDevices() ([]string, error) {
	resp, err := m.client.ListSpiBuses(m.ctx, &common.Void{})
	if err != nil {
		return nil, fmt.Errorf("ListSpiBuses: %w", err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != common.ErrorCode_ERROR_CODE_NONE {
		return nil, fmt.Errorf("ListSpiBuses: %s", resp.GetError().GetMessage())
	}
	return resp.GetDevices(), nil
}

// Open configures the SPI device at (bus, cs) and returns a handle for all subsequent operations.
func (m *SpiManager) Open(bus, cs uint32, maxSpeedHz, bitsPerWord uint32, mode int, lsbFirst bool) (*SpiDevice, error) {
	resp, err := m.client.SetSpiConfig(m.ctx, &spiv26.SpiConfigRequest{
		Bus:         bus,
		ChipSelect:  cs,
		MaxSpeedHz:  maxSpeedHz,
		BitsPerWord: bitsPerWord,
		Mode:        spiv26.SpiMode(mode),
		LsbFirst:    lsbFirst,
	})
	if err != nil {
		return nil, fmt.Errorf("SetSpiConfig spidev%d.%d: %w", bus, cs, err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != common.ErrorCode_ERROR_CODE_NONE {
		return nil, fmt.Errorf("SetSpiConfig spidev%d.%d: %s", bus, cs, resp.GetError().GetMessage())
	}
	return &SpiDevice{client: m.client, ctx: m.ctx, bus: bus, cs: cs}, nil
}

// ─── SpiDevice ────────────────────────────────────────────────────────────────

// SpiDevice is a handle to a configured SPI device (bus + chip select).
type SpiDevice struct {
	client spiv26.SpiServiceClient
	ctx    context.Context
	bus    uint32
	cs     uint32
}

// GetConfig returns the current configuration of the device.
func (d *SpiDevice) GetConfig() (*SpiConfig, error) {
	resp, err := d.client.GetSpiConfig(d.ctx, &spiv26.SpiBusRequest{
		Bus:        d.bus,
		ChipSelect: d.cs,
	})
	if err != nil {
		return nil, fmt.Errorf("GetSpiConfig spidev%d.%d: %w", d.bus, d.cs, err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != common.ErrorCode_ERROR_CODE_NONE {
		return nil, fmt.Errorf("GetSpiConfig spidev%d.%d: %s", d.bus, d.cs, resp.GetError().GetMessage())
	}
	return &SpiConfig{
		Bus:         resp.GetBus(),
		ChipSelect:  resp.GetChipSelect(),
		MaxSpeedHz:  resp.GetMaxSpeedHz(),
		BitsPerWord: resp.GetBitsPerWord(),
		Mode:        int(resp.GetMode()),
		LSBFirst:    resp.GetLsbFirst(),
	}, nil
}

// Transfer performs a full-duplex SPI transfer.
// dataOut is sent on MOSI; readLength controls how many MISO bytes are returned
// (0 = write-only; when smaller than len(dataOut) the last readLength bytes are returned).
func (d *SpiDevice) Transfer(dataOut []byte, readLength uint32) ([]byte, error) {
	resp, err := d.client.SpiTransfer(d.ctx, &spiv26.SpiTransferRequest{
		Bus:        d.bus,
		ChipSelect: d.cs,
		DataOut:    dataOut,
		ReadLength: readLength,
	})
	if err != nil {
		return nil, fmt.Errorf("SpiTransfer spidev%d.%d: %w", d.bus, d.cs, err)
	}
	if resp.GetError() != nil && resp.GetError().GetCode() != common.ErrorCode_ERROR_CODE_NONE {
		return nil, fmt.Errorf("SpiTransfer spidev%d.%d: %s", d.bus, d.cs, resp.GetError().GetMessage())
	}
	return resp.GetDataIn(), nil
}
