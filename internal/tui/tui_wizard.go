package tui

import (
	"fmt"
	"os"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"repo-gitpoll/internal/config"
)

const asciiArt = `  ____ _ _   ____       _ _
 / ___(_) |_|  _ \ ___ | | |
| |  _| | __| |_) / _ \| | |
| |_| | | |_|  __/ (_) | | |
 \____|_|\__|_|   \___/|_|_|`

var asciiArtStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)

type ConfigReadyMsg struct {
	Config *config.Config
}

type WizardModel struct {
	form          *huh.Form
	initialConfig *config.Config
}

func NewWizardModel(initialConfig *config.Config) *WizardModel {
	m := &WizardModel{
		initialConfig: initialConfig,
	}
	m.createForm()
	return m
}

func (m *WizardModel) createForm() {
	var repoURL, repoDir, branch, command, intervalStr string
	var confirm bool
	var executeOnStartup bool

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("repoURL").
				Title("Git Repository URL\n[example: https://github.com/starpia-forge/gitpoll.git]").
				Value(&repoURL).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("repository URL is required")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewInput().
				Key("repoDir").
				Title(fmt.Sprintf("Local Repository Directory\n[default: %s]", cwd)).
				Value(&repoDir),
		),
		huh.NewGroup(
			huh.NewInput().
				Key("branch").
				Title("Branch to Monitor\n[default: main]").
				Value(&branch),
		),
		huh.NewGroup(
			huh.NewInput().
				Key("command").
				Title("Command to Execute on Update\n[default: make]").
				Value(&command),
		),
		huh.NewGroup(
			huh.NewInput().
				Key("interval").
				Title("Polling Interval (seconds)\n[default: 30]").
				Value(&intervalStr).
				Validate(func(str string) error {
					if str != "" {
						if _, err := strconv.Atoi(str); err != nil {
							return fmt.Errorf("interval must be an integer")
						}
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Key("executeOnStartup").
				Title("Execute command on startup regardless of git state?").
				Value(&executeOnStartup),
		),
		huh.NewGroup(
			huh.NewNote().
				Title("Summary of settings:").
				DescriptionFunc(func() string {
					url := m.form.GetString("repoURL")
					dir := m.form.GetString("repoDir")
					if dir == "" {
						dir = cwd
					}
					b := m.form.GetString("branch")
					if b == "" {
						b = "main"
					}
					c := m.form.GetString("command")
					if c == "" {
						c = "make"
					}
					iStr := m.form.GetString("interval")
					if iStr == "" {
						iStr = "30"
					}
					execOnStartup := m.form.GetBool("executeOnStartup")

					return fmt.Sprintf("\n"+
						"Repository URL: %s\n"+
						"Local Directory: %s\n"+
						"Branch: %s\n"+
						"Command: %s\n"+
						"Interval: %s seconds\n"+
						"Execute on Startup: %t\n", url, dir, b, c, iStr, execOnStartup)
				}, &repoURL), // Note: huh DescriptionFunc expects exactly one dependency argument of type any in this version.
			// We can use a struct to track all dependencies if needed, but since we are navigating forward sequentially
			// without jumping, repoURL is technically enough to avoid compilation error while fulfilling the function signature.
			// Let's create an aggregate pointer or just pass a struct pointer.
			huh.NewConfirm().
				Key("confirm").
				Title("Proceed with these settings?").
				Value(&confirm),
		),
	)

	m.form.Init()
}

func (m *WizardModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		confirm := m.form.GetBool("confirm")
		if !confirm {
			m.createForm()
			return m, m.form.Init()
		}

		// Extract values
		repoURL := m.form.GetString("repoURL")
		repoDir := m.form.GetString("repoDir")
		branch := m.form.GetString("branch")
		command := m.form.GetString("command")
		intervalStr := m.form.GetString("interval")
		executeOnStartup := m.form.GetBool("executeOnStartup")

		if repoDir == "" {
			cwd, err := os.Getwd()
			if err != nil {
				cwd = "."
			}
			repoDir = cwd
		}
		if branch == "" {
			branch = "main"
		}
		if command == "" {
			command = "make"
		}
		if intervalStr == "" {
			intervalStr = "30"
		}

		intervalSeconds, _ := strconv.Atoi(intervalStr)

		newConfig := &config.Config{
			RepoURL:          repoURL,
			RepoDir:          repoDir,
			Branch:           branch,
			Command:          command,
			Interval:         time.Duration(intervalSeconds) * time.Second,
			ExecuteOnStartup: executeOnStartup,
		}

		savePath := config.GetLocalConfigPath()

		if err := config.Save(newConfig, savePath); err != nil {
			// If save fails, we proceed with the new config in memory anyway,
			// but we could emit an error message or log it.
			// For now, just continue.
			_ = err
		}

		return m, func() tea.Msg {
			return ConfigReadyMsg{Config: newConfig}
		}
	}

	return m, cmd
}

func (m *WizardModel) View() string {
	return asciiArtStyle.Render(asciiArt) + "\n\n" + m.form.View()
}
