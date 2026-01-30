package tui

import (
	"clocktower/model"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type GrimoireState int

const (
	StateOverview GrimoireState = iota
	StateNightWalk
	StateNightSelect
	StateNightInfoSelect1
	StateNightInfoSelect2
	StateNightInfoRole
	StateNightInfoReveal
	StateNightFortuneRedHerring
	StateNightFortuneReveal
	StateEdit
	StateEditRoleSelect
	StateRoleInfo
)

type GrimoireModel struct {
	game       *model.Game
	cursor     int
	state      GrimoireState
	nightStep  int
	nightQueue []string // List of Role Names to wake up
	// Selection state
	selectCursor int
	selectedPID  int // Player ID being targeted
	// Info Token state
	infoP1     int
	infoP2     int
	roleList   []string // Filtered list of roles to select
	roleCursor int
	infoRole   string // Selected role for reveal
}

func NewGrimoireModel(game *model.Game) *GrimoireModel {
	return &GrimoireModel{
		game:  game,
		state: StateOverview,
	}
}

func (m *GrimoireModel) Init() tea.Cmd {
	return nil
}

func (m *GrimoireModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Guard
	if len(m.game.Players) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Dispatch based on state
		switch m.state {
		case StateNightWalk:
			return m.updateNightWalk(msg)
		case StateNightSelect:
			return m.updateNightSelect(msg)
		case StateNightInfoSelect1:
			return m.updateNightInfo1(msg)
		case StateNightInfoSelect2:
			return m.updateNightInfo2(msg)
		case StateNightInfoRole:
			return m.updateNightInfoRole(msg)
		case StateNightInfoReveal:
			return m.updateNightInfoReveal(msg)
		case StateNightFortuneRedHerring:
			return m.updateNightFortuneRedHerring(msg)
		case StateNightFortuneReveal:
			return m.updateNightFortuneReveal(msg)
		case StateEdit:
			return m.updateEdit(msg)
		case StateEditRoleSelect:
			return m.updateEditRoleSelect(msg)
		case StateRoleInfo:
			return m.updateRoleInfo(msg)
		default:
			return m.updateOverview(msg)
		}
	}
	return m, nil
}

func (m *GrimoireModel) updateOverview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Ensure cursor is valid
	if m.cursor >= len(m.game.Players) {
		m.cursor = len(m.game.Players) - 1
	}

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.game.Players)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.game.Players) > 0 {
			m.game.Players[m.cursor].IsAlive = !m.game.Players[m.cursor].IsAlive
			m.game.SaveState()
		}
	case "n":
		if m.game.Phase == model.PhaseDay {
			// Start Night Sequence
			m.game.Phase = model.PhaseNight
			m.game.Turn++
			m.startNight()
		} else {
			m.game.Phase = model.PhaseDay
		}
		m.game.SaveState()
	case "q":
		return m, tea.Quit
	case "ctrl+n":
		return m, func() tea.Msg { return ResetGameMsg{} }
	case "u":
		m.game.Undo()
	case "e":
		m.state = StateEdit
	case "i":
		m.state = StateRoleInfo
	}
	return m, nil
}

func (m *GrimoireModel) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "e", "esc", "q":
		m.state = StateOverview
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.game.Players)-1 {
			m.cursor++
		}
	case "K": // Shift+k: Move Up
		if m.cursor > 0 {
			m.game.SwapPlayers(m.cursor, m.cursor-1)
			m.cursor--
			m.game.SaveState()
		}
	case "J": // Shift+j: Move Down
		if m.cursor < len(m.game.Players)-1 {
			m.game.SwapPlayers(m.cursor, m.cursor+1)
			m.cursor++
			m.game.SaveState()
		}
	case "enter", "r":
		m.state = StateEditRoleSelect
		// Show all roles
		m.prepareAllRolesList()
		m.roleCursor = 0
	}
	return m, nil
}

