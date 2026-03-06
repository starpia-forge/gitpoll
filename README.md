# gitpoll

![Go Version](https://img.shields.io/github/go-mod/go-version/starpia-forge/gitpoll)
![Latest Release](https://img.shields.io/github/v/release/starpia-forge/gitpoll)
![CI/CD](https://github.com/starpia-forge/gitpoll/actions/workflows/ci-cd.yaml/badge.svg)

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

Configuration is managed locally through a JSON file and an interactive TUI setup wizard.

### Configuration Files

`gitpoll` uses a local JSON configuration system. You do not need to create this file manually; the application provides an interactive, paginated setup wizard that will generate it for you if it is missing.

- **Local Configuration**: `./gitpoll.config.json`

### Running the Application

#### Option 1: Direct Execution
1. Download the latest pre-built binary from the GitHub Releases page, or build it yourself (`go build ./cmd/gitpoll`).
2. Run the executable:

```bash
./gitpoll
```

3. If this is your first time running the program or if the configuration is incomplete, an interactive **Setup Wizard** will launch, displaying the project's ASCII art header.
4. Follow the paginated on-screen prompts in the terminal to configure the repository URL, local directory, branch, execution command, and polling interval. You can leave fields blank to use the suggested defaults shown in brackets.
5. Review the summary of your settings and confirm to save them to `./gitpoll.config.json`.

#### Option 2: Docker

You can run `gitpoll` in a Docker container using the pre-built image hosted on Docker Hub. This method provides an isolated environment and requires no local installation other than Docker.

```bash
docker run -it \
  -v $(pwd)/gitpoll.config.json:/app/gitpoll.config.json \
  -v /path/to/your/local/repo:/app/repo \
  starpia/gitpoll:latest
```

*Note: Since the application runs via a Terminal UI (TUI) and uses interactive prompts for setup, you need to use the `-it` flag. You should also mount your configuration file and target repository directory using `-v` so the container can access your data and save your settings persistently.*

## License

All code was written by `jules`.
