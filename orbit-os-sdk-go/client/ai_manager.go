package client

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	aiv26 "github.com/OrbitOS-org/sdk-go/v26/api/ai_service/v26"
)

const uploadChunkSize = 256 * 1024 // 256 KB per chunk

// ── AIManager ─────────────────────────────────────────────────────────────────

// AIManager is the SDK client for AiService.
type AIManager struct {
	client aiv26.AiServiceClient
	ctx    context.Context
}

// NewAIManager constructs an AIManager.
func NewAIManager(client aiv26.AiServiceClient, ctx context.Context) *AIManager {
	return &AIManager{client: client, ctx: ctx}
}

// ListModels returns metadata for all currently loaded models.
func (m *AIManager) ListModels() ([]*aiv26.ModelInfo, error) {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	resp, err := m.client.ListModels(ctx, &aiv26.ListModelsRequest{})
	if err != nil {
		return nil, fmt.Errorf("ListModels: %w", err)
	}
	return resp.GetModels(), nil
}

// LoadModel loads a model file already present on the device filesystem.
// Returns an AIModel handle for inference and lifecycle operations.
func (m *AIManager) LoadModel(modelID, modelPath string, backend aiv26.ModelBackend, execution aiv26.ExecutionMode) (*AIModel, error) {
	ctx, cancel := context.WithTimeout(m.ctx, 60*time.Second)
	defer cancel()

	resp, err := m.client.LoadModel(ctx, &aiv26.LoadModelRequest{
		ModelId:   modelID,
		ModelPath: modelPath,
		Backend:   backend,
		Execution: execution,
	})
	if err != nil {
		return nil, fmt.Errorf("LoadModel: %w", err)
	}
	if !resp.GetSuccess() {
		if e := resp.GetError(); e != nil {
			return nil, fmt.Errorf("LoadModel: %s", e.GetMessage())
		}
		return nil, fmt.Errorf("LoadModel: server reported failure")
	}
	return &AIModel{client: m.client, ctx: m.ctx, modelID: modelID, Response: resp}, nil
}

// UploadAndLoadModel streams a local model file to the device then loads it.
// Use this when the model file does not yet exist on the device filesystem.
// Returns an AIModel handle for inference and lifecycle operations.
func (m *AIManager) UploadAndLoadModel(modelID, localPath string, backend aiv26.ModelBackend, execution aiv26.ExecutionMode) (*AIModel, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("UploadAndLoadModel: open %q: %w", localPath, err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("UploadAndLoadModel: stat %q: %w", localPath, err)
	}

	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Minute)
	defer cancel()

	stream, err := m.client.UploadAndLoadModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("UploadAndLoadModel: open stream: %w", err)
	}

	if err := stream.Send(&aiv26.UploadModelChunk{
		Payload: &aiv26.UploadModelChunk_Meta{
			Meta: &aiv26.UploadModelMeta{
				ModelId:    modelID,
				Backend:    backend,
				Execution:  execution,
				TotalBytes: fi.Size(),
				Filename:   filepath.Base(localPath),
			},
		},
	}); err != nil {
		return nil, fmt.Errorf("UploadAndLoadModel: send meta: %w", err)
	}

	buf := make([]byte, uploadChunkSize)
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&aiv26.UploadModelChunk{
				Payload: &aiv26.UploadModelChunk_Data{Data: buf[:n]},
			}); sendErr != nil {
				return nil, fmt.Errorf("UploadAndLoadModel: send chunk: %w", sendErr)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("UploadAndLoadModel: read file: %w", readErr)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, fmt.Errorf("UploadAndLoadModel: %w", err)
	}
	if !resp.GetSuccess() {
		if e := resp.GetError(); e != nil {
			return nil, fmt.Errorf("UploadAndLoadModel: %s", e.GetMessage())
		}
		return nil, fmt.Errorf("UploadAndLoadModel: server reported failure")
	}
	return &AIModel{client: m.client, ctx: m.ctx, modelID: modelID, Response: resp}, nil
}

// ── AIModel ───────────────────────────────────────────────────────────────────

