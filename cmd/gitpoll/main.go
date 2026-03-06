package main

import (
	"context"
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
	// 1. Load configuration from files
	cfg, isValid, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Create root context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize the Event Bus
	eventBus := events.NewBus()

	// Channels to bridge to event bus
	pollerCh := make(chan interface{}, 10)
	logCh := make(chan string, 100)

	// Callback function to initialize components once config is ready
	configReadyCb := func(finalCfg *config.Config, mainModel *tui.MainModel) {
		// Initialize components with the final configuration
		gitManager := git.NewManager(finalCfg)
		cmdExecutor := executor.NewExecutor(finalCfg)
		repoPoller := poller.NewPoller(finalCfg, nil)

		// Create the monitor model - no longer explicitly passed back since mainModel handles it

		// Wait... Since Bubbletea executes commands async, we need to send the initialized monitor
		// to the main model so it can route correctly. But wait, `MainModel` expects a MonitorModel
		// and we created one inside its `Update(ConfigReadyMsg)` method too...
		// Actually, let's keep it simple: `MainModel` handles the state transition, but we need
		// to hook up the event bus and components.
		// So we just start the background workers here.

		// Route poller channel to bus
		go func() {
			for {
				select {
				case msg := <-pollerCh:
					switch m := msg.(type) {
					case events.UpdateDetectedMsg:
						eventBus.Publish(events.RepoChanged, m)
					case events.ErrorMsg:
						eventBus.Publish(events.ErrorOccurred, m)
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		// Route executor logs to bus
		go func() {
			for {
				select {
				case logLine := <-logCh:
					eventBus.Publish(events.LogEmitted, events.LogEmittedMsg{Log: logLine})
				case <-ctx.Done():
					return
				}
			}
		}()

		var execCancel context.CancelFunc
		var execDone chan struct{}

		// Wire up event listeners
		eventBus.Subscribe(events.RepoChanged, func(payload interface{}) {
			err := gitManager.Pull(ctx)
			if err != nil {
				eventBus.Publish(events.ErrorOccurred, fmt.Errorf("git pull failed: %w", err))
				return
			}
			eventBus.Publish(events.RepoUpdated, nil)
		})

		eventBus.Subscribe(events.RepoUpdated, func(payload interface{}) {
			// Stop previous execution AFTER code is successfully pulled
			if execCancel != nil {
				execCancel()
				// Wait for it to cleanly shut down
				if execDone != nil {
					<-execDone
				}
				execCancel = nil
				execDone = nil
			}

			// Start new execution
			var execCtx context.Context
			execCtx, execCancel = context.WithCancel(ctx)
			execDone = make(chan struct{})

			go func(c context.Context, done chan struct{}) {
				defer close(done)
				err := cmdExecutor.Execute(c, logCh)
				if err != nil {
					// Check if context was canceled, ignore error if it was a deliberate cancel
					if c.Err() == context.Canceled {
						return
					}
					eventBus.Publish(events.ErrorOccurred, fmt.Errorf("command execution failed: %w", err))
					return
				}
				eventBus.Publish(events.CommandExecuted, nil)
			}(execCtx, execDone)
		})

		// Start background worker
		go repoPoller.Start(ctx, pollerCh)

		// Send the setup monitor message so TUI knows everything is ready and can display the correct initialized monitor.
		// However, it's not possible to easily send tea.Cmd from outside.
		// Wait, we can pass `eventBus` into `tui.NewMonitorModel`...
		// Ah, we can just initialize `monitorModel` and inject it to mainModel.
		// No need to overcomplicate: we already pass `bus` and `cancel` to `tui.NewMainModel`.
	}

	appTUI := tui.NewMainModel(cfg, isValid, eventBus, cancel, configReadyCb)

	// 3. Start TUI (Blocks until exit)
	p := tea.NewProgram(appTUI, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		cancel() // Ensure background tasks are canceled if TUI fails
		os.Exit(1)
	}

	// At this point TUI exited, context is canceled.
}
