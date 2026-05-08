# Orbit OS SDK — API Reference (v26)

Import path: `github.com/OrbitOS-org/sdk-go/v26/client`

---

## Client

```go
func NewClientAuto(tcpHost string) (*Client, error)
func NewTCPClient(host string, port int) (*Client, error)
func NewUDSClient() (*Client, error)
func NewToolClient(host string, username, password string) (*Client, error)

func (c *Client) Close() error

func GetSDKAPIVersion() int32
func GetSDKAPIRevision() int32
func GetSDKAPIVersionInfo() string
```

Managers are accessed as fields on `*Client`:

```go
client.AIManager          *AIManager
client.AppHubManager      *AppHubManager
client.AuthManager        *AuthManager
client.BluetoothManager   *BluetoothManager
client.CameraManager      *CameraManager
client.DevelopmentManager *DevelopmentManager
client.EthernetManager    *EthernetManager
client.EventManager       *EventManager
client.FirewallManager    *FirewallManager
client.GpioManager        *GpioManager
client.I2CManager         *I2CManager
client.PackageManager     *PackageManager
client.PowerManager       *PowerManager
client.PwmManager         *PwmManager
client.SpiManager         *SpiManager
client.SystemManager      *SystemManager
client.UartManager        *UartManager
client.UpdateManager      *UpdateManager
client.VPNManager         *VPNManager
client.WiFiManager        *WiFiManager
```

---

## AIManager

```go
func (m *AIManager) LoadModel(modelID, modelPath string, backend aiv26.ModelBackend, execution aiv26.ExecutionMode) (*aiv26.LoadModelResponse, error)
func (m *AIManager) UploadAndLoadModel(modelID, localPath string, backend aiv26.ModelBackend, execution aiv26.ExecutionMode) (*aiv26.LoadModelResponse, error)
func (m *AIManager) UnloadModel(modelID string) error
func (m *AIManager) ListModels() ([]*aiv26.ModelInfo, error)
func (m *AIManager) IsModelLoaded(modelID string) (*aiv26.IsModelLoadedResponse, error)
func (m *AIManager) Infer(ctx context.Context, modelID string, inputData []byte, inputShape []int32, dtype aiv26.TensorDataType) (*aiv26.InferResponse, error)
```

---

## AppHubManager

```go
func (m *AppHubManager) RegisterWebUI(addr, route string) error
func (m *AppHubManager) RegisterService(req *apphubv26.RegisterServiceRequest) error
func (m *AppHubManager) UnregisterService() error
func (m *AppHubManager) AddRoute(path string) error
func (m *AppHubManager) RemoveRoute(path string) error
```

---

## AuthManager

```go
func (m *AuthManager) Login(username, password string) (token string, expiresAt int64, err error)
func (m *AuthManager) Logout(token string) error
```

---

## BluetoothManager

```go
// AdapterInfo is the SDK representation of the local BT adapter state.
type AdapterInfo struct {
    Address      string
    Name         string
    Powered      bool
    Discoverable bool
    Discovering  bool
}

// BtDevice is the SDK representation of a remote Bluetooth device.
type BtDevice struct {
    Address string
    Name    string
    Type    string // "classic", "le", "dual", "unknown"
    Bonded  bool
    RSSI    int32
}

// BondEvent is emitted during bonding.
type BondEvent struct {
    State string // "bonding", "bonded", "failed"
    Pin   string
    Error string
}

// BLEScanFilter filters BLE advertisement packets.
type BLEScanFilter struct {
    NamePrefix  string
    Address     string
    ServiceUUID string
}

// BLEScanResult is the SDK representation of a BLE advertisement.
type BLEScanResult struct {
    Address      string
    Name         string
    RSSI         int32
    Connectable  bool
    ServiceUUIDs []string
}

func (m *BluetoothManager) GetAdapterInfo(ctx context.Context) (*AdapterInfo, error)
func (m *BluetoothManager) EnableBluetooth(ctx context.Context) (bool, error)
func (m *BluetoothManager) DisableBluetooth(ctx context.Context) (bool, error)
func (m *BluetoothManager) GetLocalName(ctx context.Context) (string, error)
func (m *BluetoothManager) SetLocalName(ctx context.Context, name string) (bool, error)
func (m *BluetoothManager) SetDiscoverable(ctx context.Context, enable bool, durationSec int32) (bool, error)
func (m *BluetoothManager) ScanClassic(ctx context.Context, onResult func(BtDevice)) error
func (m *BluetoothManager) GetBondedDevices(ctx context.Context) ([]BtDevice, error)
func (m *BluetoothManager) BondDevice(ctx context.Context, address string, onEvent func(BondEvent)) error
func (m *BluetoothManager) RemoveBond(ctx context.Context, address string) (bool, error)
func (m *BluetoothManager) ConnectDevice(ctx context.Context, address string) (bool, error)
func (m *BluetoothManager) DisconnectDevice(ctx context.Context, address string) (bool, error)
func (m *BluetoothManager) GetConnectionState(ctx context.Context, address string) (string, error)
func (m *BluetoothManager) ScanBLE(ctx context.Context, filters []BLEScanFilter, onResult func(BLEScanResult)) error
```

