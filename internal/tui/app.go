package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"repo-gitpoll/internal/events"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	infoStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	logStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
)

// MainModel holds the state for the TUI
type MainModel struct {
	eventBus    events.Bus
	cancelFunc  context.CancelFunc

	status      string
	latestHash  string
	logs        []string
	maxLogs     int
	viewport    viewport.Model
	ready       bool
	lastError   error

	eventCh     chan tea.Msg
}

// NewApp creates a new bubbletea model initialized with the event bus
func NewApp(bus events.Bus, cancelFunc context.CancelFunc) *MainModel {
	m := &MainModel{
		eventBus:   bus,
		cancelFunc: cancelFunc,
		status:     "Polling",
		logs:       make([]string, 0),
		maxLogs:    500,
		eventCh:    make(chan tea.Msg, 100), // Buffer to avoid blocking
	}

	// Helper to send to channel without blocking forever
	sendEvent := func(msg tea.Msg) {
		select {
		case m.eventCh <- msg:
		default:
			// If buffer is full, drop or block. For log streaming we shouldn't block bus.
		}
	}

	bus.Subscribe(events.RepoChanged, func(payload interface{}) {
		if msg, ok := payload.(events.UpdateDetectedMsg); ok {
			sendEvent(msg)
		}
	})
	bus.Subscribe(events.RepoUpdated, func(payload interface{}) {
		sendEvent(events.PullCompletedMsg{})
	})
	bus.Subscribe(events.CommandExecuted, func(payload interface{}) {
		sendEvent(events.CommandExecutedMsg{})
	})
	bus.Subscribe(events.ErrorOccurred, func(payload interface{}) {
		if msg, ok := payload.(events.ErrorMsg); ok {
			sendEvent(msg)
		} else if err, ok := payload.(error); ok {
			sendEvent(events.ErrorMsg{Err: err})
		}
	})
	bus.Subscribe(events.LogEmitted, func(payload interface{}) {
		if msg, ok := payload.(events.LogEmittedMsg); ok {
			sendEvent(msg)
		} else if str, ok := payload.(string); ok {
			sendEvent(events.LogEmittedMsg{Log: str})
		}
	})

	return m
}

// waitForEvents is a tea.Cmd that continuously reads from our event channel
func (m *MainModel) waitForEvents() tea.Msg {
	return <-m.eventCh
}

func (m *MainModel) Init() tea.Cmd {
	return m.waitForEvents
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.cancelFunc() // graceful shutdown trigger
			return m, tea.Quit
		case "up", "k":
			m.viewport.LineUp(1)
		case "down", "j":
			m.viewport.LineDown(1)
		}

	case tea.WindowSizeMsg:
		headerHeight := 8
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight)
			m.viewport.SetContent(strings.Join(m.logs, "\n"))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight
		}

	case events.UpdateDetectedMsg:
		m.latestHash = msg.NewHash
		m.status = "Pulling"
		m.lastError = nil
		cmds = append(cmds, m.waitForEvents)

	case events.PullCompletedMsg:
		m.status = "Executing Command"
		cmds = append(cmds, m.waitForEvents)

	case events.CommandExecutedMsg:
		m.status = "Polling"
		cmds = append(cmds, m.waitForEvents)

	case events.LogEmittedMsg:
		m.logs = append(m.logs, msg.Log)
		if len(m.logs) > m.maxLogs {
			m.logs = m.logs[len(m.logs)-m.maxLogs:]
		}
		if m.ready {
			m.viewport.SetContent(strings.Join(m.logs, "\n"))
			m.viewport.GotoBottom()
		}
		cmds = append(cmds, m.waitForEvents)

	case events.ErrorMsg:
		m.status = "Error (Retrying)"
		m.lastError = msg.Err
		cmds = append(cmds, m.waitForEvents)
	}

	return m, tea.Batch(cmds...)
}

func (m *MainModel) View() string {
	if !m.ready {
		return "\n  Initializing... (Waiting for resize event to set viewport)"
	}

	title := titleStyle.Render("Gitpoll TUI")
	statusLine := fmt.Sprintf("Status: %s", m.status)
	hashLine := fmt.Sprintf("Latest Commit: %s", m.latestHash)

	header := fmt.Sprintf("%s\n\n%s\n%s\n", title, statusLine, hashLine)

	if m.lastError != nil {
		header += errorStyle.Render(fmt.Sprintf("Error: %v\n", m.lastError))
	} else {
		header += "\n"
	}

	logHeader := infoStyle.Render("--- Execution Logs ---")

	body := fmt.Sprintf("%s\n%s\n%s\n", header, logHeader, logStyle.Render(m.viewport.View()))
	footer := infoStyle.Render("\nPress 'q' or 'Ctrl+C' to quit. Use Up/Down arrows to scroll logs.")

	return body + footer
}
