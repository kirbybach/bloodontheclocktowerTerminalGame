package tui

import (
	"clocktower/model"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

type SetupModel struct {
	form     *huh.Form
	game     *model.Game
	width    int
	height   int
	finished bool
}

func NewSetupModel(game *model.Game) *SetupModel {
	// Initialize with empty form, will be built in Init or Update
	m := &SetupModel{
		game: game,
	}
	m.form = m.buildScriptSelectionForm()
	return m
}

func (m *SetupModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Process form
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
		cmds = append(cmds, cmd)
	}

	// Check if current form is completed
	if m.form.State == huh.StateCompleted {
		// Logic to move to next step/form or finish setup
		if m.game.Script.Name == "" {
			// Script just selected, load it
			m.loadScript()
			// Now ask for player count
			m.form = m.buildPlayerCountForm()
			cmds = append(cmds, m.form.Init())
		} else if len(m.game.Players) == 0 {
			// Player count selected, build names form
			countStr := m.form.GetString("player_count")
			count, _ := strconv.Atoi(countStr)
			m.form = m.buildPlayerNamesForm(count)
			cmds = append(cmds, m.form.Init())
		} else if m.game.Players[0] == nil {
			// Names entered
			count := len(m.game.Players)
			for i := 0; i < count; i++ {
				name := m.form.GetString(fmt.Sprintf("player_%d", i))
				m.game.Players[i] = model.NewPlayer(i+1, name)
			}
			m.finished = true
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *SetupModel) View() string {
	if m.finished {
		return "Setup Complete! Press Enter to start."
	}
	return m.form.View()
}

// Helpers

func (m *SetupModel) buildScriptSelectionForm() *huh.Form {
	// Find scripts
	files, _ := filepath.Glob("data/scripts/*.json")
	options := make([]huh.Option[string], len(files))
	for i, f := range files {
		options[i] = huh.NewOption(filepath.Base(f), f)
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("script_path").
				Options(options...).
				Title("Choose a Script"),
		),
	)
}

func (m *SetupModel) loadScript() {
	path := m.form.GetString("script_path")
	data, _ := os.ReadFile(path)
	var script model.Script
	json.Unmarshal(data, &script)
	m.game.Script = script
}

func (m *SetupModel) buildPlayerCountForm() *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("player_count").
				Title("How many players?").
				Validate(func(s string) error {
					i, err := strconv.Atoi(s)
					if err != nil {
						return fmt.Errorf("must be a number")
					}
					if i < 5 || i > 15 {
						return fmt.Errorf("must be between 5 and 15")
					}
					return nil
				}),
		),
	)
}

func (m *SetupModel) buildPlayerNamesForm(count int) *huh.Form {
	fields := make([]huh.Field, count)
	for i := 0; i < count; i++ {
		fields[i] = huh.NewInput().
			Key(fmt.Sprintf("player_%d", i)).
			Title(fmt.Sprintf("Player %d Name", i+1))
	}

	// Temporarily create players
	m.game.Players = make([]*model.Player, count)

	return huh.NewForm(
		huh.NewGroup(fields...),
	)
}
