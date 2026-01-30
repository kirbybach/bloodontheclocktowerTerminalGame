package tui

import (
	"clocktower/model"
	"math/rand"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type MainModel struct {
	game      *model.Game
	setup     *SetupModel
	grimoire  *GrimoireModel
	viewState ViewState
}

type ViewState int

const (
	ViewSetup ViewState = iota
	ViewGrimoire
)

func NewMainModel() *MainModel {
	g := model.NewGame()

	// Try loading existing state
	if err := g.LoadState(); err == nil {
		// If loaded, go straight to Grimoire
		return &MainModel{
			game:      g,
			grimoire:  NewGrimoireModel(g),
			viewState: ViewGrimoire,
		}
	}

	return &MainModel{
		game:      g,
		setup:     NewSetupModel(g),
		viewState: ViewSetup,
	}
}

func (m *MainModel) Init() tea.Cmd {
	if m.viewState == ViewSetup {
		return m.setup.Init()
	}
	return nil
}

type ResetGameMsg struct{}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case ResetGameMsg:
		os.Remove("game_state.json")
		return m, tea.Quit
	}

	var cmd tea.Cmd

	switch m.viewState {
	case ViewSetup:
		_, cmd = m.setup.Update(msg)
		if m.setup.finished {
			m.transitionToGame()
		}
	case ViewGrimoire:
		_, cmd = m.grimoire.Update(msg)
	}

	return m, cmd
}

func (m *MainModel) View() string {
	switch m.viewState {
	case ViewSetup:
		return m.setup.View()
	case ViewGrimoire:
		return m.grimoire.View()
	}
	return "Unknown State"
}

func (m *MainModel) transitionToGame() {
	// Finalize setup
	// Assign roles randomly if not set (simple implementation for now)
	if len(m.game.Players) > 0 && m.game.Players[0].Role.Name == "" {
		m.assignRandomRoles()
	}

	// Initialize Grimoire
	m.grimoire = NewGrimoireModel(m.game)
	m.viewState = ViewGrimoire
	m.game.SaveState()
}

func (m *MainModel) assignRandomRoles() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Separate roles by type
	var townsfolk, outsiders, minions, demons []model.Role
	for _, role := range m.game.Script.Roles {
		switch role.Type {
		case model.Townsfolk:
			townsfolk = append(townsfolk, role)
		case model.Outsider:
			outsiders = append(outsiders, role)
		case model.Minion:
			minions = append(minions, role)
		case model.Demon:
			demons = append(demons, role)
		}
	}

	// Shuffle buckets
	shuffle := func(roles []model.Role) {
		r.Shuffle(len(roles), func(i, j int) { roles[i], roles[j] = roles[j], roles[i] })
	}
	shuffle(townsfolk)
	shuffle(outsiders)
	shuffle(minions)
	shuffle(demons)

	// Get distribution counts
	tfCount, outCount, minCount, demCount := model.GetDistribution(len(m.game.Players))

	// Select roles
	var selectedRoles []model.Role
	selectedRoles = append(selectedRoles, townsfolk[:tfCount]...)
	selectedRoles = append(selectedRoles, outsiders[:outCount]...)
	selectedRoles = append(selectedRoles, minions[:minCount]...)
	selectedRoles = append(selectedRoles, demons[:demCount]...)

	// Shuffle final selection so they are distributed randomly to players
	shuffle(selectedRoles)

	for i, p := range m.game.Players {
		if i < len(selectedRoles) {
			p.Role = selectedRoles[i]
		}
	}
}
