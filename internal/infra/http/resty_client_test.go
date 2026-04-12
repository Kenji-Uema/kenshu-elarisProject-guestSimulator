package http

import (
	"testing"
	"time"
)

func TestNewRestyClientConfiguresBaseURLAndTimeout(t *testing.T) {
	client := NewRestyClient("example.test", 8080)

	if client.BaseURL != "http://example.test:8080" {
		t.Fatalf("unexpected base URL: %q", client.BaseURL)
	}
	if client.GetClient().Timeout != 5*time.Second {
		t.Fatalf("unexpected timeout: %s", client.GetClient().Timeout)
	}
	if client.GetClient().Transport == nil {
		t.Fatal("expected transport to be configured")
	}
}
