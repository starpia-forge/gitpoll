package tui

import (
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"repo-gitpoll/internal/config"
)

type ConfigReadyMsg struct {
	Config *config.Config
}

type WizardModel struct {
	form          *huh.Form
	initialConfig *config.Config
}

func NewWizardModel(initialConfig *config.Config) *WizardModel {
	if initialConfig == nil {
		initialConfig = &config.Config{
			Branch:   "main",
			Command:  "make",
			Interval: 30 * time.Second,
		}
	} else {
		if initialConfig.Branch == "" {
			initialConfig.Branch = "main"
		}
		if initialConfig.Command == "" {
			initialConfig.Command = "make"
		}
		if initialConfig.Interval == 0 {
			initialConfig.Interval = 30 * time.Second
		}
	}

	var repoURL, repoDir, branch, command, intervalStr string
	var saveLocation string

	repoURL = initialConfig.RepoURL
	repoDir = initialConfig.RepoDir
	branch = initialConfig.Branch
	command = initialConfig.Command
	intervalStr = fmt.Sprintf("%d", int(initialConfig.Interval.Seconds()))
	saveLocation = "local" // default

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("repoURL").
				Title("Git Repository URL").
				Value(&repoURL).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("repository URL is required")
					}
					return nil
				}),

			huh.NewInput().
				Key("repoDir").
				Title("Local Repository Directory").
				Value(&repoDir).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("local directory is required")
					}
					return nil
				}),

			huh.NewInput().
				Key("branch").
				Title("Branch to Monitor").
				Value(&branch).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("branch is required")
					}
					return nil
				}),

			huh.NewInput().
				Key("command").
				Title("Command to Execute on Update").
				Value(&command).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("command is required")
					}
					return nil
				}),

			huh.NewInput().
				Key("interval").
				Title("Polling Interval (seconds)").
				Value(&intervalStr).
				Validate(func(str string) error {
					if str == "" {
						return fmt.Errorf("interval is required")
					}
					if _, err := strconv.Atoi(str); err != nil {
						return fmt.Errorf("interval must be an integer")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("saveLocation").
				Title("Where would you like to save this configuration?").
				Options(
					huh.NewOption("Local (.gitpoll.json)", "local"),
					huh.NewOption("Global (~/.config/gitpoll/config.json)", "global"),
				).
				Value(&saveLocation),
		),
	)

	form.Init()

	return &WizardModel{
		form:          form,
		initialConfig: initialConfig,
	}
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
		// Extract values
		repoURL := m.form.GetString("repoURL")
		repoDir := m.form.GetString("repoDir")
		branch := m.form.GetString("branch")
		command := m.form.GetString("command")
		intervalStr := m.form.GetString("interval")
		saveLocation := m.form.GetString("saveLocation")

		intervalSeconds, _ := strconv.Atoi(intervalStr)

		newConfig := &config.Config{
			RepoURL:  repoURL,
			RepoDir:  repoDir,
			Branch:   branch,
			Command:  command,
			Interval: time.Duration(intervalSeconds) * time.Second,
		}

		var savePath string
		var err error
		if saveLocation == "global" {
			savePath, err = config.GetGlobalConfigPath()
			if err != nil {
				// We should ideally show error in UI, but for now we fallback to local or panic
				savePath = config.GetLocalConfigPath()
			}
		} else {
			savePath = config.GetLocalConfigPath()
		}

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
	return m.form.View()
}