---

## CameraManager

```go
// CameraDeviceInfo describes a V4L2 video device.
type CameraDeviceInfo struct {
    DeviceID         string
    Driver           string
    Card             string
    SupportedFormats []string
    Resolutions      []string
}

// CaptureImageResult holds the raw image returned by CaptureImage.
type CaptureImageResult struct {
    ImageData []byte
    Format    string
    Timestamp int64
}

func (m *CameraManager) ListDevices(ctx context.Context) ([]*CameraDeviceInfo, error)
func (m *CameraManager) GetDeviceInfo(ctx context.Context, deviceID string) (*CameraDeviceInfo, error)
func (m *CameraManager) LockCamera(ctx context.Context, deviceID, clientID string) error
func (m *CameraManager) UnlockCamera(ctx context.Context, deviceID, clientID string) error
func (m *CameraManager) CaptureImage(ctx context.Context, deviceID string, width, height int32, format string) (*CaptureImageResult, error)
func (m *CameraManager) StreamFrames(ctx context.Context, req *camv26.StreamFramesRequest) (grpc.ServerStreamingClient[camv26.Frame], error)
```

---

## DevelopmentManager

```go
// LogFilter controls which log entries are streamed.
type LogFilter struct {
    App      string           // filter by app name (empty = all)
    Tag      string           // filter by tag (empty = all)
    MinLevel devsvcv26.LogLevel
}

// LogEntry is the SDK representation of a log line.
type LogEntry struct {
    TimestampMs int64
    Timestamp   time.Time
    App         string
    Tag         string
    Level       devsvcv26.LogLevel
    LevelStr    string
    Message     string
}

func (d *DevelopmentManager) SubscribeLogs(ctx context.Context, filter LogFilter, onEntry func(LogEntry)) error
func (d *DevelopmentManager) SubscribeLogsAsync(ctx context.Context, filter LogFilter) <-chan LogEntry
```

---

## EthernetManager

```go
func (e *EthernetManager) ListEthernetInterfaces() ([]*ethsvcv26.EthernetLinkProperties, error)
func (e *EthernetManager) IsEthernetConnected(interfaceName string) (bool, error)
func (e *EthernetManager) GetEthernetLinkProperties(interfaceName string) (*ethsvcv26.EthernetLinkProperties, error)
func (e *EthernetManager) SetEthernetConfig(interfaceName string, enable, dhcpEnable bool, ipv4Address, ipv4Gateway string, ipv4Dns []string) (bool, error)
func (e *EthernetManager) EnableEthernet(interfaceName string) (bool, error)
func (e *EthernetManager) DisableEthernet(interfaceName string) (bool, error)
```

---

## EventManager

```go
func (e *EventManager) Subscribe(ctx context.Context, handler func(*eventsvcv26.Event), types ...eventsvcv26.EventType) error
```

---

## FirewallManager

```go
func (f *FirewallManager) ListZones() ([]*fwsvcv26.ZoneRequest, error)
func (f *FirewallManager) AddZone(name string, interfaces []string, inputPolicy, outputPolicy fwsvcv26.ZonePolicy, masquerade bool) (bool, error)
func (f *FirewallManager) RemoveZone(name string) (bool, error)
func (f *FirewallManager) ListRules() ([]*fwsvcv26.FirewallRule, error)
func (f *FirewallManager) AddRule(srcZone, dstZone string, protocol fwsvcv26.FirewallProtocol, srcIP string, destPort int32, action fwsvcv26.ZonePolicy, comment string) (bool, error)
func (f *FirewallManager) RemoveRule(id string) (bool, error)
func (f *FirewallManager) FlushRules() (bool, error)
```

---

## GpioManager

```go
type GpioLevel int32     // GPIO_LEVEL_LOW | GPIO_LEVEL_HIGH
type GpioDirection int32 // GPIO_DIR_OUT | GPIO_DIR_IN

// GpioPin identifies a GPIO line.
type GpioPin struct {
    Name       string
    Number     int32 // line offset within the chip
    ChipNumber int32 // chip index (e.g. 0 for /dev/gpiochip0)
}

func (m *GpioManager) ListPins() ([]*GpioPin, error)
func (m *GpioManager) GetLevel(pin *GpioPin) (GpioLevel, error)
func (m *GpioManager) SetLevel(pin *GpioPin, level GpioLevel) error
func (m *GpioManager) GetDirection(pin *GpioPin) (GpioDirection, error)
func (m *GpioManager) SetDirection(pin *GpioPin, dir GpioDirection) error
```

