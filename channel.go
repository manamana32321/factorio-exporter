package main

import "context"

// Channel abstracts an external chat platform (Discord, Slack, Telegram, etc.).
type Channel interface {
	Name() string
	Send(ctx context.Context, event GameEvent) error
	Messages() <-chan InboundMessage
	Start(ctx context.Context) error
	Close() error
}
