package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/kristofferrisa/powerctl-cli/internal/models"
)

const (
	// UserAgent identifies this CLI to the Tibber API
	UserAgent = "powerctl-cli/1.0"
)

const (
	// WebSocketEndpoint is the Tibber WebSocket API URL
	WebSocketEndpoint = "wss://websocket-api.tibber.com/v1-beta/gql/subscriptions"
)

// LiveClient handles WebSocket connections for real-time data
type LiveClient struct {
	token  string
	homeID string
	// endpoint is WebSocketEndpoint unless a test points it at a local server.
	endpoint string
}

// NewLiveClient creates a new WebSocket client for live data
func NewLiveClient(token, homeID string) *LiveClient {
	return &LiveClient{
		token:    token,
		homeID:   homeID,
		endpoint: WebSocketEndpoint,
	}
}

// wsMessage represents a WebSocket protocol message
type wsMessage struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Subscribe connects to the live measurement stream
func (c *LiveClient) Subscribe(ctx context.Context, handler func(*models.LiveMeasurement) error) error {
	// Connect with subprotocol and proper headers
	headers := http.Header{}
	headers.Set("User-Agent", UserAgent)

	conn, resp, err := websocket.Dial(ctx, c.endpoint, &websocket.DialOptions{
		Subprotocols: []string{"graphql-transport-ws"},
		HTTPHeader:   headers,
	})
	if err != nil {
		// A refused handshake is a credentials or endpoint problem rather than a
		// blip, and would be refused identically on every reconnect.
		if resp != nil && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) {
			return markFatal(fmt.Errorf("failed to connect: %w", err))
		}
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		if ctx.Err() != nil {
			// The caller is already shutting down. A close handshake would make
			// Ctrl+C wait up to five seconds for a reply the server may never
			// send, so drop the socket instead.
			_ = conn.CloseNow()
			return
		}
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	// Send connection_init
	initPayload, _ := json.Marshal(map[string]string{
		"token": c.token,
	})
	initMsg := wsMessage{
		Type:    "connection_init",
		Payload: initPayload,
	}
	if err := c.sendMessage(ctx, conn, initMsg); err != nil {
		return fmt.Errorf("failed to send init: %w", err)
	}

	// Wait for connection_ack
	if err := c.waitForAck(ctx, conn); err != nil {
		return err
	}

	// Send subscription
	subPayload, _ := json.Marshal(map[string]interface{}{
		"query": SubscriptionLiveMeasurement,
		"variables": map[string]string{
			"homeId": c.homeID,
		},
	})
	subMsg := wsMessage{
		Type:    "subscribe",
		ID:      "1",
		Payload: subPayload,
	}
	if err := c.sendMessage(ctx, conn, subMsg); err != nil {
		return fmt.Errorf("failed to send subscription: %w", err)
	}

	// Read messages
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			_, data, err := conn.Read(ctx)
			if err != nil {
				return fmt.Errorf("read error: %w", err)
			}

			var msg wsMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "next":
				measurement, err := c.parsePayload(msg.Payload)
				if err != nil {
					// The subscription itself is wrong (bad home ID, GraphQL
					// error): a fresh connection gets the same payload back.
					return markFatal(err)
				}
				if err := handler(measurement); err != nil {
					return markFatal(err)
				}
			case "error":
				return markFatal(fmt.Errorf("subscription error: %s", string(msg.Payload)))
			case "complete":
				return nil
			}
		}
	}
}

func (c *LiveClient) sendMessage(ctx context.Context, conn *websocket.Conn, msg wsMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, data)
}

func (c *LiveClient) waitForAck(ctx context.Context, conn *websocket.Conn) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, data, err := conn.Read(ctx)
	if err != nil {
		return fmt.Errorf("failed to read ack: %w", err)
	}

	var msg wsMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to parse ack: %w", err)
	}

	if msg.Type == "connection_error" {
		return markFatal(fmt.Errorf("connection rejected: %s", string(msg.Payload)))
	}

	if msg.Type != "connection_ack" {
		return fmt.Errorf("expected connection_ack, got %s", msg.Type)
	}

	return nil
}

func (c *LiveClient) parsePayload(payload json.RawMessage) (*models.LiveMeasurement, error) {
	var data struct {
		Data struct {
			LiveMeasurement *models.LiveMeasurement `json:"liveMeasurement"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}

	if len(data.Errors) > 0 {
		msg := data.Errors[0].Message
		if msg == "user not authorized to access home" || msg == "home not found" {
			return nil, errors.New("invalid or non-existing home ID")
		}
		return nil, errors.New(msg)
	}

	if data.Data.LiveMeasurement == nil {
		return nil, errors.New("no measurement data received (invalid home ID?)")
	}

	return data.Data.LiveMeasurement, nil
}