---

## I2CManager

```go
// I2CConfig holds bus configuration.
type I2CConfig struct {
    Bus             uint32
    ClockHz         uint32
    TenBitAddr      bool
    ClockStretching bool
}

// I2CTransferRequest describes an I2C operation.
// Operation is inferred: Data only → write; ReadLength only → read; both → write-then-read.
type I2CTransferRequest struct {
    Bus        uint32
    Address    uint32 // 7-bit device address
    Data       []byte
    ReadLength uint32
    Flags      uint32
}

func (m *I2CManager) ListBuses(opts ...grpc.CallOption) ([]uint32, error)
func (m *I2CManager) ScanBus(bus uint32, opts ...grpc.CallOption) ([]uint32, error)
func (m *I2CManager) GetConfig(bus uint32, opts ...grpc.CallOption) (I2CConfig, error)
func (m *I2CManager) SetConfig(cfg I2CConfig, opts ...grpc.CallOption) error
func (m *I2CManager) Transfer(req I2CTransferRequest, opts ...grpc.CallOption) ([]byte, error)
```

---

## PackageManager

```go
func (p *PackageManager) GetInstalledPackages() ([]*pmv26.InstalledPackage, error)
func (p *PackageManager) InstallPackageFromFile(ctx context.Context, orbPath string) error
func (p *PackageManager) RemovePackage(ctx context.Context, packageID string) error
```

---

## PowerManager

```go
// PowerResult is the SDK result for Reboot and Shutdown.
type PowerResult struct {
    Success bool
    Message string
}

func (p *PowerManager) Reboot(force bool, reason string) (*PowerResult, error)
func (p *PowerManager) Shutdown(force bool, reason string) (*PowerResult, error)
```

---

## PwmManager

```go
// PwmChannel identifies a PWM output.
type PwmChannel struct {
    Channel uint32
    Name    string
}

// PwmProperties is the current state of a PWM channel.
type PwmProperties struct {
    Channel     *PwmChannel
    Enabled     bool
    DutyCycle   float64 // 0.0–1.0
    FrequencyHz float64
}

func (m *PwmManager) ListChannels() ([]*PwmChannel, error)
func (m *PwmManager) GetProperties(ch *PwmChannel) (*PwmProperties, error)
func (m *PwmManager) SetPwm(ch *PwmChannel, dutyCycle, frequencyHz float64) error
func (m *PwmManager) StopPwm(ch *PwmChannel) error
```

---

## SpiManager

```go
// SpiConfig holds SPI bus configuration.
type SpiConfig struct {
    Bus         uint32
    ChipSelect  uint32
    MaxSpeedHz  uint32
    BitsPerWord uint32
    Mode        int  // 0–3 (CPOL/CPHA)
    LSBFirst    bool
}

func (m *SpiManager) ListDevices() ([]string, error)
func (m *SpiManager) GetConfig(bus, cs uint32) (*SpiConfig, error)
func (m *SpiManager) SetConfig(cfg SpiConfig) error
func (m *SpiManager) Transfer(bus, cs uint32, dataOut []byte, readLength uint32) ([]byte, error)
```

---

## SystemManager

```go
func (s *SystemManager) GetApiVersion() (version int64, revision int64, err error)
func (s *SystemManager) GetApiVersionInfo() (string, error)

// Device identity
func (s *SystemManager) GetDeviceName() (string, error)
func (s *SystemManager) GetMachineId() (string, error)
func (s *SystemManager) GetSystemUuid() (string, error)
func (s *SystemManager) GetArchitecture() (string, error)
func (s *SystemManager) GetSocModel() (string, error)
func (s *SystemManager) GetSocVendor() (string, error)
func (s *SystemManager) GetBoardModel() (string, error)
func (s *SystemManager) GetBoardVendor() (string, error)
func (s *SystemManager) GetBoardSerial() (string, error)
func (s *SystemManager) GetCpuSerial() (string, error)
func (s *SystemManager) GetHardwareModel() (string, error)
func (s *SystemManager) GetHardwareVersion() (string, error)

// CPU & memory
func (s *SystemManager) GetCpuModel() (string, error)
func (s *SystemManager) GetCpuCores() (int64, error)
func (s *SystemManager) GetCpuThreads() (int64, error)
func (s *SystemManager) GetCpuMinMhz() (float64, error)
func (s *SystemManager) GetCpuMaxMhz() (float64, error)
func (s *SystemManager) GetTotalRAM() (uint64, error)

// OS & runtime
func (s *SystemManager) GetOsName() (string, error)
func (s *SystemManager) GetOsVersion() (string, error)
func (s *SystemManager) GetOsRevision() (string, error)
func (s *SystemManager) GetKernelVersion() (string, error)
func (s *SystemManager) GetDistro() (string, error)
func (s *SystemManager) GetDistroVersion() (string, error)
func (s *SystemManager) GetRuntimeVersion() (string, error)
func (s *SystemManager) GetRuntimeBuildDate() (string, error)
func (s *SystemManager) GetBuildDate() (string, error)

// Metrics
func (s *SystemManager) GetMetrics() (*systemv26.MetricsInfoResponse, error)
```