func (m *GrimoireModel) updateEditRoleSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.roleCursor > 0 {
			m.roleCursor--
		}
	case "down", "j":
		if m.roleCursor < len(m.roleList)-1 {
			m.roleCursor++
		}
	case "enter":
		selectedRole := m.roleList[m.roleCursor]
		m.game.SetPlayerRole(m.cursor, selectedRole)
		m.game.SaveState()
		m.state = StateEdit
	case "esc":
		m.state = StateEdit
	}
	return m, nil
}

func (m *GrimoireModel) updateRoleInfo(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q", "i":
		m.state = StateOverview
	}
	return m, nil
}

func (m *GrimoireModel) startNight() {
	// Reset transient night flags (poison/protection) at start of night
	m.game.ResetNightChanges()

	m.state = StateNightWalk
	m.nightStep = 0

	// Determine which list to use
	list := m.game.Script.OtherNight

	// If Turn is 1 (First Night), use FirstNight list
	// Assuming Turn starts at 0 or 1. Let's make sure we increment it.
	if m.game.Turn <= 1 {
		list = m.game.Script.FirstNight
	}

	// Filter queue: Only include roles that are actually in play
	var activeQueue []string
	for _, roleName := range list {
		// Check if any player has this role
		for _, p := range m.game.Players {
			if p != nil && p.Role.Name == roleName {
				activeQueue = append(activeQueue, roleName)
				break
			}
		}
	}
	m.nightQueue = activeQueue
}

func (m *GrimoireModel) updateNightWalk(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Guard against empty queue or out of bounds
		if len(m.nightQueue) == 0 || m.nightStep >= len(m.nightQueue) {
			m.state = StateOverview
			return m, nil
		}

		// Check if current role has an action that requires selection
		roleName := m.nightQueue[m.nightStep]
		// Find player role
		var currentRole model.Role
		for _, p := range m.game.Players {
			if p.Role.Name == roleName {
				currentRole = p.Role
				break
			}
		}

		// If action required, go to selection
		if currentRole.Name == "Fortune Teller" {
			// Special Logic: Check if Red Herring is set
			redHerringSet := false
			for _, p := range m.game.Players {
				if p.IsRedHerring {
					redHerringSet = true
					break
				}
			}

			if !redHerringSet {
				// Force Red Herring Selection
				m.state = StateNightFortuneRedHerring
				m.selectCursor = 0
				return m, nil
			}

			// Proceed to standard selection (2 players)
			m.state = StateNightInfoSelect1
			m.selectCursor = 0
			return m, nil
		} else if currentRole.ActionType == model.ActionSelectPlayer {
			m.state = StateNightSelect
			m.selectCursor = 0
			return m, nil
		} else if currentRole.ActionType == model.ActionInfoToken {
			m.state = StateNightInfoSelect1
			m.selectCursor = 0
			return m, nil
		}

		// Otherwise, just advance
		m.nextStep()

	case "right", "l":
		m.nextStep()
	case "f":
		// Feature: Set Red Herring for Fortune Teller
		// Only valid if current role is Fortune Teller
		roleName := m.nightQueue[m.nightStep]
		if roleName == "Fortune Teller" {
			// Trigger a mode to select Red Herring?
			// Or just reuse NightSelect but with a special flag?
			// Simpler: Just allow editing Red Herring logic via a specific state?
			// Let's reuse StateNightSelect, but we need to know WHY we are selecting.
			// Adding a new state might be cleaner.
			m.state = StateNightFortuneRedHerring
			m.selectCursor = 0
		}
	case "esc":
		m.state = StateOverview
	}
	return m, nil
}

func (m *GrimoireModel) nextStep() {
	m.nightStep++
	if m.nightStep >= len(m.nightQueue) {
		m.state = StateOverview
	}
}

