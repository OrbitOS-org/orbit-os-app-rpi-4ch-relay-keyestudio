package client

import (
	"context"
	"io"

	eventsvcv26 "github.com/OrbitOS-org/sdk-go/v26/api/event_service/v26"
)

// EventType represents the type of system event.
type EventType int

const (
	// EVENT_APP_INSTALLED - app installed event
	EVENT_APP_INSTALLED EventType = iota + 1
	// EVENT_APP_REMOVED - app removed event
	EVENT_APP_REMOVED
	// EVENT_APP_UPDATED - app updated event
	EVENT_APP_UPDATED
	// EVENT_APP_STARTED - app started event
	EVENT_APP_STARTED
	// EVENT_APP_STOPPED - app stopped event
	EVENT_APP_STOPPED
	// EVENT_APP_CRASHED - app crashed event
	EVENT_APP_CRASHED
	// EVENT_APP_REJECTED - app rejected event
	EVENT_APP_REJECTED
	// EVENT_SYSTEM_REBOOT - system reboot event
	EVENT_SYSTEM_REBOOT
	// EVENT_SYSTEM_FACTORY_RESET - system factory reset event
	EVENT_SYSTEM_FACTORY_RESET
	// EVENT_SYSTEM_UPDATE - system update event
	EVENT_SYSTEM_UPDATE
	// EVENT_NET_UP - network up event
	EVENT_NET_UP
	// EVENT_NET_DOWN - network down event
	EVENT_NET_DOWN
)

// toGrpcEventType converts internal EventType to gRPC EventType.
func toGrpcEventType(t EventType) eventsvcv26.EventType {
	switch t {
	case EVENT_APP_INSTALLED:
		return eventsvcv26.EventType_EVENT_APP_INSTALLED
	case EVENT_APP_REMOVED:
		return eventsvcv26.EventType_EVENT_APP_REMOVED
	case EVENT_APP_UPDATED:
		return eventsvcv26.EventType_EVENT_APP_UPDATED
	case EVENT_APP_STARTED:
		return eventsvcv26.EventType_EVENT_APP_STARTED
	case EVENT_APP_STOPPED:
		return eventsvcv26.EventType_EVENT_APP_STOPPED
	case EVENT_APP_CRASHED:
		return eventsvcv26.EventType_EVENT_APP_CRASHED
	case EVENT_APP_REJECTED:
		return eventsvcv26.EventType_EVENT_APP_REJECTED
	case EVENT_SYSTEM_REBOOT:
		return eventsvcv26.EventType_EVENT_SYSTEM_REBOOT
	case EVENT_SYSTEM_FACTORY_RESET:
		return eventsvcv26.EventType_EVENT_SYSTEM_FACTORY_RESET
	case EVENT_SYSTEM_UPDATE:
		return eventsvcv26.EventType_EVENT_SYSTEM_UPDATE
	case EVENT_NET_UP:
		return eventsvcv26.EventType_EVENT_NET_UP
	case EVENT_NET_DOWN:
		return eventsvcv26.EventType_EVENT_NET_DOWN
	default:
		return eventsvcv26.EventType_EVENT_TYPE_UNKNOWN
	}
}

type EventManager struct {
	client eventsvcv26.EventServiceClient
	ctx    context.Context
}

func NewEventManager(client eventsvcv26.EventServiceClient, ctx context.Context) *EventManager {
	return &EventManager{client: client, ctx: ctx}
}

// Subscribe opens a server-side stream and calls handler for each received event.
// Pass no types to receive all events.
// Blocks until the context is cancelled, the server closes the stream, or an error occurs.
func (e *EventManager) Subscribe(ctx context.Context, handler func(*eventsvcv26.Event), types ...EventType) error {
	grpcTypes := make([]eventsvcv26.EventType, len(types))
	for i, t := range types {
		grpcTypes[i] = toGrpcEventType(t)
	}
	req := &eventsvcv26.SubscribeRequest{Types: grpcTypes}
	stream, err := e.client.Subscribe(ctx, req)
	if err != nil {
		return err
	}
	for {
		ev, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		handler(ev)
	}
}
