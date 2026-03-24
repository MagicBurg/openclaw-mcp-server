package openclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"nhooyr.io/websocket"
)

// WSClient connects to an OpenClaw gateway via WebSocket for RPC calls.
type WSClient struct {
	url   string
	token string

	conn      *websocket.Conn
	mu        sync.Mutex
	reqID     atomic.Int64
	pending   map[string]chan json.RawMessage
	pendingMu sync.Mutex
}

// NewWSClient creates a new WebSocket client (not yet connected).
func NewWSClient(baseURL, token string) *WSClient {
	// Convert http(s) to ws(s)
	wsURL := baseURL
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	wsURL = strings.TrimRight(wsURL, "/")

	return &WSClient{
		url:     wsURL,
		token:   token,
		pending: make(map[string]chan json.RawMessage),
	}
}

// wsFrame represents the JSON framing used by the OpenClaw gateway.
type wsFrame struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  any             `json:"params,omitempty"`
	Event   string          `json:"event,omitempty"`
	OK      bool            `json:"ok,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   *wsError        `json:"error,omitempty"`
}

type wsError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Connect establishes the WebSocket connection and performs the handshake.
func (c *WSClient) Connect(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, c.url, nil)
	if err != nil {
		return fmt.Errorf("ws dial: %w", err)
	}
	conn.SetReadLimit(16 * 1024 * 1024)
	c.conn = conn

	// Wait for connect.challenge event.
	var challenge wsFrame
	if err := c.readFrame(ctx, &challenge); err != nil {
		conn.Close(websocket.StatusNormalClosure, "")
		return fmt.Errorf("read challenge: %w", err)
	}
	if challenge.Event != "connect.challenge" {
		conn.Close(websocket.StatusNormalClosure, "")
		return fmt.Errorf("expected connect.challenge, got %q", challenge.Event)
	}

	// Send connect request.
	connectID := c.nextID()
	connectReq := wsFrame{
		Type:   "req",
		ID:     connectID,
		Method: "connect",
		Params: map[string]any{
			"minProtocol": 3,
			"maxProtocol": 3,
			"client": map[string]any{
				"id":          "gateway-client",
				"displayName": "OpenClaw MCP Server",
				"version":     "0.1.0",
				"platform":    "linux",
				"mode":        "backend",
			},
			"auth": map[string]any{
				"token": c.token,
			},
		},
	}
	if err := c.writeFrame(ctx, connectReq); err != nil {
		conn.Close(websocket.StatusNormalClosure, "")
		return fmt.Errorf("send connect: %w", err)
	}

	// Read until we get the connect response.
	for {
		var resp wsFrame
		if err := c.readFrame(ctx, &resp); err != nil {
			conn.Close(websocket.StatusNormalClosure, "")
			return fmt.Errorf("read connect response: %w", err)
		}
		if resp.Type == "event" {
			continue // skip events during handshake
		}
		if resp.Type == "res" && resp.ID == connectID {
			if !resp.OK {
				conn.Close(websocket.StatusNormalClosure, "")
				msg := "connect failed"
				if resp.Error != nil {
					msg = resp.Error.Message
				}
				return fmt.Errorf("connect rejected: %s", msg)
			}
			break
		}
	}

	// Start background reader to dispatch responses.
	go c.readLoop(ctx)

	return nil
}

// Call sends an RPC request and waits for the response.
func (c *WSClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.nextID()
	ch := make(chan json.RawMessage, 1)

	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

	req := wsFrame{
		Type:   "req",
		ID:     id,
		Method: method,
		Params: params,
	}
	if err := c.writeFrame(ctx, req); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case raw := <-ch:
		// raw is the full response frame
		var resp wsFrame
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}
		if !resp.OK {
			msg := "rpc error"
			if resp.Error != nil {
				msg = resp.Error.Message
			}
			return nil, fmt.Errorf("%s: %s", method, msg)
		}
		return resp.Payload, nil
	}
}

// Close closes the WebSocket connection.
func (c *WSClient) Close() error {
	if c.conn != nil {
		return c.conn.Close(websocket.StatusNormalClosure, "bye")
	}
	return nil
}

func (c *WSClient) readLoop(ctx context.Context) {
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return
		}

		var frame wsFrame
		if err := json.Unmarshal(data, &frame); err != nil {
			continue
		}

		if frame.Type == "res" && frame.ID != "" {
			c.pendingMu.Lock()
			ch, ok := c.pending[frame.ID]
			c.pendingMu.Unlock()
			if ok {
				ch <- data
			}
		}
		// Events are silently dropped for now.
	}
}

func (c *WSClient) readFrame(ctx context.Context, frame *wsFrame) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, data, err := c.conn.Read(ctx)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, frame)
}

func (c *WSClient) writeFrame(ctx context.Context, frame wsFrame) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	return c.conn.Write(ctx, websocket.MessageText, data)
}

func (c *WSClient) nextID() string {
	return fmt.Sprintf("mcp-%d", c.reqID.Add(1))
}

// ToolsCatalog calls tools.catalog via WebSocket RPC.
// Returns the raw payload or an error.
func (c *WSClient) ToolsCatalog(ctx context.Context) (*ToolsCatalogResult, error) {
	payload, err := c.Call(ctx, "tools.catalog", map[string]any{"includePlugins": true})
	if err != nil {
		return nil, err
	}

	var result ToolsCatalogResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("parse tools.catalog: %w", err)
	}
	return &result, nil
}

// SkillsStatus calls skills.status via WebSocket RPC.
func (c *WSClient) SkillsStatus(ctx context.Context) (*SkillsStatusResult, error) {
	payload, err := c.Call(ctx, "skills.status", map[string]any{})
	if err != nil {
		return nil, err
	}

	var result SkillsStatusResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("parse skills.status: %w", err)
	}
	return &result, nil
}