func (m *GrimoireModel) updateNightSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectCursor > 0 {
			m.selectCursor--
		}
	case "down", "j":
		if m.selectCursor < len(m.game.Players)-1 {
			m.selectCursor++
		}
	case "enter":
		// Confirm selection
		target := m.game.Players[m.selectCursor]
		actorName := m.nightQueue[m.nightStep]

		// Execute Logic
		resultMsg := m.game.ResolveNightAction(actorName, target)

		// Log action
		m.game.Log = append(m.game.Log, fmt.Sprintf("[Night] %s", resultMsg))
		m.game.SaveState()

		// Return to walk and advance
		m.state = StateNightWalk
		m.nextStep()

	case "esc":
		// Cancel selection
		m.state = StateNightWalk
	}
	return m, nil
}

func (m *GrimoireModel) updateNightInfo1(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectCursor > 0 {
			m.selectCursor--
		}
	case "down", "j":
		if m.selectCursor < len(m.game.Players)-1 {
			m.selectCursor++
		}
	case "enter":
		m.infoP1 = m.selectCursor
		m.state = StateNightInfoSelect2
		m.selectCursor = 0 // Reset for next selection
	case "esc":
		m.state = StateNightWalk
	}
	return m, nil
}

func (m *GrimoireModel) updateNightInfo2(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectCursor > 0 {
			m.selectCursor--
		}
	case "down", "j":
		if m.selectCursor < len(m.game.Players)-1 {
			m.selectCursor++
		}
	case "enter":
		m.infoP2 = m.selectCursor

		// Check if Fortune Teller
		actor := m.nightQueue[m.nightStep]
		if actor == "Fortune Teller" {
			m.state = StateNightFortuneReveal
		} else {
			m.state = StateNightInfoRole
			m.prepareRoleList()
			m.roleCursor = 0
		}
	case "esc":
		m.state = StateNightWalk
	}
	return m, nil
}

func (m *GrimoireModel) prepareRoleList() {
	actorName := m.nightQueue[m.nightStep]
	var targetType model.RoleType

	// Determine logic based on actor ability
	// Washerwoman -> Townsfolk
	// Librarian -> Outsider
	// Investigator -> Minion
	switch actorName {
	case "Washerwoman":
		targetType = model.Townsfolk
	case "Librarian":
		targetType = model.Outsider
	case "Investigator":
		targetType = model.Minion
	default:
		// Fallback: Show all roles? Or maybe none?
		// If custom script uses InfoToken, we might need configuration.
		// For now, show all.
		targetType = ""
	}

	// Filter logic:
	// User requested: "the roles it shows has to be of the townsfolk selected"
	// We prioritize the roles of the selected players (infoP1, infoP2) if they match the target type.

	candidates := make(map[string]bool)
	p1 := m.game.Players[m.infoP1]
	p2 := m.game.Players[m.infoP2]

	if targetType == "" {
		// Fallback/Default: All roles
		for _, r := range m.game.Script.Roles {
			m.roleList = append(m.roleList, r.Name)
		}
		return
	}

	// Check P1
	if p1.Role.Type == targetType {
		candidates[p1.Role.Name] = true
	}
	// Check P2
	if p2.Role.Type == targetType {
		candidates[p2.Role.Name] = true
	}

	// Construct list
	var list []string
	if len(candidates) > 0 {
		for name := range candidates {
			list = append(list, name)
		}
	} else {
		// Fallback: If neither player matches (e.g. Bluffing/Drunk/Spy interactions or user error),
		// show all compatible roles from script so they aren't stuck.
		for _, r := range m.game.Script.Roles {
			if r.Type == targetType {
				list = append(list, r.Name)
			}
		}
	}
	m.roleList = list
}

func (m *GrimoireModel) prepareAllRolesList() {
	var list []string
	for _, r := range m.game.Script.Roles {
		list = append(list, r.Name)
	}
	m.roleList = list
}

