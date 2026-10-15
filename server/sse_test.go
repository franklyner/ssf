package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBeginSSEAndSend(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx := newSSETestContext(httptest.NewRequest(http.MethodGet, "/events", nil), recorder)

	stream, err := ctx.BeginSSE()
	if err != nil {
		t.Fatalf("BeginSSE returned an error: %v", err)
	}

	if !ctx.IsResponseSent {
		t.Fatal("expected the response to be marked as sent")
	}
	if ctx.ResponseCode != http.StatusOK {
		t.Fatalf("expected response code 200, got %d", ctx.ResponseCode)
	}
	if recorder.Header().Get("Content-Type") != SSEContentType {
		t.Fatalf("unexpected content type: %q", recorder.Header().Get("Content-Type"))
	}
	if recorder.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("unexpected cache control: %q", recorder.Header().Get("Cache-Control"))
	}
	if recorder.Header().Get("X-Accel-Buffering") != "no" {
		t.Fatalf("unexpected proxy buffering header: %q", recorder.Header().Get("X-Accel-Buffering"))
	}
	if !recorder.Flushed {
		t.Fatal("expected BeginSSE to flush the response headers")
	}

	err = stream.Send(SSEEvent{
		Event: "message",
		ID:    "event-1",
		Data:  "first line\r\nsecond line",
		Retry: 1500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Send returned an error: %v", err)
	}

	expected := "event: message\nid: event-1\nretry: 1500\ndata: first line\ndata: second line\n\n"
	if recorder.Body.String() != expected {
		t.Fatalf("unexpected SSE frame:\n%q\nwant:\n%q", recorder.Body.String(), expected)
	}
}

func TestSSESendJSONAndComment(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx := newSSETestContext(httptest.NewRequest(http.MethodGet, "/events", nil), recorder)
	stream, err := ctx.BeginSSE()
	if err != nil {
		t.Fatalf("BeginSSE returned an error: %v", err)
	}

	if err := stream.SendJSON("state", "run-1", struct {
		Status string `json:"status"`
	}{Status: "PLANNING"}); err != nil {
		t.Fatalf("SendJSON returned an error: %v", err)
	}
	if err := stream.SendComment("heartbeat\nstill-alive"); err != nil {
		t.Fatalf("SendComment returned an error: %v", err)
	}

	expected := "event: state\nid: run-1\ndata: {\"status\":\"PLANNING\"}\n\n: heartbeat\n: still-alive\n\n"
	if recorder.Body.String() != expected {
		t.Fatalf("unexpected SSE output:\n%q\nwant:\n%q", recorder.Body.String(), expected)
	}
}

func TestSSERejectsInvalidFields(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx := newSSETestContext(httptest.NewRequest(http.MethodGet, "/events", nil), recorder)
	stream, err := ctx.BeginSSE()
	if err != nil {
		t.Fatalf("BeginSSE returned an error: %v", err)
	}

	if err := stream.Send(SSEEvent{Event: "message\ninjected", Data: "ignored"}); err == nil {
		t.Fatal("expected an event name containing a newline to fail")
	}
	if err := stream.Send(SSEEvent{ID: "one\rtwo", Data: "ignored"}); err == nil {
		t.Fatal("expected an id containing a newline to fail")
	}
	if err := stream.Send(SSEEvent{Retry: -time.Second}); err == nil {
		t.Fatal("expected a negative retry duration to fail")
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("invalid events must not write data, got %q", recorder.Body.String())
	}
}

func TestBeginSSERejectsInvalidResponseState(t *testing.T) {
	t.Run("response already sent", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx := newSSETestContext(httptest.NewRequest(http.MethodGet, "/events", nil), recorder)
		ctx.IsResponseSent = true

		_, err := ctx.BeginSSE()
		if !errors.Is(err, ErrSSEResponseAlreadySent) {
			t.Fatalf("expected ErrSSEResponseAlreadySent, got %v", err)
		}
	})

	t.Run("writer cannot flush", func(t *testing.T) {
		writer := &nonFlushingResponseWriter{header: make(http.Header)}
		ctx := newSSETestContext(httptest.NewRequest(http.MethodGet, "/events", nil), writer)

		_, err := ctx.BeginSSE()
		if !errors.Is(err, ErrSSEStreamingUnsupported) {
			t.Fatalf("expected ErrSSEStreamingUnsupported, got %v", err)
		}
		if ctx.IsResponseSent {
			t.Fatal("unsupported streaming must not mark the response as sent")
		}
		if writer.status != 0 {
			t.Fatalf("unsupported streaming must not commit a status, got %d", writer.status)
		}
	})
}