---

## UartManager

```go
type UartParity int        // UartParityNone | UartParityEven | UartParityOdd
type UartStopBits int      // UartStopBits1 | UartStopBits2
type UartFlowControl int   // UartFlowNone | UartFlowHardware | UartFlowSoftware

// UartConfig holds the configuration to open a UART port.
type UartConfig struct {
    Port        string
    Baudrate    int
    DataBits    int // 5, 6, 7 or 8
    Parity      UartParity
    StopBits    UartStopBits
    FlowControl UartFlowControl
}

func (m *UartManager) ListPorts() ([]string, error)
func (m *UartManager) Open(cfg UartConfig) error
func (m *UartManager) Close(port string) error
func (m *UartManager) GetConfig(port string) (*UartConfig, error)
func (m *UartManager) Write(port string, data []byte) (int, error)
func (m *UartManager) Listen(ctx context.Context, port string, maxChunkSize int, onChunk func([]byte)) error
func (m *UartManager) ListenAsync(ctx context.Context, port string, maxChunkSize int) (<-chan []byte, error)
```

---

## UpdateManager

```go
func (u *UpdateManager) InstallOtaFromFile(ctx context.Context, otaPath string) error
func (u *UpdateManager) FactoryReset() (bool, error)
```

---

## VPNManager

```go
func (v *VPNManager) GetCapabilities() (*vpnv26.VpnCapabilities, error)
func (v *VPNManager) ListProfiles() ([]*vpnv26.VpnProfile, error)
func (v *VPNManager) ApplyProfile(profile *vpnv26.VpnProfile, connectAfterApply bool) (string, error)
func (v *VPNManager) ApplyWireGuard(displayName string, configData []byte, autoConnect bool) (string, error)
func (v *VPNManager) ApplyOpenVPN(displayName string, configData []byte, autoConnect bool) (string, error)
func (v *VPNManager) RemoveProfile(profileID string) (bool, error)
func (v *VPNManager) Connect(profileID string) (sessionID string, err error)
func (v *VPNManager) Disconnect(profileID string) (bool, error)
func (v *VPNManager) IsConnected() (bool, error)
func (v *VPNManager) GetStatus() (*vpnv26.Session, string, error)
func (v *VPNManager) ListSessions() ([]*vpnv26.Session, error)
func (v *VPNManager) WatchEvents(profileID string, handler func(*vpnv26.VPNEvent)) error
```

---

## WiFiManager

```go
func (w *WiFiManager) ListInterfaces() ([]*wifisvcv26.WiFiLinkProperties, error)
func (w *WiFiManager) GetLinkProperties(ifname string) (*wifisvcv26.WiFiLinkProperties, error)
func (w *WiFiManager) IsConnected(ifname string) (bool, error)
func (w *WiFiManager) GetMode(ifname string) (wifisvcv26.WiFiMode, error)
func (w *WiFiManager) SetModeClient(ifname string) (bool, error)
func (w *WiFiManager) SetClientConfig(ifname, ssid, password, security string, dhcpEnable bool, ipv4Address, ipv4Gateway string, ipv4Dns []string) (bool, error)
func (w *WiFiManager) GetClientProperties(ifname string) (*wifisvcv26.ClientProperties, error)
func (w *WiFiManager) Connect(ifname string) (bool, error)
func (w *WiFiManager) Disconnect(ifname string) (bool, error)
func (w *WiFiManager) Scan(ifname string, forceRescan bool) ([]*wifisvcv26.ScannedNetwork, error)
func (w *WiFiManager) StartHotspot(ifname, ssid, password, band string, channel int32) (bool, error)
func (w *WiFiManager) StopHotspot(ifname string) (bool, error)
func (w *WiFiManager) GetAPProperties(ifname string) (*wifisvcv26.APProperties, error)
```
