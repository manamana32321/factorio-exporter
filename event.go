package main

import "time"

// GameEvent represents any event from the Factorio server.
type GameEvent struct {
	Type    string            // "chat", "join", "leave", "research", "rocket", "save", "research_started", "player_died", etc.
	Player  string            // Player name (empty for non-player events)
	Message string            // Chat message content
	Extra   map[string]string // Event-specific data (tech, cause, surface, name, etc.)
	Time    time.Time
}

// InboundMessage represents a message from an external channel destined for Factorio.
type InboundMessage struct {
	Source  string // Channel name (e.g., "Discord")
	Author  string
	Content string
}

// LogSubscriber receives parsed events from the LogTailer or EventPoller.
type LogSubscriber interface {
	OnLogEvent(event GameEvent)
}
