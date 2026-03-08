package poller

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	gogit "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"

	"repo-gitpoll/internal/config"
	"repo-gitpoll/internal/events"
)

// GitClient defines an interface for remote git operations
type GitClient interface {
	LsRemote(ctx context.Context, repoURL, branch string) (string, error)
}

// defaultGitClient implements GitClient using go-git
type defaultGitClient struct{}

func (c *defaultGitClient) LsRemote(ctx context.Context, repoURL, branch string) (string, error) {
	rem := gogit.NewRemote(memory.NewStorage(), &gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})

	listOpts := &gogit.ListOptions{}

	if ep, err := transport.NewEndpoint(repoURL); err == nil && ep.Protocol == "ssh" {
		user := ep.User
		if user == "" {
			user = "git"
		}

		// Try ssh-agent first
		auth, authErr := ssh.DefaultAuthBuilder(user)

		// Fallback to local keys if agent is not available
		if authErr != nil {
			if homeDir, err := os.UserHomeDir(); err == nil {
				keys := []string{"id_ed25519", "id_rsa", "id_ecdsa", "id_dsa"}
				for _, key := range keys {
					keyPath := filepath.Join(homeDir, ".ssh", key)
					if _, err := os.Stat(keyPath); err == nil {
						if pkAuth, err := ssh.NewPublicKeysFromFile(user, keyPath, ""); err == nil {
							auth = pkAuth
							break
						}
					}
				}
			}
		}

		if auth != nil {
			listOpts.Auth = auth
		}
	}

	refs, err := rem.ListContext(ctx, listOpts)
	if err != nil {
		return "", err
	}

	for _, ref := range refs {
		if ref.Name().Short() == branch {
			return ref.Hash().String(), nil
		}
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

	baseInterval     time.Duration
	maxJitter        time.Duration
	backoffBase      time.Duration
	backoffMax       time.Duration
	executeOnStartup bool

	lastHash string
}

func NewPoller(cfg *config.Config, client GitClient) Poller {
	if client == nil {
		client = &defaultGitClient{}
	}

	interval := cfg.Interval
	if interval == 0 {
		interval = 30 * time.Second
	}

	return &defaultPoller{
		repoURL:          cfg.RepoURL,
		branch:           cfg.Branch,
		client:           client,
		baseInterval:     interval,
		maxJitter:        20 * time.Second,
		backoffBase:      5 * time.Second,
		backoffMax:       5 * time.Minute,
		executeOnStartup: cfg.ExecuteOnStartup,
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
			isFirstPoll := p.lastHash == ""
			p.lastHash = hash

			if isFirstPoll && !p.executeOnStartup {
				// Skip emitting the initial update message
			} else {
				select {
				case out <- events.UpdateDetectedMsg{NewHash: hash}:
				case <-ctx.Done():
					return
				}
			}
		}

		jitter := time.Duration(0)
		if p.maxJitter > 0 {
			// #nosec G404 - weak random is acceptable for jitter
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
