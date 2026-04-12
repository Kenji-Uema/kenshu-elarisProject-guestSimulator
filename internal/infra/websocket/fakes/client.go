package fakes

import (
	"context"

	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/lodging"
)

type Client struct {
	Messages    []*lodging.ChatMessage
	WaitErr     error
	ReplyErr    error
	SendErr     error
	CloseErr    error
	Replies     []*lodging.GuestResponse
	ReplyTo     []*lodging.ChatMessage
	SendActions []lodging.GuestAction
}

func (c *Client) Close() error { return c.CloseErr }

func (c *Client) SendAction(_ context.Context, action lodging.GuestAction) error {
	if c.SendErr != nil {
		return c.SendErr
	}
	c.SendActions = append(c.SendActions, action)
	return nil
}

func (c *Client) Reply(_ context.Context, request *lodging.ChatMessage, response *lodging.GuestResponse) error {
	if c.ReplyErr != nil {
		return c.ReplyErr
	}
	c.ReplyTo = append(c.ReplyTo, request)
	c.Replies = append(c.Replies, response)
	return nil
}

func (c *Client) WaitForNextSystemMessage(context.Context) (*lodging.ChatMessage, error) {
	if c.WaitErr != nil {
		return nil, c.WaitErr
	}
	if len(c.Messages) == 0 {
		return nil, context.Canceled
	}
	msg := c.Messages[0]
	c.Messages = c.Messages[1:]
	return msg, nil
}
