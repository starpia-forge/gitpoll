package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"repo-gitpoll/internal/config"
	"repo-gitpoll/internal/events"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	infoStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	logStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	boxStyle   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)
)

// MonitorModel holds the state for the TUI monitor
type MonitorModel struct {
	eventBus   events.Bus
	cancelFunc context.CancelFunc

	status     string
	latestHash string
	logs       []string
	maxLogs    int
	viewport   viewport.Model
	ready      bool
	lastError  error

	eventCh chan tea.Msg
	width   int
	height  int
}

// NewMonitorModel creates a new bubbletea model initialized with the event bus
func NewMonitorModel(bus events.Bus, cancelFunc context.CancelFunc) *MonitorModel {
	m := &MonitorModel{
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
func (m *MonitorModel) waitForEvents() tea.Msg {
	return <-m.eventCh
}

func (m *MonitorModel) Init() tea.Cmd {
	return m.waitForEvents
}

func (m *MonitorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.cancelFunc() // graceful shutdown trigger
			return m, tea.Quit
		}
		// Forward other key events to viewport (handles up, down, pgup, pgdown, mouse wheel, etc.)
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := lipgloss.Height(m.headerView()) + 2 // include footer and margins

		// The box style uses borders, which consumes space. Let's calculate inner dimensions.
		h, v := boxStyle.GetFrameSize()

		vpWidth := msg.Width - h - 2
		vpHeight := msg.Height - headerHeight - v

		if vpWidth < 0 {
			vpWidth = 0
		}
		if vpHeight < 0 {
			vpHeight = 0
		}

		if !m.ready {
			m.viewport = viewport.New(vpWidth, vpHeight)
			m.viewport.SetContent(strings.Join(m.logs, "\n"))
			m.ready = true
		} else {
			m.viewport.Width = vpWidth
			m.viewport.Height = vpHeight
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

func (m *MonitorModel) headerView() string {
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
	return fmt.Sprintf("%s\n%s", header, logHeader)
}

func (m *MonitorModel) View() string {
	if !m.ready {
		return "\n  Initializing... (Waiting for resize event to set viewport)"
	}

	header := m.headerView()

	// Wrap viewport in a styled box
	viewportView := boxStyle.
		Width(m.width - 2). // adjust width
		Render(logStyle.Render(m.viewport.View()))

	footer := infoStyle.Render("Press 'q' or 'Ctrl+C' to quit. Use Up/Down arrows to scroll logs.")

	return fmt.Sprintf("%s\n%s\n%s", header, viewportView, footer)
}

type MainState int

const (
	StateWizard MainState = iota
	StateMonitor
)

type MainModel struct {
	state   MainState
	wizard  *WizardModel
	monitor *MonitorModel

	ConfigReadyCb func(*config.Config, *MainModel)
	initCfg       *config.Config
	isValid       bool
	bus           events.Bus
	cancelFunc    context.CancelFunc
	width         int
	height        int
}

func NewMainModel(cfg *config.Config, isValid bool, bus events.Bus, cancelFunc context.CancelFunc, configReadyCb func(*config.Config, *MainModel)) *MainModel {
	m := &MainModel{
		ConfigReadyCb: configReadyCb,
		initCfg:       cfg,
		isValid:       isValid,
		bus:           bus,
		cancelFunc:    cancelFunc,
	}

	if !isValid {
		m.state = StateWizard
		m.wizard = NewWizardModel(cfg)
	} else {
		m.state = StateMonitor
		m.monitor = NewMonitorModel(bus, cancelFunc)
		// We can't trigger the callback directly from here safely if it requires Bubble Tea cmds,
		// but since we start in StateMonitor, the caller can just start everything.
		// Wait, we need a way to tell the caller that config is ready if we bypassed the wizard.
		// A cleaner way is to handle that in the Init() function using a command.
	}

	return m
}

type SetupMonitorMsg struct {
	Monitor *MonitorModel
}

func (m *MainModel) SetupMonitor(mon *MonitorModel) tea.Cmd {
	return func() tea.Msg {
		return SetupMonitorMsg{Monitor: mon}
	}
}

func (m *MainModel) Init() tea.Cmd {
	if m.state == StateWizard {
		return m.wizard.Init()
	}

	// If starting directly in monitor, notify caller
	if m.ConfigReadyCb != nil {
		go m.ConfigReadyCb(m.initCfg, m)
	}
	return m.monitor.Init()
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.state == StateMonitor && m.monitor != nil {
			var mModel tea.Model
			mModel, cmd = m.monitor.Update(msg)
			if mon, ok := mModel.(*MonitorModel); ok {
				m.monitor = mon
			}
			return m, cmd
		}

	case ConfigReadyMsg:
		m.state = StateMonitor
		m.monitor = NewMonitorModel(m.bus, m.cancelFunc)

		if m.ConfigReadyCb != nil {
			go m.ConfigReadyCb(msg.Config, m)
		}

		// If we already received window size, pass it to monitor
		if m.width > 0 && m.height > 0 {
			m.monitor.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}

		return m, m.monitor.Init()

	case SetupMonitorMsg:
		m.monitor = msg.Monitor
		// Simulate resize if we already received it
		if m.width > 0 && m.height > 0 {
			m.monitor.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		return m, m.monitor.Init()
	}

	if m.state == StateWizard {
		var wModel tea.Model
		wModel, cmd = m.wizard.Update(msg)

		if w, ok := wModel.(*WizardModel); ok {
			m.wizard = w
		}

		// If wizard returned ConfigReadyMsg in Batch or Cmd
		// It will be caught in the next Update cycle.
		return m, cmd
	}

	var monModel tea.Model
	monModel, cmd = m.monitor.Update(msg)
	if mon, ok := monModel.(*MonitorModel); ok {
		m.monitor = mon
	}

	return m, cmd
}

func (m *MainModel) View() string {
	if m.state == StateWizard {
		if m.wizard != nil {
			return m.wizard.View()
		}
		return "Initializing wizard..."
	}
	if m.monitor != nil {
		return m.monitor.View()
	}
	return "Initializing monitor..."
}
