package poller

import (
	"context"
	"math/rand"
	"strings"
	"time"
	"os/exec"

	"repo-gitpoll/internal/events"
)

// GitClient defines an interface for remote git operations
type GitClient interface {
	LsRemote(ctx context.Context, repoURL, branch string) (string, error)
}

// defaultGitClient implements GitClient using os/exec
type defaultGitClient struct{}

func (c *defaultGitClient) LsRemote(ctx context.Context, repoURL, branch string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", repoURL, branch)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	parts := strings.Fields(string(out))
	if len(parts) > 0 {
		return parts[0], nil
	}
	return "", nil
}

// Poller defines the interface for watching repository changes
type Poller interface {
	Start(ctx context.Context, out chan<- interface{})
}

type defaultPoller struct {
	repoURL string
	branch  string

	client GitClient

	baseInterval time.Duration
	maxJitter    time.Duration
	backoffBase  time.Duration
	backoffMax   time.Duration

	lastHash string
}

func NewPoller(url, branch string, client GitClient) Poller {
	if client == nil {
		client = &defaultGitClient{}
	}
	return &defaultPoller{
		repoURL:      url,
		branch:       branch,
		client:       client,
		baseInterval: 10 * time.Second,
		maxJitter:    20 * time.Second,
		backoffBase:  5 * time.Second,
		backoffMax:   5 * time.Minute,
	}
}

func (p *defaultPoller) Start(ctx context.Context, out chan<- interface{}) {
	var backoff time.Duration

	for {
		// Only apply backoff if there was an error
		if backoff > 0 {
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
		}

		hash, err := p.client.LsRemote(ctx, p.repoURL, p.branch)
		if err != nil {
			select {
			case out <- events.ErrorMsg{Err: err}:
			case <-ctx.Done():
				return
			}

			if backoff == 0 {
				backoff = p.backoffBase
			} else {
				backoff *= 2
				if backoff > p.backoffMax {
					backoff = p.backoffMax
				}
			}
			continue
		}

		// Success, reset backoff
		backoff = 0

		if hash != "" && hash != p.lastHash {
			p.lastHash = hash
			select {
			case out <- events.UpdateDetectedMsg{NewHash: hash}:
			case <-ctx.Done():
				return
			}
		}

		jitter := time.Duration(0)
		if p.maxJitter > 0 {
			jitter = time.Duration(rand.Int63n(int64(p.maxJitter)))
		}
		nextTick := p.baseInterval + jitter

		select {
		case <-time.After(nextTick):
		case <-ctx.Done():
			return
		}
	}
}
