package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"repo-gitpoll/internal/config"
	"repo-gitpoll/internal/events"
	"repo-gitpoll/internal/executor"
	"repo-gitpoll/internal/git"
	"repo-gitpoll/internal/poller"
	"repo-gitpoll/internal/tui"
)

func main() {
	// 1. Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize the Event Bus
	eventBus := events.NewBus()

	// 3. Initialize components
	gitManager := git.NewManager(cfg.RepoDir, cfg.Branch)
	cmdExecutor := executor.NewExecutor(cfg.Command)
	repoPoller := poller.NewPoller(cfg.RepoURL, cfg.Branch, cfg.Interval, eventBus)
	appTUI := tui.NewApp(eventBus)

	// 4. Wire up event listeners
	// When the poller detects a change, the git manager should pull
	eventBus.Subscribe(events.RepoChanged, func(payload interface{}) {
		err := gitManager.Pull()
		if err != nil {
			eventBus.Publish(events.ErrorOccurred, fmt.Errorf("git pull failed: %w", err))
			return
		}
		eventBus.Publish(events.RepoUpdated, nil)
	})

	// When the repo is successfully updated, the executor should run the command
	eventBus.Subscribe(events.RepoUpdated, func(payload interface{}) {
		err := cmdExecutor.Execute()
		if err != nil {
			eventBus.Publish(events.ErrorOccurred, fmt.Errorf("command execution failed: %w", err))
			return
		}
		eventBus.Publish(events.CommandExecuted, nil)
	})

	// 5. Start background worker (Poller)
	go repoPoller.Start()

	// 6. Start TUI (Blocks until exit)
	p := tea.NewProgram(appTUI)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
