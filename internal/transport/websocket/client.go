package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const protocolVersion = "lodging.v1"

type Client struct {
	conn      *websocket.Conn
	writeMu   sync.Mutex
	inboundCh chan *domain.ChatMessage
	ackCh     chan *domain.ChatMessage
	errCh     chan error
	pending   []*domain.ChatMessage
}

func NewClient(ctx context.Context, url string) (*Client, error) {
	ctx, span := telemetry.Tracer.Start(ctx, "NewLodgingClient")
	defer span.End()

	headers := http.Header{}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(headers))

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, headers)
	if err != nil {
		return nil, err
	}

	client := &Client{
		conn:      conn,
		inboundCh: make(chan *domain.ChatMessage, 32),
		ackCh:     make(chan *domain.ChatMessage, 32),
		errCh:     make(chan error, 1),
	}
	go client.readLoop()

	return client, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) SendAction(ctx context.Context, action domain.GuestAction) error {
	return c.sendAndWaitAck(ctx, &domain.ChatMessage{
		MessageID:       uuid.NewString(),
		CorrelationID:   uuid.NewString(),
		Sender:          domain.SenderGuest,
		ProtocolVersion: protocolVersion,
		GuestAction:     action,
	})
}

func (c *Client) Reply(ctx context.Context, request *domain.ChatMessage, response *domain.GuestResponse) error {
	if request == nil {
		return fmt.Errorf("request is nil")
	}

	return c.sendAndWaitAck(ctx, &domain.ChatMessage{
		MessageID:       uuid.NewString(),
		CorrelationID:   request.CorrelationID,
		Sender:          domain.SenderGuest,
		ProtocolVersion: protocolVersion,
		GuestResponse:   response,
	})
}

func (c *Client) WaitForNextSystemMessage(ctx context.Context) (*domain.ChatMessage, error) {
	for {
		msg, err := c.nextMessage(ctx)
		if err != nil {
			return nil, err
		}

		if msg.SystemNotification != "" || msg.SystemRequest != "" {
			return msg, nil
		}

		slog.DebugContext(ctx, "ignoring unexpected websocket message", "message", msg)
	}
}

func (c *Client) readLoop() {
	for {
		_, payload, err := c.conn.ReadMessage()
		if err != nil {
			select {
			case c.errCh <- err:
			default:
			}
			return
		}

		var msg domain.ChatMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			select {
			case c.errCh <- err:
			default:
			}
			return
		}

		if msg.Ack != nil {
			c.ackCh <- &msg
			continue
		}

		c.sendAck(context.Background(), &msg)
		c.inboundCh <- &msg
	}
}

func (c *Client) sendAck(ctx context.Context, msg *domain.ChatMessage) {
	if msg == nil || msg.MessageID == "" {
		return
	}

	ack := &domain.ChatMessage{
		MessageID:       uuid.NewString(),
		CorrelationID:   msg.MessageID,
		Sender:          domain.SenderGuest,
		ProtocolVersion: protocolVersion,
		Ack: &domain.Ack{
			AcknowledgedMessageID: msg.MessageID,
			Status:                "ACK_STATUS_ACCEPTED",
			Code:                  "ERROR_CODE_NONE",
		},
	}

	if err := c.write(ctx, ack); err != nil {
		slog.WarnContext(ctx, "failed to send websocket ack", "err", err, "messageId", msg.MessageID)
	}
}

func (c *Client) sendAndWaitAck(ctx context.Context, msg *domain.ChatMessage) error {
	if err := c.write(ctx, msg); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-c.errCh:
			return err
		case ack := <-c.ackCh:
			if ack.Ack != nil && ack.Ack.AcknowledgedMessageID == msg.MessageID {
				return nil
			}
		case inbound := <-c.inboundCh:
			c.pending = append(c.pending, inbound)
		}
	}
}

func (c *Client) nextMessage(ctx context.Context) (*domain.ChatMessage, error) {
	if len(c.pending) > 0 {
		msg := c.pending[0]
		c.pending = c.pending[1:]
		return msg, nil
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-c.errCh:
		return nil, err
	case msg := <-c.inboundCh:
		return msg, nil
	}
}

func (c *Client) write(ctx context.Context, msg *domain.ChatMessage) error {
	if msg != nil {
		carrier := propagation.MapCarrier{}
		otel.GetTextMapPropagator().Inject(ctx, carrier)
		if len(carrier) > 0 {
			msg.TraceContext = map[string]string(carrier)
		} else {
			msg.TraceContext = nil
		}
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, payload)
}