func (m *GrimoireModel) updateNightInfoRole(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.roleCursor > 0 {
			m.roleCursor--
		}
	case "down", "j":
		if m.roleCursor < len(m.roleList)-1 {
			m.roleCursor++
		}
	case "enter":
		// Confirm
		m.infoRole = m.roleList[m.roleCursor]
		m.state = StateNightInfoReveal
	case "esc":
		m.state = StateNightWalk
	}
	return m, nil
}

func (m *GrimoireModel) updateNightInfoReveal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Execute Logic
		actorName := m.nightQueue[m.nightStep]
		p1 := m.game.Players[m.infoP1]
		p2 := m.game.Players[m.infoP2]
		roleName := m.infoRole

		resultMsg := m.game.ResolveInfoAction(actorName, p1, p2, roleName)
		m.game.Log = append(m.game.Log, fmt.Sprintf("[Night] %s", resultMsg))
		m.game.SaveState()

		m.state = StateNightWalk
		m.nextStep()
	case "esc":
		// Go back to role selection
		m.state = StateNightInfoRole
	}
	return m, nil
}

func (m *GrimoireModel) updateNightFortuneRedHerring(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectCursor > 0 {
			m.selectCursor--
		}
	case "down", "j":
		if m.selectCursor < len(m.game.Players)-1 {
			m.selectCursor++
		}
	case "enter":
		// Set Red Herring
		// Logic: Clear previous, set new
		for _, p := range m.game.Players {
			p.IsRedHerring = false
		}
		target := m.game.Players[m.selectCursor]
		target.IsRedHerring = true

		m.game.Log = append(m.game.Log, fmt.Sprintf("[Setup] Fortune Teller Red Herring set to %s", target.Name))
		m.game.SaveState()

		m.state = StateNightWalk
	case "esc":
		m.state = StateNightWalk
	}
	return m, nil
}

func (m *GrimoireModel) updateNightFortuneReveal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Commit to log
		actorName := m.nightQueue[m.nightStep]
		p1 := m.game.Players[m.infoP1]
		p2 := m.game.Players[m.infoP2]

		// Find actor logic re-implemented or helper needed?
		// We need the actor *Player* object for ResolveFortuneTeller to check poison/drunk
		var actor *model.Player
		for _, p := range m.game.Players {
			if p.Role.Name == actorName {
				actor = p
				break
			}
		}

		resultMsg := "Error: Actor not found"
		if actor != nil {
			resultMsg = m.game.ResolveFortuneTeller(actor, p1, p2)
		}

		m.game.Log = append(m.game.Log, fmt.Sprintf("[Night] %s", resultMsg))
		m.game.SaveState()

		m.state = StateNightWalk
		m.nextStep()
	case "esc":
		// Back to second selection
		m.state = StateNightInfoSelect2
	}
	return m, nil
}

func (m *GrimoireModel) View() string {
	switch m.state {
	case StateNightWalk:
		return m.viewNightWalk()
	case StateNightSelect:
		return m.viewNightSelect()
	case StateNightInfoSelect1:
		return m.viewNightInfoSelect1()
	case StateNightInfoSelect2:
		return m.viewNightInfoSelect2()
	case StateNightInfoRole:
		return m.viewNightInfoRole()
	case StateNightInfoReveal:
		return m.viewNightInfoReveal()
	case StateNightFortuneRedHerring:
		return m.viewNightFortuneRedHerring()
	case StateNightFortuneReveal:
		return m.viewNightFortuneReveal()
	case StateEdit:
		return m.viewEdit()
	case StateEditRoleSelect:
		return m.viewEditRoleSelect()
	case StateRoleInfo:
		return m.viewRoleInfo()
	}
	return m.viewOverview()
}

