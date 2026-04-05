package websocket

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/lodging"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/encoding/protojson"
)

const protocolVersion = "lodging.v1"

type ClientFactory struct{}

type Client struct {
	conn      *websocket.Conn
	writeMu   sync.Mutex
	inboundCh chan *lodging.ChatMessage
	ackCh     chan *lodging.ChatMessage
	errCh     chan error
	pending   []*lodging.ChatMessage
}

func (ClientFactory) NewClient(ctx context.Context, url string) (port.LodgingChatClient, error) {
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
		inboundCh: make(chan *lodging.ChatMessage, 32),
		ackCh:     make(chan *lodging.ChatMessage, 32),
		errCh:     make(chan error, 1),
	}
	go client.readLoop()

	return client, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) SendAction(ctx context.Context, action lodging.GuestAction) error {
	return c.sendAndWaitAck(ctx, &lodging.ChatMessage{
		MessageId:       uuid.NewString(),
		CorrelationId:   uuid.NewString(),
		Sender:          lodging.Sender_SENDER_GUEST,
		ProtocolVersion: protocolVersion,
		Payload:         &lodging.ChatMessage_GuestAction{GuestAction: action},
	})
}

func (c *Client) Reply(ctx context.Context, request *lodging.ChatMessage, response *lodging.GuestResponse) error {
	if request == nil {
		return fmt.Errorf("request is nil")
	}

	return c.sendAndWaitAck(ctx, &lodging.ChatMessage{
		MessageId:       uuid.NewString(),
		CorrelationId:   request.GetCorrelationId(),
		Sender:          lodging.Sender_SENDER_GUEST,
		ProtocolVersion: protocolVersion,
		Payload:         &lodging.ChatMessage_GuestResponse{GuestResponse: response},
	})
}

func (c *Client) WaitForNextSystemMessage(ctx context.Context) (*lodging.ChatMessage, error) {
	for {
		msg, err := c.nextMessage(ctx)
		if err != nil {
			return nil, err
		}

		if msg.GetSystemNotification() != lodging.SystemNotification_SYSTEM_NOTIFICATION_UNSPECIFIED ||
			msg.GetSystemRequest() != lodging.SystemRequest_SYSTEM_REQUEST_UNSPECIFIED {
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

		var msg lodging.ChatMessage
		if err := protojson.Unmarshal(payload, &msg); err != nil {
			select {
			case c.errCh <- err:
			default:
			}
			return
		}

		if msg.GetAck() != nil {
			c.ackCh <- &msg
			continue
		}

		c.sendAck(context.Background(), &msg)
		c.inboundCh <- &msg
	}
}

func (c *Client) sendAck(ctx context.Context, msg *lodging.ChatMessage) {
	if msg == nil || msg.GetMessageId() == "" {
		return
	}

	ack := &lodging.ChatMessage{
		MessageId:       uuid.NewString(),
		CorrelationId:   msg.GetMessageId(),
		Sender:          lodging.Sender_SENDER_GUEST,
		ProtocolVersion: protocolVersion,
		Payload: &lodging.ChatMessage_Ack{Ack: &lodging.Ack{
			AcknowledgedMessageId: msg.GetMessageId(),
			Status:                lodging.AckStatus_ACK_STATUS_ACCEPTED,
			Code:                  lodging.ErrorCode_ERROR_CODE_NONE,
		}},
	}

	if err := c.write(ctx, ack); err != nil {
		slog.WarnContext(ctx, "failed to send websocket ack", "err", err, "messageId", msg.GetMessageId())
	}
}

func (c *Client) sendAndWaitAck(ctx context.Context, msg *lodging.ChatMessage) error {
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
			if ack.GetAck() != nil && ack.GetAck().GetAcknowledgedMessageId() == msg.GetMessageId() {
				return nil
			}
		case inbound := <-c.inboundCh:
			c.pending = append(c.pending, inbound)
		}
	}
}

func (c *Client) nextMessage(ctx context.Context) (*lodging.ChatMessage, error) {
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

func (c *Client) write(ctx context.Context, msg *lodging.ChatMessage) error {
	if msg != nil {
		carrier := propagation.MapCarrier{}
		otel.GetTextMapPropagator().Inject(ctx, carrier)
		if len(carrier) > 0 {
			msg.TraceContext = carrier
		} else {
			msg.TraceContext = nil
		}
	}

	payload, err := protojson.Marshal(msg)
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
