package port

import (
	"context"

	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/lodging"
)

type LodgingChatClient interface {
	Close() error
	SendAction(ctx context.Context, action lodging.GuestAction) error
	Reply(ctx context.Context, request *lodging.ChatMessage, response *lodging.GuestResponse) error
	WaitForNextSystemMessage(ctx context.Context) (*lodging.ChatMessage, error)
}

type LodgingChatClientFactory interface {
	NewClient(ctx context.Context, url string) (LodgingChatClient, error)
}
