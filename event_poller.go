package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"
)

// RCONEvent represents a single event from the Lua event queue.
type RCONEvent struct {
	Type    string `json:"type"`
	Name    string `json:"name,omitempty"`
	Player  string `json:"player,omitempty"`
	Cause   string `json:"cause,omitempty"`
	Surface string `json:"surface,omitempty"`
	State   string `json:"state,omitempty"`
	Text    string `json:"text,omitempty"`
	Tick    int64  `json:"tick"`
}

// EventPoller registers Lua event handlers via RCON and polls the event queue.
type EventPoller struct {
	rcon            *RCONPool
	registerScripts []string
	pollLua         string
	pollInterval    time.Duration
	subscribers     []LogSubscriber
	registered      bool
}

func NewEventPoller(pool *RCONPool, registerScripts []string, pollLua string, interval time.Duration) *EventPoller {
	return &EventPoller{
		rcon:            pool,
		registerScripts: registerScripts,
		pollLua:         pollLua,
		pollInterval:    interval,
	}
}

func (p *EventPoller) Subscribe(sub LogSubscriber) {
	p.subscribers = append(p.subscribers, sub)
}

func (p *EventPoller) Run(ctx context.Context) {
	p.registerWithRetry(ctx)

	pollTicker := time.NewTicker(p.pollInterval)
	defer pollTicker.Stop()

	healthTicker := time.NewTicker(60 * time.Second)
	defer healthTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			p.poll()
		case <-healthTicker.C:
			p.healthCheck(ctx)
		}
	}
}

func (p *EventPoller) registerWithRetry(ctx context.Context) {
	for {
		if p.executeScripts() {
			p.registered = true
			log.Println("RCON event handlers registered")
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(15 * time.Second):
		}
	}
}

func (p *EventPoller) executeScripts() bool {
	for i, script := range p.registerScripts {
		resp, err := p.rcon.Execute("/sc " + script)
		if err != nil || strings.TrimSpace(resp) != "ok" {
			log.Printf("event registration failed at script %d (err=%v, resp=%s), retrying in 15s", i+1, err, resp)
			return false
		}
	}
	return true
}

func (p *EventPoller) poll() {
	resp, err := p.rcon.Execute("/sc " + p.pollLua)
	if err != nil {
		log.Printf("event poll error: %v", err)
		p.registered = false
		return
	}

	resp = strings.TrimSpace(resp)
	if resp == "" || resp == "[]" {
		return
	}

	var events []RCONEvent
	if err := json.Unmarshal([]byte(resp), &events); err != nil {
		log.Printf("event poll parse error: %v (resp=%.200s)", err, resp)
		return
	}

	for _, e := range events {
		ge := e.toGameEvent()
		for _, sub := range p.subscribers {
			sub.OnLogEvent(ge)
		}
	}
}

func (p *EventPoller) healthCheck(ctx context.Context) {
	if !p.registered {
		p.registerWithRetry(ctx)
		return
	}
	resp, err := p.rcon.Execute(`/sc rcon.print(storage.bridge_events ~= nil and "ok" or "missing")`)
	if err != nil || strings.TrimSpace(resp) != "ok" {
		log.Println("event handlers missing, re-registering...")
		p.registered = false
		p.registerWithRetry(ctx)
	}
}

func (e *RCONEvent) toGameEvent() GameEvent {
	ge := GameEvent{
		Type:  e.Type,
		Time:  time.Now(),
		Extra: make(map[string]string),
	}
	if e.Player != "" {
		ge.Player = e.Player
	}
	if e.Name != "" {
		ge.Extra["name"] = e.Name
	}
	if e.Cause != "" {
		ge.Extra["cause"] = e.Cause
	}
	if e.Surface != "" {
		ge.Extra["surface"] = e.Surface
	}
	if e.State != "" {
		ge.Extra["state"] = e.State
	}
	if e.Text != "" {
		ge.Extra["text"] = e.Text
	}
	return ge
}
