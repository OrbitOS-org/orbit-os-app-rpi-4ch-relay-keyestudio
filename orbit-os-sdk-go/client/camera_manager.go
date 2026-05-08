package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	camv26 "github.com/OrbitOS-org/sdk-go/v26/api/camera_service/v26"
)

// ── Types ─────────────────────────────────────────────────────────────────────

// CameraDeviceInfo describes a V4L camera node (formats, resolutions, driver).
type CameraDeviceInfo struct {
	DeviceID         string
	Driver           string
	Card             string
	SupportedFormats []string
	Resolutions      []string
}

// CaptureImageResult is one still frame returned by CaptureImage.
type CaptureImageResult struct {
	ImageData []byte
	Format    string
	Timestamp int64
}

// ── CameraManager ─────────────────────────────────────────────────────────────

// CameraManager is the SDK client for CameraService.
type CameraManager struct {
	client camv26.CameraServiceClient
}

// NewCameraManager constructs a CameraManager.
func NewCameraManager(client camv26.CameraServiceClient) *CameraManager {
	return &CameraManager{client: client}
}

// ListDevices returns all V4L2 video nodes available on the device.
func (m *CameraManager) ListDevices(ctx context.Context) ([]*CameraDeviceInfo, error) {
	resp, err := m.client.ListDevices(ctx, &camv26.ListDevicesRequest{})
	if err != nil {
		return nil, fmt.Errorf("ListDevices: %w", err)
	}
	out := make([]*CameraDeviceInfo, 0, len(resp.GetDevices()))
	for _, d := range resp.GetDevices() {
		out = append(out, &CameraDeviceInfo{
			DeviceID:         d.GetDeviceId(),
			Driver:           d.GetDriver(),
			Card:             d.GetCard(),
			SupportedFormats: append([]string(nil), d.GetSupportedFormats()...),
			Resolutions:      append([]string(nil), d.GetResolutions()...),
		})
	}
	return out, nil
}

// GetDeviceInfo returns metadata for a specific camera device.
func (m *CameraManager) GetDeviceInfo(ctx context.Context, deviceID string) (*CameraDeviceInfo, error) {
	resp, err := m.client.GetDeviceInfo(ctx, &camv26.DeviceInfoRequest{DeviceId: deviceID})
	if err != nil {
		return nil, fmt.Errorf("GetDeviceInfo: %w", err)
	}
	return &CameraDeviceInfo{
		DeviceID:         resp.GetDeviceId(),
		Driver:           resp.GetDriver(),
		Card:             resp.GetCard(),
		SupportedFormats: append([]string(nil), resp.GetSupportedFormats()...),
		Resolutions:      append([]string(nil), resp.GetResolutions()...),
	}, nil
}

// Lock acquires an exclusive lock on a camera and returns a LockedCamera handle
// for capture and streaming. Call Unlock on the handle when done.
func (m *CameraManager) Lock(ctx context.Context, deviceID, clientID string) (*LockedCamera, error) {
	resp, err := m.client.LockCamera(ctx, &camv26.LockRequest{
		DeviceId: deviceID,
		ClientId: clientID,
	})
	if err != nil {
		return nil, fmt.Errorf("LockCamera: %w", err)
	}
	if err := rpcError(resp.GetError()); err != nil {
		return nil, err
	}
	if !resp.GetSuccess() {
		return nil, fmt.Errorf("LockCamera: server reported failure")
	}
	return &LockedCamera{client: m.client, deviceID: deviceID, clientID: clientID}, nil
}

// ── LockedCamera ──────────────────────────────────────────────────────────────

// LockedCamera is a handle to an exclusively locked camera device.
type LockedCamera struct {
	client   camv26.CameraServiceClient
	deviceID string
	clientID string
}

// Unlock releases the camera lock.
func (c *LockedCamera) Unlock(ctx context.Context) error {
	resp, err := c.client.UnlockCamera(ctx, &camv26.UnlockRequest{
		DeviceId: c.deviceID,
		ClientId: c.clientID,
	})
	if err != nil {
		return fmt.Errorf("UnlockCamera: %w", err)
	}
	if err := rpcError(resp.GetError()); err != nil {
		return err
	}
	if !resp.GetSuccess() {
		return fmt.Errorf("UnlockCamera: server reported failure")
	}
	return nil
}

// CaptureImage captures a single frame from the locked camera.
func (c *LockedCamera) CaptureImage(ctx context.Context, width, height int32, format string) (*CaptureImageResult, error) {
	resp, err := c.client.CaptureImage(ctx, &camv26.CaptureImageRequest{
		DeviceId: c.deviceID,
		Width:    width,
		Height:   height,
		Format:   format,
	})
	if err != nil {
		return nil, fmt.Errorf("CaptureImage: %w", err)
	}
	if err := rpcError(resp.GetError()); err != nil {
		return nil, err
	}
	if !resp.GetSuccess() {
		return nil, fmt.Errorf("CaptureImage: server reported failure")
	}
	return &CaptureImageResult{
		ImageData: resp.GetImageData(),
		Format:    resp.GetFormat(),
		Timestamp: resp.GetTimestamp(),
	}, nil
}

// StreamFrames opens a server-streaming frame stream at the requested FPS and resolution.
// Read from the returned stream until io.EOF or error.
func (c *LockedCamera) StreamFrames(ctx context.Context, fps, width, height int32) (grpc.ServerStreamingClient[camv26.Frame], error) {
	stream, err := c.client.StreamFrames(ctx, &camv26.StreamFramesRequest{
		DeviceId: c.deviceID,
		Fps:      fps,
		Width:    width,
		Height:   height,
	})
	if err != nil {
		return nil, fmt.Errorf("StreamFrames: %w", err)
	}
	return stream, nil
}
