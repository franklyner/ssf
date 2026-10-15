package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	SSEContentType = "text/event-stream"
)

var (
	ErrSSEResponseAlreadySent  = errors.New("response was already sent")
	ErrSSEStreamingUnsupported = errors.New("response writer does not support streaming")
)

// SSEEvent represents one Server-Sent Event.
type SSEEvent struct {
	Event string
	ID    string
	Data  string
	Retry time.Duration
}

// SSEWriter writes correctly framed Server-Sent Events to an HTTP response.
// It is safe for concurrent use. The request context controls its lifetime.
type SSEWriter struct {
	ctx        context.Context
	writer     http.ResponseWriter
	controller *http.ResponseController
	mu         sync.Mutex
}

// BeginSSE starts a Server-Sent Events response.
//
// It commits a 200 response immediately, disables the HTTP server write
// deadline when supported, and flushes the response headers. Callers should
// stop writing when the request context is cancelled or a write returns an
// error.
func (ctx *Context) BeginSSE() (*SSEWriter, error) {
	if ctx.IsResponseSent {
		return nil, ErrSSEResponseAlreadySent
	}

	writer := ctx.GetResponseWriter()
	if writer == nil {
		return nil, ErrSSEStreamingUnsupported
	}
	if !supportsSSEFlushing(writer) {
		return nil, ErrSSEStreamingUnsupported
	}

	controller := http.NewResponseController(writer)
	if err := controller.SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return nil, fmt.Errorf("unable to disable SSE write deadline: %w", err)
	}

	writer.Header().Set("Content-Type", SSEContentType)
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("X-Accel-Buffering", "no")
	writer.WriteHeader(http.StatusOK)

	ctx.ResponseCode = http.StatusOK
	ctx.IsResponseSent = true

	if err := controller.Flush(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSSEStreamingUnsupported, err)
	}

	return &SSEWriter{
		ctx:        ctx.Request.Context(),
		writer:     writer,
		controller: controller,
	}, nil
}

// Send writes and flushes one event.
func (writer *SSEWriter) Send(event SSEEvent) error {
	if err := validateSSEField("event", event.Event); err != nil {
		return err
	}
	if err := validateSSEField("id", event.ID); err != nil {
		return err
	}
	if event.Retry < 0 {
		return fmt.Errorf("SSE retry duration must not be negative")
	}

	var frame strings.Builder
	if event.Event != "" {
		fmt.Fprintf(&frame, "event: %s\n", event.Event)
	}
	if event.ID != "" {
		fmt.Fprintf(&frame, "id: %s\n", event.ID)
	}
	if event.Retry > 0 {
		fmt.Fprintf(&frame, "retry: %d\n", event.Retry.Milliseconds())
	}
	writeSSEData(&frame, event.Data)
	frame.WriteByte('\n')

	return writer.writeAndFlush(frame.String())
}

// SendJSON marshals data as JSON, then writes and flushes one event.
func (writer *SSEWriter) SendJSON(eventType string, id string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("unable to marshal SSE data: %w", err)
	}
	return writer.Send(SSEEvent{
		Event: eventType,
		ID:    id,
		Data:  string(payload),
	})
}

// SendComment writes and flushes an SSE comment. Comments are commonly used
// as heartbeats to keep idle connections open.
func (writer *SSEWriter) SendComment(comment string) error {
	normalized := normalizeSSELines(comment)
	var frame strings.Builder
	for _, line := range strings.Split(normalized, "\n") {
		frame.WriteString(":")
		if line != "" {
			frame.WriteByte(' ')
			frame.WriteString(line)
		}
		frame.WriteByte('\n')
	}
	frame.WriteByte('\n')
	return writer.writeAndFlush(frame.String())
}

func (writer *SSEWriter) writeAndFlush(frame string) error {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	if err := writer.ctx.Err(); err != nil {
		return err
	}
	if _, err := writer.writer.Write([]byte(frame)); err != nil {
		return fmt.Errorf("unable to write SSE frame: %w", err)
	}
	if err := writer.controller.Flush(); err != nil {
		return fmt.Errorf("unable to flush SSE frame: %w", err)
	}
	return nil
}

func validateSSEField(name string, value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("SSE %s must not contain newlines", name)
	}
	return nil
}

func writeSSEData(frame *strings.Builder, data string) {
	for _, line := range strings.Split(normalizeSSELines(data), "\n") {
		frame.WriteString("data:")
		if line != "" {
			frame.WriteByte(' ')
			frame.WriteString(line)
		}
		frame.WriteByte('\n')
	}
}

func normalizeSSELines(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(value, "\r", "\n")
}

func supportsSSEFlushing(writer http.ResponseWriter) bool {
	for {
		if _, ok := writer.(http.Flusher); ok {
			return true
		}
		unwrapper, ok := writer.(interface {
			Unwrap() http.ResponseWriter
		})
		if !ok {
			return false
		}
		writer = unwrapper.Unwrap()
		if writer == nil {
			return false
		}
	}
}
