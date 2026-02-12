package main

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// BridgeSubscriber forwards GameEvents to the Bridge's event channel.
type BridgeSubscriber struct {
	events chan<- GameEvent
}

func (s *BridgeSubscriber) OnLogEvent(event GameEvent) {
	select {
	case s.events <- event:
	default:
		// Drop event if channel is full (avoid blocking log tailer)
	}
}

// Bridge fans out GameEvents to all channels and handles inbound messages.
type Bridge struct {
	rcon     *RCONPool
	channels []Channel
	events   chan GameEvent
}

func NewBridge(pool *RCONPool, channels []Channel) *Bridge {
	return &Bridge{
		rcon:     pool,
		channels: channels,
		events:   make(chan GameEvent, 100),
	}
}

// Events returns the event channel for subscribers to write to.
func (b *Bridge) Events() chan<- GameEvent {
	return b.events
}

// FanOutEvents reads events and sends them to all channels.
func (b *Bridge) FanOutEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-b.events:
			for _, ch := range b.channels {
				if err := ch.Send(ctx, event); err != nil {
					log.Printf("send to %s: %v", ch.Name(), err)
				}
			}
		}
	}
}

// HandleInbound reads messages from a channel and sends them to Factorio.
func (b *Bridge) HandleInbound(ctx context.Context, ch Channel) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch.Messages():
			b.sendToFactorio(msg)
		}
	}
}

func (b *Bridge) sendToFactorio(msg InboundMessage) {
	safe := msg.Content
	safe = strings.ReplaceAll(safe, `\`, `\\`)
	safe = strings.ReplaceAll(safe, `"`, `\"`)
	safe = strings.ReplaceAll(safe, "\n", " ")

	if len(safe) > 200 {
		safe = safe[:200] + "..."
	}

	cmd := fmt.Sprintf(`/sc game.print("[color=purple][%s][/color] %s: %s")`,
		msg.Source, msg.Author, safe)

	if _, err := b.rcon.Execute(cmd); err != nil {
		log.Printf("rcon send to factorio: %v", err)
	}
}
