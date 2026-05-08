package client

import (
	"context"
	"fmt"
	"io"
	"time"

	devsvcv26 "github.com/OrbitOS-org/sdk-go/v26/api/development_service/v26"
)

const reconnectDelay = 2 * time.Second

// LogLevel represents the minimum log level for filtering.
type LogLevel int

const (
	// LOG_LEVEL_DEBUG filters for DEBUG and above.
	LOG_LEVEL_DEBUG LogLevel = iota
	// LOG_LEVEL_INFO filters for INFO and above.
	LOG_LEVEL_INFO
	// LOG_LEVEL_WARNING filters for WARNING and above.
	LOG_LEVEL_WARNING
	// LOG_LEVEL_ERROR filters for ERROR and above.
	LOG_LEVEL_ERROR
	// LOG_LEVEL_FATAL filters for FATAL only.
	LOG_LEVEL_FATAL
)

// toGrpcLogLevel converts internal LogLevel to gRPC LogLevel.
func toGrpcLogLevel(level LogLevel) devsvcv26.LogLevel {
	switch level {
	case LOG_LEVEL_DEBUG:
		return devsvcv26.LogLevel_LOG_LEVEL_DEBUG
	case LOG_LEVEL_INFO:
		return devsvcv26.LogLevel_LOG_LEVEL_INFO
	case LOG_LEVEL_WARNING:
		return devsvcv26.LogLevel_LOG_LEVEL_WARNING
	case LOG_LEVEL_ERROR:
		return devsvcv26.LogLevel_LOG_LEVEL_ERROR
	case LOG_LEVEL_FATAL:
		return devsvcv26.LogLevel_LOG_LEVEL_FATAL
	default:
		return devsvcv26.LogLevel_LOG_LEVEL_DEBUG
	}
}

// LogEntry is a received log entry from the DevelopmentService.
type LogEntry struct {
	TimestampMs int64
	Timestamp   time.Time
	App         string
	Tag         string
	Level       LogLevel
	LevelStr    string
	Message     string
}

// LogFilter controls which entries are streamed from the server.
type LogFilter struct {
	// App filters by app name substring (empty = all apps)
	App string
	// Tag filters by tag substring (empty = all tags)
	Tag string
	// MinLevel only delivers entries at this level or above (default: DEBUG = all)
	MinLevel LogLevel
}

// DevelopmentManager provides access to the DevelopmentService gRPC service.
type DevelopmentManager struct {
	client devsvcv26.DevelopmentServiceClient
	ctx    context.Context
}

func NewDevelopmentManager(client devsvcv26.DevelopmentServiceClient, ctx context.Context) *DevelopmentManager {
	return &DevelopmentManager{client: client, ctx: ctx}
}

// SubscribeLogs opens a server-streaming log subscription. Each received
// entry is forwarded to the provided callback. The call blocks until the
// context is cancelled, the server closes the stream, or a receive error
// occurs. Returns nil if the stream ended normally (ctx cancelled).
func (d *DevelopmentManager) SubscribeLogs(ctx context.Context, app string, tag string, level LogLevel, onEntry func(LogEntry)) error {
	req := &devsvcv26.LogSubscribeRequest{
		App:   app,
		Tag:   tag,
		Level: toGrpcLogLevel(level),
	}

	stream, err := d.client.SubscribeLogs(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to subscribe logs: %w", err)
	}

	for {
		pb, err := stream.Recv()
		if err != nil {
			if err == io.EOF || ctx.Err() != nil {
				return nil // normal end
			}
			return fmt.Errorf("log stream error: %w", err)
		}
		onEntry(toLogEntry(pb))
	}
}

// SubscribeLogsAsync starts SubscribeLogs in a goroutine and returns a
// channel. It reconnects automatically if the stream drops. The channel is
// only closed when ctx is cancelled.
func (d *DevelopmentManager) SubscribeLogsAsync(ctx context.Context, app string, tag string, level LogLevel) <-chan LogEntry {
	ch := make(chan LogEntry, 64)
	go func() {
		defer close(ch)
		for {
			err := d.SubscribeLogs(ctx, app, tag, level, func(e LogEntry) {
				select {
				case ch <- e:
				case <-ctx.Done():
				}
			})

			// Context cancelled — stop reconnecting
			if ctx.Err() != nil {
				return
			}

			// Stream ended (server closed or network error) — wait and retry
			if err != nil {
				fmt.Printf("[DevelopmentManager] stream error: %v — reconnecting in %s\n", err, reconnectDelay)
			} else {
				fmt.Printf("[DevelopmentManager] stream ended — reconnecting in %s\n", reconnectDelay)
			}
			select {
			case <-time.After(reconnectDelay):
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toLogEntry(pb *devsvcv26.LogEntry) LogEntry {
	return LogEntry{
		TimestampMs: pb.GetTimestampMs(),
		Timestamp:   time.UnixMilli(pb.GetTimestampMs()),
		App:         pb.GetApp(),
		Tag:         pb.GetTag(),
		Level:       LogLevel(pb.GetLevel()),
		LevelStr:    levelToStr(pb.GetLevel()),
		Message:     pb.GetMessage(),
	}
}

func levelToStr(l devsvcv26.LogLevel) string {
	switch l {
	case devsvcv26.LogLevel_LOG_LEVEL_DEBUG:
		return "D"
	case devsvcv26.LogLevel_LOG_LEVEL_INFO:
		return "I"
	case devsvcv26.LogLevel_LOG_LEVEL_WARNING:
		return "W"
	case devsvcv26.LogLevel_LOG_LEVEL_ERROR:
		return "E"
	case devsvcv26.LogLevel_LOG_LEVEL_FATAL:
		return "F"
	default:
		return "?"
	}
}
