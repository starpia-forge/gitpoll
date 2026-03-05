# gitpoll

A background worker program written in Go that monitors a target GitHub repository branch for changes. When a change is detected, it automatically pulls the latest state to the local machine and executes a specified command.

The application currently features a Terminal User Interface (TUI) built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), and is designed with an event-driven architecture to easily support a future Web GUI.

## Architecture & Components

The application is structured into four main components that communicate asynchronously via an internal **Event Bus** (`internal/events`). This decoupled design ensures high maintainability and scalability, especially for integrating a future web interface without altering core logic.

1. **Polling Worker (`internal/poller`)**
   - Periodically checks the remote GitHub repository for any new commits or state changes on the specified branch.
   - Publishes `RepoChanged` events to the Event Bus when new changes are detected.

2. **Git Manager (`internal/git`)**
   - Subscribes to `RepoChanged` events.
   - Responsible for local repository operations, primarily pulling the latest changes from the remote repository.
   - Publishes a `RepoUpdated` event upon successful synchronization.

3. **Command Executor (`internal/executor`)**
   - Subscribes to `RepoUpdated` events.
   - Executes the user-defined shell command (e.g., restarting a service, building a project) once the repository has been successfully updated locally.
   - Publishes a `CommandExecuted` event upon completion.

4. **TUI / State Manager (`internal/tui`)**
   - The primary presentation layer for the terminal.
   - Listens to various events on the Event Bus to update the UI state in real-time (e.g., displaying sync status, execution logs, or errors).

5. **Future Web GUI (`internal/server`)**
   - An architectural placeholder intended to host a local web server in the future.
   - By leveraging the same Event Bus used by the TUI, it will be able to broadcast real-time state changes to a web frontend via WebSockets or Server-Sent Events (SSE).

## Directory Structure

```text
.
├── cmd
│   └── gitpoll
│       └── main.go           # Application entry point, component initialization & wiring
├── internal
│   ├── config                # Environment variable parsing and configuration models
│   ├── events                # The Event Bus implementation and event type definitions
│   ├── executor              # Logic for executing arbitrary shell commands
│   ├── git                   # Logic for local git operations (clone, pull)
│   ├── poller                # Background worker checking for remote git changes
│   ├── server                # Skeleton for future Web GUI integration
│   └── tui                   # Bubble Tea based Terminal UI
├── go.mod
└── README.md
```

## Setup & Configuration

Configuration is injected strictly through Environment Variables.

### Environment Variables

| Variable | Description | Default | Required |
| --- | --- | --- | --- |
| `GITPOLL_REPO_URL` | The URL of the GitHub repository to watch | | Yes |
| `GITPOLL_REPO_DIR` | The local directory where the repository is/will be cloned | | Yes |
| `GITPOLL_BRANCH` | The branch name to monitor | `main` | No |
| `GITPOLL_COMMAND` | The command to execute after a successful git pull | | Yes |
| `GITPOLL_INTERVAL` | The interval duration for polling (e.g., `30s`, `1m`) | `1m` | No |

### Running the Application

1. Ensure you have Go installed.
2. Clone this repository and navigate to its root directory.
3. Export the necessary environment variables:

```bash
export GITPOLL_REPO_URL="https://github.com/user/repo.git"
export GITPOLL_REPO_DIR="/path/to/local/repo"
export GITPOLL_BRANCH="main"
export GITPOLL_COMMAND="systemctl restart my-service"
export GITPOLL_INTERVAL="30s"
```

4. Run the program:

```bash
go run ./cmd/gitpoll/main.go
```
