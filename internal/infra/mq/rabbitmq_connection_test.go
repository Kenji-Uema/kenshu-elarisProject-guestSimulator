package mq

import (
	"strings"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
)

func TestRabbitMqConnectionReportsClosedWithoutUnderlyingConnection(t *testing.T) {
	if (&RabbitMqConnection{}).IsConnectionOpen() {
		t.Fatal("expected connection to be closed")
	}
}

func TestRabbitMqConnectionCloseIsNilSafe(t *testing.T) {
	conn := &RabbitMqConnection{}

	if err := conn.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
	if !conn.closed {
		t.Fatal("expected connection to be marked closed")
	}
}

func TestRabbitMqConnectionOpenConnectionRejectsClosedState(t *testing.T) {
	conn := &RabbitMqConnection{closed: true}

	opened, err := conn.openConnection()
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if opened != nil {
		t.Fatalf("expected no connection, got %#v", opened)
	}
}

func TestNewRabbitMqChannelKeepsConnectionReference(t *testing.T) {
	conn := &RabbitMqConnection{}
	channel := NewRabbitMqChannel(conn)

	if channel.RabbitMqConnection != conn {
		t.Fatalf("unexpected channel connection: %#v", channel.RabbitMqConnection)
	}
}

func TestRabbitMqChannelCloseChannelIsNilSafe(t *testing.T) {
	channel := &RabbitMqChannel{}
	if err := channel.CloseChannel(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestNewRabbitmqConsumerReturnsErrorWhenConnectionIsClosed(t *testing.T) {
	consumer, err := NewRabbitmqConsumer(&RabbitMqConnection{closed: true}, config.ConsumeConfig{})
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if consumer != nil {
		t.Fatalf("expected nil consumer, got %#v", consumer)
	}
}
