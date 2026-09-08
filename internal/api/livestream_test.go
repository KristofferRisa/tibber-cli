package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/kristofferrisa/powerctl-cli/internal/models"
)

// liveServer is a minimal graphql-transport-ws server standing in for Tibber, so
// the reconnect path can be exercised over a real socket rather than a stub.
type liveServer struct {
	*httptest.Server

	mu    sync.Mutex
	conns int
}

// startLiveServer accepts WebSocket connections and hands each one to session,
// already past connection_init, with the 1-based connection number.
func startLiveServer(t *testing.T, session func(ctx context.Context, conn *websocket.Conn, n int)) *liveServer {
	t.Helper()

	s := &liveServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols: []string{"graphql-transport-ws"},
		})
		if err != nil {
			return
		}
		defer conn.CloseNow()

		s.mu.Lock()
		s.conns++
		n := s.conns
		s.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if _, _, err := conn.Read(ctx); err != nil { // connection_init
			return
		}
		session(ctx, conn, n)
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *liveServer) connections() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conns
}

// client returns a LiveClient pointed at this server.
func (s *liveServer) client() *LiveClient {
	c := NewLiveClient("test-token", "test-home")
	c.endpoint = "ws" + strings.TrimPrefix(s.URL, "http")
	return c
}

func send(ctx context.Context, conn *websocket.Conn, msgType string, payload string) error {
	data, err := json.Marshal(wsMessage{Type: msgType, ID: "1", Payload: json.RawMessage(payload)})
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, data)
}

func ackAndAwaitSubscribe(ctx context.Context, conn *websocket.Conn) error {
	if err := send(ctx, conn, "connection_ack", ""); err != nil {
		return err
	}
	_, _, err := conn.Read(ctx) // subscribe
	return err
}

const measurementPayload = `{"data":{"liveMeasurement":{"timestamp":"2026-01-01T12:00:00Z","power":1234}}}`

// fastPolicy retries almost immediately so the tests do not wait on real backoff.
func fastPolicy() ReconnectPolicy {
	p := DefaultReconnectPolicy()
	p.BaseDelay = time.Millisecond
	p.MaxDelay = time.Millisecond
	return p
}

func TestSubscribeWithReconnect_RecoversFromDroppedConnection(t *testing.T) {
	server := startLiveServer(t, func(ctx context.Context, conn *websocket.Conn, n int) {
		if err := ackAndAwaitSubscribe(ctx, conn); err != nil {
			return
		}
		if err := send(ctx, conn, "next", measurementPayload); err != nil {
			return
		}
		if n == 1 {
			conn.CloseNow() // the blip: no close handshake, just a dead socket
			return
		}
		_ = send(ctx, conn, "complete", "")
	})

	var measurements []*models.LiveMeasurement
	err := server.client().SubscribeWithReconnect(context.Background(), fastPolicy(),
		func(m *models.LiveMeasurement) error {
			measurements = append(measurements, m)
			return nil
		})

	if err != nil {
		t.Fatalf("expected the stream to recover, got %v", err)
	}
	if server.connections() != 2 {
		t.Errorf("expected 2 connections, got %d", server.connections())
	}
	if len(measurements) != 2 {
		t.Fatalf("expected 2 measurements across the reconnect, got %d", len(measurements))
	}
	for i, m := range measurements {
		if m == nil || m.Power != 1234 {
			t.Errorf("measurement %d: expected 1234 W, got %v", i, m)
		}
	}
}

func TestSubscribeWithReconnect_DoesNotRetryFatalErrors(t *testing.T) {
	tests := []struct {
		name    string
		session func(ctx context.Context, conn *websocket.Conn, n int)
		wantErr string
	}{
		{
			name: "rejected token",
			session: func(ctx context.Context, conn *websocket.Conn, _ int) {
				_ = send(ctx, conn, "connection_error", `{"message":"Invalid token"}`)
			},
			wantErr: "connection rejected",
		},
		{
			name: "token refused with close code 4403",
			session: func(_ context.Context, conn *websocket.Conn, _ int) {
				_ = conn.Close(statusInvalidToken, "Invalid token")
			},
			wantErr: "4403",
		},
		{
			name: "unknown home ID",
			session: func(ctx context.Context, conn *websocket.Conn, _ int) {
				if err := ackAndAwaitSubscribe(ctx, conn); err != nil {
					return
				}
				_ = send(ctx, conn, "next", `{"errors":[{"message":"home not found"}],"data":{"liveMeasurement":null}}`)
			},
			wantErr: "invalid or non-existing home ID",
		},
		{
			name: "subscription error",
			session: func(ctx context.Context, conn *websocket.Conn, _ int) {
				if err := ackAndAwaitSubscribe(ctx, conn); err != nil {
					return
				}
				_ = send(ctx, conn, "error", `[{"message":"query is not valid"}]`)
			},
			wantErr: "subscription error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := startLiveServer(t, tt.session)

			err := server.client().SubscribeWithReconnect(context.Background(), fastPolicy(),
				func(*models.LiveMeasurement) error { return nil })

			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected an error containing %q, got %q", tt.wantErr, err)
			}
			if errors.Is(err, ErrReconnectBudget) {
				t.Error("retried a fatal error until the connection budget ran out")
			}
			if server.connections() != 1 {
				t.Errorf("expected 1 connection attempt, got %d", server.connections())
			}
		})
	}
}

func TestSubscribeWithReconnect_StopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := startLiveServer(t, func(sctx context.Context, conn *websocket.Conn, _ int) {
		if err := ackAndAwaitSubscribe(sctx, conn); err != nil {
			return
		}
		for {
			if err := send(sctx, conn, "next", measurementPayload); err != nil {
				return
			}
			time.Sleep(time.Millisecond)
		}
	})

	received := 0
	start := time.Now()
	err := server.client().SubscribeWithReconnect(ctx, fastPolicy(),
		func(*models.LiveMeasurement) error {
			received++
			if received == 3 {
				cancel() // stand-in for Ctrl+C
			}
			return nil
		})

	// Ctrl+C must not wait on a close handshake the server may never answer.
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("expected a prompt exit on cancellation, took %v", elapsed)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if server.connections() != 1 {
		t.Errorf("expected no reconnect after cancellation, got %d connections", server.connections())
	}
}

func TestSubscribe_NoReconnectStopsOnFirstError(t *testing.T) {
	server := startLiveServer(t, func(ctx context.Context, conn *websocket.Conn, _ int) {
		if err := ackAndAwaitSubscribe(ctx, conn); err != nil {
			return
		}
		if err := send(ctx, conn, "next", measurementPayload); err != nil {
			return
		}
		conn.CloseNow()
	})

	// What `powerctl live --no-reconnect` does: one connection, then out.
	err := server.client().Subscribe(context.Background(), func(*models.LiveMeasurement) error { return nil })

	if err == nil {
		t.Fatal("expected a read error, got nil")
	}
	if server.connections() != 1 {
		t.Errorf("expected 1 connection, got %d", server.connections())
	}
}
