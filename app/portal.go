package app

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type Portal struct {
	conns map[string]*Conn
	mu    sync.RWMutex
}

func NewPortal() *Portal {
	return &Portal{
		conns: make(map[string]*Conn),
	}
}

func (p *Portal) GetConnection(id string) (*Conn, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	conn, ok := p.conns[id]
	if !ok {
		return nil, fmt.Errorf("connection not found")
	}
	return conn, nil
}

func (p *Portal) CreateConnection() (string, error) {
	return p.createConnectionWithRetries(3)
}

func (p *Portal) createConnectionWithRetries(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("failed to create connection")
	}
	id := generateId()
	p.mu.Lock()
	if _, ok := p.conns[id]; ok {
		p.mu.Unlock()
		return p.createConnectionWithRetries(n - 1)
	}
	defer p.mu.Unlock()
	conn := NewConn()
	p.conns[id] = conn
	go p.disposeIdleConnection(id, conn)
	return id, nil
}

func (p *Portal) disposeIdleConnection(id string, conn *Conn) {
	timer := time.NewTimer(time.Minute * 5)
	defer timer.Stop()
	select {
	case <-conn.AnyJoined():
		return
	case <-timer.C:
		p.mu.Lock()
		defer p.mu.Unlock()
		conn.Close()
		delete(p.conns, id)
		log.Println("disposed idle connection", id)
		return
	}
}
