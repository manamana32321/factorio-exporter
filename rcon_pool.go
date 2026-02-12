package main

import (
	"fmt"
	"net"
	"sync"

	"github.com/gorcon/rcon"
)

// RCONPool provides a shared, mutex-protected RCON connection with auto-reconnect.
type RCONPool struct {
	addr     string
	password string
	mu       sync.Mutex
	conn     *rcon.Conn
}

func NewRCONPool(host, port, password string) *RCONPool {
	return &RCONPool{
		addr:     net.JoinHostPort(host, port),
		password: password,
	}
}

// Execute runs an RCON command, reconnecting on failure.
func (p *RCONPool) Execute(cmd string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	conn, err := p.getConn()
	if err != nil {
		return "", fmt.Errorf("rcon connect: %w", err)
	}

	resp, err := conn.Execute(cmd)
	if err != nil {
		// Connection may be stale; close and retry once
		p.conn.Close()
		p.conn = nil

		conn, err = p.getConn()
		if err != nil {
			return "", fmt.Errorf("rcon reconnect: %w", err)
		}
		resp, err = conn.Execute(cmd)
		if err != nil {
			p.conn.Close()
			p.conn = nil
			return "", fmt.Errorf("rcon execute after reconnect: %w", err)
		}
	}
	return resp, nil
}

func (p *RCONPool) getConn() (*rcon.Conn, error) {
	if p.conn != nil {
		return p.conn, nil
	}
	conn, err := rcon.Dial(p.addr, p.password, rcon.SetMaxCommandLen(4096))
	if err != nil {
		return nil, err
	}
	p.conn = conn
	return conn, nil
}

func (p *RCONPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