func (m *GrimoireModel) viewRoleInfo() string {
	s := strings.Builder{}

	// Get selected player (fallback safety)
	if m.cursor < 0 || m.cursor >= len(m.game.Players) {
		return "No player selected."
	}
	p := m.game.Players[m.cursor]
	r := p.Role

	s.WriteString(StyleGridHeader.Render(" ROLE INFO ") + "\n\n")

	s.WriteString(fmt.Sprintf("Name:      %s\n", styleRole(r.Name, r.Type)))
	s.WriteString(fmt.Sprintf("Type:      %s\n", styleRoleType(r.Type)))
	s.WriteString(fmt.Sprintf("\nAbility:\n%s\n", r.Ability))

	if len(r.Reminders) > 0 {
		s.WriteString(fmt.Sprintf("\nReminders: %v\n", r.Reminders))
	}

	s.WriteString("\n\n(Esc) Back")
	return s.String()
}

func (m *GrimoireModel) viewEdit() string {
	s := strings.Builder{}
	s.WriteString(StyleGridHeader.Render(" EDIT MODE ") + "\n\n")

	// Table header
	s.WriteString(fmt.Sprintf("%-3s | %-12s | %-15s | %-10s\n", "#", "Name", "Role", "Type"))
	s.WriteString(strings.Repeat("-", 60) + "\n")

	for i, p := range m.game.Players {
		if p == nil {
			continue
		}
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		if m.cursor == i {
			rName := fmt.Sprintf("%-15s", p.Role.Name)
			rType := fmt.Sprintf("%-10s", p.Role.Type)

			coloredRole := styleRole(rName, p.Role.Type)
			coloredType := styleRole(rType, p.Role.Type)

			row := fmt.Sprintf("%s %-3d | %-12s | %s | %s",
				cursor, p.ID, p.Name, coloredRole, coloredType)
			s.WriteString(StyleSelected.Render(row) + "\n")
		} else {
			rName := fmt.Sprintf("%-15s", p.Role.Name)
			rType := fmt.Sprintf("%-10s", p.Role.Type)

			coloredRole := styleRole(rName, p.Role.Type)
			coloredType := styleRole(rType, p.Role.Type)

			row := fmt.Sprintf("%s %-3d | %-12s | %s | %s",
				cursor, p.ID, p.Name, coloredRole, coloredType)
			s.WriteString(StyleCell.Render(row) + "\n")
		}
	}

	s.WriteString("\n\n(e/Esc) Exit • (K/J) Move Up/Down • (Enter) Change Role")
	return s.String()
}

func (m *GrimoireModel) viewEditRoleSelect() string {
	s := strings.Builder{}
	s.WriteString(StyleGridHeader.Render(" SELECT NEW ROLE ") + "\n\n")

	for i, r := range m.roleList {
		cursor := " "
		if m.roleCursor == i {
			cursor = ">"
		}

		// Find role type for coloring
		var rType model.RoleType
		for _, def := range m.game.Script.Roles {
			if def.Name == r {
				rType = def.Type
				break
			}
		}

		line := fmt.Sprintf("%s %s", cursor, styleRole(r, rType))
		if m.roleCursor == i {
			s.WriteString(StyleSelected.Render(line) + "\n")
		} else {
			s.WriteString(StyleCell.Render(line) + "\n")
		}
	}
	s.WriteString("\n(Enter) Confirm • (Esc) Cancel")
	return s.String()
}

func (m *GrimoireModel) viewNightInfoReveal() string {
	s := strings.Builder{}
	actor := m.nightQueue[m.nightStep]
	p1 := m.game.Players[m.infoP1]
	p2 := m.game.Players[m.infoP2]
	role := m.infoRole

	// Determine role type for styling
	var rType model.RoleType
	for _, r := range m.game.Script.Roles {
		if r.Name == role {
			rType = r.Type
			break
		}
	}

	s.WriteString(StyleGridHeader.Render(" CONFIRM INFORMATION ") + "\n\n")
	s.WriteString(fmt.Sprintf("%s learns that:\n\n", strings.ToUpper(actor)))

	line := fmt.Sprintf("%s OR %s is the %s",
		p1.Name,
		p2.Name,
		styleRole(role, rType),
	)

	s.WriteString(StyleSelected.Render(line) + "\n\n")
	s.WriteString("(Enter) Confirm & Log • (Esc) Back")
	return s.String()
}