func TestSSEStopsAfterRequestCancellation(t *testing.T) {
	requestContext, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(requestContext)
	recorder := httptest.NewRecorder()
	ctx := newSSETestContext(request, recorder)
	stream, err := ctx.BeginSSE()
	if err != nil {
		t.Fatalf("BeginSSE returned an error: %v", err)
	}

	cancel()
	err = stream.Send(SSEEvent{Data: "too late"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("cancelled stream must not write data, got %q", recorder.Body.String())
	}
}

func TestSSEClearsHTTPServerWriteDeadline(t *testing.T) {
	handlerDone := make(chan error, 1)
	testServer := httptest.NewUnstartedServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		ctx := newSSETestContext(request, response)
		stream, err := ctx.BeginSSE()
		if err == nil {
			time.Sleep(75 * time.Millisecond)
			err = stream.Send(SSEEvent{Data: "after deadline"})
		}
		handlerDone <- err
	}))
	testServer.Config.WriteTimeout = 20 * time.Millisecond
	testServer.Start()
	defer testServer.Close()

	response, err := testServer.Client().Get(testServer.URL)
	if err != nil {
		t.Fatalf("unable to open SSE request: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("unable to read SSE response: %v", err)
	}
	if string(body) != "data: after deadline\n\n" {
		t.Fatalf("unexpected response body: %q", string(body))
	}
	if err := <-handlerDone; err != nil {
		t.Fatalf("stream failed after the server write timeout: %v", err)
	}
}

func TestSSERequestContextCancelsWhenClientDisconnects(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	testServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		ctx := newSSETestContext(request, response)
		if _, err := ctx.BeginSSE(); err != nil {
			return
		}
		close(started)
		<-request.Context().Done()
		close(cancelled)
	}))
	defer testServer.Close()

	response, err := testServer.Client().Get(testServer.URL)
	if err != nil {
		t.Fatalf("unable to open SSE request: %v", err)
	}
	<-started
	if err := response.Body.Close(); err != nil {
		t.Fatalf("unable to close SSE response: %v", err)
	}

	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("request context was not cancelled after client disconnect")
	}
}

func TestSSEWritesAreAtomic(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx := newSSETestContext(httptest.NewRequest(http.MethodGet, "/events", nil), recorder)
	stream, err := ctx.BeginSSE()
	if err != nil {
		t.Fatalf("BeginSSE returned an error: %v", err)
	}

	done := make(chan error, 2)
	go func() {
		done <- stream.Send(SSEEvent{ID: "one", Data: strings.Repeat("a", 20)})
	}()
	go func() {
		done <- stream.Send(SSEEvent{ID: "two", Data: strings.Repeat("b", 20)})
	}()
	for range 2 {
		if err := <-done; err != nil {
			t.Fatalf("concurrent Send returned an error: %v", err)
		}
	}

	body := recorder.Body.String()
	frameOne := "id: one\ndata: " + strings.Repeat("a", 20) + "\n\n"
	frameTwo := "id: two\ndata: " + strings.Repeat("b", 20) + "\n\n"
	if body != frameOne+frameTwo && body != frameTwo+frameOne {
		t.Fatalf("concurrent frames were interleaved: %q", body)
	}
}

func newSSETestContext(request *http.Request, writer http.ResponseWriter) *Context {
	return &Context{
		Request:        request,
		responseWriter: writer,
	}
}

type nonFlushingResponseWriter struct {
	header http.Header
	status int
}

func (writer *nonFlushingResponseWriter) Header() http.Header {
	return writer.header
}

func (writer *nonFlushingResponseWriter) Write(body []byte) (int, error) {
	return len(body), nil
}

func (writer *nonFlushingResponseWriter) WriteHeader(status int) {
	writer.status = status
}
