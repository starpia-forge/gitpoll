package server

import (
	"repo-watcher/internal/events"
)

// Server defines the interface for a web/API server (Web GUI)
type Server interface {
	Start(port int) error
	Stop() error
}

type webServer struct {
	eventBus events.Bus
}

// NewServer creates a new Web GUI Server instance
// This demonstrates how a future web component can seamlessly integrate
// by consuming the same Event Bus.
func NewServer(bus events.Bus) Server {
	return &webServer{
		eventBus: bus,
	}
}

func (s *webServer) Start(port int) error {
	// TODO: Initialize HTTP router (e.g., Gin, Echo, or net/http)
	// TODO: Serve static assets for the GUI
	// TODO: Serve websocket or SSE endpoints subscribing to s.eventBus for real-time updates
	return nil
}

func (s *webServer) Stop() error {
	return nil
}