func (m *GrimoireModel) viewNightFortuneRedHerring() string {
	s := strings.Builder{}
	s.WriteString(StyleGridHeader.Render(" SELECT RED HERRING ") + "\n\n")

	// Reuse selection list but maybe show who is currently Red Herring?
	for i, p := range m.game.Players {
		cursor := " "
		if m.selectCursor == i {
			cursor = ">"
		}

		mark := ""
		if p.IsRedHerring {
			mark = " [CURRENT]"
		}

		line := fmt.Sprintf("%s %-12s | %-12s%s", cursor, p.Name, p.Role.Name, mark)
		if m.selectCursor == i {
			s.WriteString(StyleSelected.Render(line) + "\n")
		} else {
			s.WriteString(StyleCell.Render(line) + "\n")
		}
	}
	s.WriteString("\n(Enter) Set Red Herring • (Esc) Cancel")
	return s.String()
}

func (m *GrimoireModel) viewNightFortuneReveal() string {
	s := strings.Builder{}
	actor := m.nightQueue[m.nightStep]
	p1 := m.game.Players[m.infoP1]
	p2 := m.game.Players[m.infoP2]

	// Calculate result purely for display (logic repeated in update, harmless)
	// We need actor player object
	var actorPlayer *model.Player
	for _, p := range m.game.Players {
		if p.Role.Name == actor {
			actorPlayer = p
			break
		}
	}

	result := "ERROR"
	details := ""
	if actorPlayer != nil {
		hasDemon := m.game.IsDemonOrRedHerring(p1) || m.game.IsDemonOrRedHerring(p2)
		if hasDemon {
			result = "YES"
		} else {
			result = "NO"
		}

		// Malfunction check for UI warning
		if actorPlayer.IsPoisoned || actorPlayer.IsDrunk {
			details = " (MALFUNCTION - LIED?)"
		}
	}

	s.WriteString(StyleGridHeader.Render(" FORTUNE TELLER RESULT ") + "\n\n")
	s.WriteString(fmt.Sprintf("%s checks %s and %s\n\n", strings.ToUpper(actor), p1.Name, p2.Name))

	resStyle := lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Padding(1, 2).Border(lipgloss.RoundedBorder())
	if result == "YES" {
		resStyle = resStyle.Foreground(ColorDemonRed).BorderForeground(ColorDemonRed)
	}

	s.WriteString(resStyle.Render(result) + details + "\n\n")

	s.WriteString("(Enter) Confirm & Log • (Esc) Back")
	return s.String()
}

func (m *GrimoireModel) viewNightInfoSelect1() string {
	s := strings.Builder{}
	actor := m.nightQueue[m.nightStep]
	s.WriteString(StyleGridHeader.Render(" SELECT PLAYER 1 for "+strings.ToUpper(actor)) + "\n\n")
	s.WriteString(m.renderPlayerSelectionList())
	s.WriteString("\n(Enter) Confirm 1st Target • (Esc) Cancel")
	return s.String()
}

func (m *GrimoireModel) viewNightInfoSelect2() string {
	s := strings.Builder{}
	actor := m.nightQueue[m.nightStep]
	s.WriteString(StyleGridHeader.Render(" SELECT PLAYER 2 for "+strings.ToUpper(actor)) + "\n\n")

	// Show list but maybe highlight the first selection?
	// Re-using renderPlayerSelectionList for consistency, but distinct cursor.
	s.WriteString(m.renderPlayerSelectionList())
	s.WriteString("\n(Enter) Confirm 2nd Target • (Esc) Cancel")
	return s.String()
}