// AIModel is a handle to a loaded model. Response holds the tensor schema
// returned at load time (inputs/outputs).
type AIModel struct {
	client   aiv26.AiServiceClient
	ctx      context.Context
	modelID  string
	Response *aiv26.LoadModelResponse
}

// Unload frees the model from the inference backend.
func (m *AIModel) Unload() error {
	ctx, cancel := context.WithTimeout(m.ctx, 15*time.Second)
	defer cancel()

	resp, err := m.client.UnloadModel(ctx, &aiv26.UnloadModelRequest{ModelId: m.modelID})
	if err != nil {
		return fmt.Errorf("UnloadModel: %w", err)
	}
	if !resp.GetSuccess() {
		if e := resp.GetError(); e != nil {
			return fmt.Errorf("UnloadModel: %s", e.GetMessage())
		}
		return fmt.Errorf("UnloadModel: server reported failure")
	}
	return nil
}

// IsLoaded returns true if the model is currently loaded, along with its tensor schema.
func (m *AIModel) IsLoaded() (*aiv26.IsModelLoadedResponse, error) {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	resp, err := m.client.IsModelLoaded(ctx, &aiv26.IsModelLoadedRequest{ModelId: m.modelID})
	if err != nil {
		return nil, fmt.Errorf("IsModelLoaded: %w", err)
	}
	return resp, nil
}

// Infer runs one synchronous forward pass. The caller controls the timeout via ctx.
// inputData is raw little-endian bytes. inputShape is optional (nil = use schema from load).
func (m *AIModel) Infer(ctx context.Context, inputData []byte, inputShape []int32, dtype aiv26.TensorDataType) (*aiv26.InferResponse, error) {
	resp, err := m.client.Infer(ctx, &aiv26.InferRequest{
		ModelId:    m.modelID,
		InputData:  inputData,
		InputShape: inputShape,
		InputDtype: dtype,
	})
	if err != nil {
		return nil, fmt.Errorf("Infer: %w", err)
	}
	if !resp.GetSuccess() {
		if e := resp.GetError(); e != nil {
			return nil, fmt.Errorf("Infer: %s", e.GetMessage())
		}
		return nil, fmt.Errorf("Infer: server reported failure")
	}
	return resp, nil
}

// StreamInfer opens a bidirectional inference stream for continuous inference.
// Use Send to submit requests and Recv to read results. Call Close when done.
func (m *AIModel) StreamInfer(ctx context.Context) (*AIInferStream, error) {
	stream, err := m.client.StreamInfer(ctx)
	if err != nil {
		return nil, fmt.Errorf("StreamInfer: %w", err)
	}
	return &AIInferStream{modelID: m.modelID, stream: stream}, nil
}

// ── AIInferStream ─────────────────────────────────────────────────────────────

// AIInferStream is a handle to an open bidirectional inference stream.
type AIInferStream struct {
	modelID string
	stream  interface {
		Send(*aiv26.InferRequest) error
		Recv() (*aiv26.InferResponse, error)
		CloseSend() error
	}
}

// Send submits an inference request on the stream.
func (s *AIInferStream) Send(inputData []byte, inputShape []int32, dtype aiv26.TensorDataType) error {
	return s.stream.Send(&aiv26.InferRequest{
		ModelId:    s.modelID,
		InputData:  inputData,
		InputShape: inputShape,
		InputDtype: dtype,
	})
}

// Recv reads the next inference result. Returns io.EOF when the server closes the stream.
func (s *AIInferStream) Recv() (*aiv26.InferResponse, error) {
	resp, err := s.stream.Recv()
	if err != nil {
		return nil, err
	}
	if !resp.GetSuccess() {
		if e := resp.GetError(); e != nil {
			return nil, fmt.Errorf("StreamInfer: %s", e.GetMessage())
		}
		return nil, fmt.Errorf("StreamInfer: server reported failure")
	}
	return resp, nil
}

// Close signals the end of the send side of the stream.
func (s *AIInferStream) Close() error {
	return s.stream.CloseSend()
}
