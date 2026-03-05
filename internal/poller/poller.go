package poller

import (
	"repo-gitpoll/internal/events"
	"time"
)

// Poller defines the interface for watching repository changes
type Poller interface {
	Start()
	Stop()
}

type defaultPoller struct {
	repoURL  string
	branch   string
	interval time.Duration
	eventBus events.Bus
	stopCh   chan struct{}
}

// NewPoller creates a new instance of a Git repository poller
func NewPoller(url, branch string, interval time.Duration, bus events.Bus) Poller {
	return &defaultPoller{
		repoURL:  url,
		branch:   branch,
		interval: interval,
		eventBus: bus,
		stopCh:   make(chan struct{}),
	}
}

func (p *defaultPoller) Start() {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// TODO: Implement actual remote git ls-remote or git fetch logic here
			// If change detected:
			// p.eventBus.Publish(events.RepoChanged, nil)
		case <-p.stopCh:
			return
		}
	}
}

func (p *defaultPoller) Stop() {
	close(p.stopCh)
}