func (m *GrimoireModel) viewNightInfoRole() string {
	s := strings.Builder{}
	actor := m.nightQueue[m.nightStep]
	s.WriteString(StyleGridHeader.Render(" SELECT ROLE for "+strings.ToUpper(actor)) + "\n\n")

	for i, r := range m.roleList {
		cursor := " "
		if m.roleCursor == i {
			cursor = ">"
		}

		// Find role type for coloring
		var rType model.RoleType
		for _, def := range m.game.Script.Roles {
			if def.Name == r {
				rType = def.Type
				break
			}
		}

		line := fmt.Sprintf("%s %s", cursor, styleRole(r, rType))
		if m.roleCursor == i {
			s.WriteString(StyleSelected.Render(line) + "\n")
		} else {
			s.WriteString(StyleCell.Render(line) + "\n")
		}
	}
	s.WriteString("\n(Enter) Confirm Info • (Esc) Cancel")
	return s.String()
}

func (m *GrimoireModel) renderPlayerSelectionList() string {
	s := strings.Builder{}
	for i, p := range m.game.Players {
		cursor := " "
		if m.selectCursor == i {
			cursor = ">"
		}

		// Mark previously selected
		mark := ""
		if m.state == StateNightInfoSelect2 && m.infoP1 == i {
			mark = " [1]"
		}

		line := fmt.Sprintf("%s %-12s | %-12s%s", cursor, p.Name, styleRole(p.Role.Name, p.Role.Type), mark)
		if m.selectCursor == i {
			s.WriteString(StyleSelected.Render(line) + "\n")
		} else {
			s.WriteString(StyleCell.Render(line) + "\n")
		}
	}
	return s.String()
}

func (m *GrimoireModel) viewNightWalk() string {
	if m.nightStep >= len(m.nightQueue) {
		return "Night ends... Press Enter."
	}

	roleName := m.nightQueue[m.nightStep]

	// Find player with this role
	var player *model.Player
	for _, p := range m.game.Players {
		if p != nil && p.Role.Name == roleName {
			player = p
			break
		}
	}

	s := strings.Builder{}
	s.WriteString(StyleGridHeader.Render(" NIGHT PHASE ") + "\n\n")

	// Show Grimoire Table for context (dimmed/compact?)
	s.WriteString(m.renderGrimoireTable())
	s.WriteString("\n" + strings.Repeat("=", 80) + "\n\n")

	s.WriteString(fmt.Sprintf("Step %d/%d:  %s\n\n", m.nightStep+1, len(m.nightQueue), strings.ToUpper(roleName)))

	if player != nil {
		status := "Alive"
		if !player.IsAlive {
			status = "DEAD"
		}

		// Status Effects
		effects := ""
		if player.IsPoisoned {
			effects += " ☠️ POISONED"
		}
		if player.IsDrunk {
			effects += " 🍺 DRUNK"
		}
		if player.IsProtected {
			effects += " 🛡️ PROTECTED"
		}

		// Team
		team := "GOOD"
		if player.Role.Type == model.Minion || player.Role.Type == model.Demon {
			team = "EVIL"
		}

		s.WriteString(fmt.Sprintf("Player: %s\n", player.Name))
		s.WriteString(fmt.Sprintf("Status: %s%s\n", status, effects))
		s.WriteString(fmt.Sprintf("Team:   %s (%s)\n", team, player.Role.Type))
		s.WriteString(fmt.Sprintf("Ability: %s\n\n", player.Role.Ability))
		if len(player.Role.Reminders) > 0 {
			s.WriteString(fmt.Sprintf("Reminders: %v\n", player.Role.Reminders))
		}

		// Empath Logic
		if player.Role.Name == "Empath" {
			info := m.game.GetEmpathInfo(player)
			s.WriteString(fmt.Sprintf("\n[Empath Info]\n%s\n", info))
		}

		s.WriteString("\n[Action Required]\n")

		if player.Role.ActionType == model.ActionSelectPlayer {
			s.WriteString("(Press Enter to select a target player)")
		} else {
			s.WriteString("Perform action physically. Press Enter to continue.")
		}

	} else {
		s.WriteString("(Role not in play. Skip?)")
	}

	s.WriteString("\n\n(Enter) Next • (Esc) Skip Night")
	return s.String()
}

