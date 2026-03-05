package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"repo-gitpoll/internal/events"
)

// MainModel holds the state for the TUI
type MainModel struct {
	eventBus    events.Bus
	status      string
	lastUpdated string
	log         []string
}

// NewApp creates a new bubbletea model initialized with the event bus
func NewApp(bus events.Bus) tea.Model {
	m := &MainModel{
		eventBus: bus,
		status:   "Initializing...",
		log:      make([]string, 0),
	}

	// TUI should ideally receive messages via tea.Cmd instead of direct event callbacks,
	// but for skeleton purposes, this shows how they might integrate.
	// You might use a channel that returns tea.Msg to bridge event bus to bubbletea.

	return m
}

func (m *MainModel) Init() tea.Cmd {
	// Initialize things like cursor blink, start tick, etc.
	return nil
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle keyboard input (e.g., 'q' to quit)
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	// TODO: Handle custom messages bridging from eventBus (RepoUpdated, ErrorOccurred)
	return m, nil
}

func (m *MainModel) View() string {
	// Simple view
	s := fmt.Sprintf("Repository Gitpoll TUI\n\nStatus: %s\nLast Updated: %s\n\n", m.status, m.lastUpdated)
	s += "Logs:\n"
	for _, l := range m.log {
		s += "- " + l + "\n"
	}
	s += "\nPress 'q' to quit.\n"
	return s
}