func (m *GrimoireModel) viewNightSelect() string {
	s := strings.Builder{}
	actor := m.nightQueue[m.nightStep]
	s.WriteString(StyleGridHeader.Render(" SELECT TARGET for "+strings.ToUpper(actor)) + "\n\n")

	// Show list of potential targets (all players)
	for i, p := range m.game.Players {
		cursor := " "
		if m.selectCursor == i {
			cursor = ">"
		}

		line := fmt.Sprintf("%s %-12s | %-12s", cursor, p.Name, p.Role.Name)
		if m.selectCursor == i {
			s.WriteString(StyleSelected.Render(line) + "\n")
		} else {
			s.WriteString(StyleCell.Render(line) + "\n")
		}
	}

	s.WriteString("\n(Enter) Confirm Target • (Esc) Cancel")
	return s.String()
}

func (m *GrimoireModel) viewOverview() string {
	s := strings.Builder{}
	s.WriteString(StyleGridHeader.Render(fmt.Sprintf("Phase: %s", m.game.Phase)) + "\n\n")

	s.WriteString(m.renderGrimoireTable())
	s.WriteString("\n\n(j/k) Move • (e) Edit • (i) Info • (enter) Toggle Life • (n) Next Phase • (u) Undo • (q) Quit • (Ctrl+n) Wipe & Quit")
	return s.String()
}

func (m *GrimoireModel) renderGrimoireTable() string {
	s := strings.Builder{}
	// Table header
	s.WriteString(fmt.Sprintf("%-3s | %-12s | %-15s | %-10s | %-8s | %-10s\n", "#", "Name", "Role", "Type", "Status", "Effects"))
	s.WriteString(strings.Repeat("-", 80) + "\n")

	for i, p := range m.game.Players {
		if p == nil {
			continue
		}
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		status := "ALIVE"
		if !p.IsAlive {
			status = "DEAD"
		}

		// Status Effects
		effects := ""
		if p.IsPoisoned {
			effects += "☠️ "
		}
		if p.IsDrunk {
			effects += "🍺 "
		}
		if p.IsProtected {
			effects += "🛡️ "
		}
		// Red Herring
		if p.IsRedHerring {
			effects += "🚩 "
		}

		// Check selection
		isSelected := m.cursor == i

		// Role Coloring
		rName := fmt.Sprintf("%-15s", p.Role.Name)
		rType := fmt.Sprintf("%-10s", p.Role.Type)

		coloredRole := styleRole(rName, p.Role.Type)
		coloredType := styleRole(rType, p.Role.Type)

		row := fmt.Sprintf("%s %-3d | %-12s | %s | %s | %-8s | %-10s",
			cursor, p.ID, p.Name, coloredRole, coloredType, status, effects)

		if isSelected {
			s.WriteString(StyleSelected.Render(row) + "\n")
		} else {
			s.WriteString(StyleCell.Render(row) + "\n")
		}
	}
	return s.String()
}

func styleRole(name string, roleType model.RoleType) string {
	style := lipgloss.NewStyle()
	switch roleType {
	case model.Townsfolk:
		style = style.Foreground(ColorTownsfolk)
	case model.Outsider:
		style = style.Foreground(ColorOutsider)
	case model.Minion:
		style = style.Foreground(ColorMinionRed)
	case model.Demon:
		style = style.Foreground(ColorDemonRed)
	}
	return style.Render(name)
}

func styleRoleType(t model.RoleType) string {
	str := string(t)
	return styleRole(str, t)
}
