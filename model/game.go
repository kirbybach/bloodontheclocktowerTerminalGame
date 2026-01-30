package model

import (
	"encoding/json"
	"fmt"
	"os"
)

type Phase string

const (
	PhaseSetup Phase = "Setup"
	PhaseDay   Phase = "Day"
	PhaseNight Phase = "Night"
)

type Script struct {
	Name       string   `json:"name"`
	Roles      []Role   `json:"roles"`
	FirstNight []string `json:"first_night"`
	OtherNight []string `json:"other_night"`
}

type GameSnapshot struct {
	Players []*Player `json:"players"`
	Phase   Phase     `json:"phase"`
	Log     []string  `json:"log"`
}

type Game struct {
	Players []*Player      `json:"players"`
	Phase   Phase          `json:"phase"`
	Script  Script         `json:"script"`
	Turn    int            `json:"turn"` // 1-indexed turn counter
	Log     []string       `json:"log"`
	History []GameSnapshot `json:"-"` // Do not persist history to avoid recursion/bloat
}

func NewGame() *Game {
	return &Game{
		Players: make([]*Player, 0),
		Phase:   PhaseSetup,
		History: make([]GameSnapshot, 0),
	}
}

// Logic: Neighbors

func (g *Game) GetNextLivingNeighbor(startIndex int, clockwise bool) *Player {
	n := len(g.Players)
	if n == 0 {
		return nil
	}

	step := 1
	if !clockwise {
		step = -1
	}

	curr := startIndex
	for i := 0; i < n; i++ { // Prevent infinite loop if everyone is dead (except maybe self)
		curr = (curr + step + n) % n
		if curr == startIndex {
			continue // Skip self if circle is full (though typically we want neighbor)
		}

		// If valid neighbor found
		if g.Players[curr] != nil && g.Players[curr].IsAlive {
			return g.Players[curr]
		}
	}

	// If no one else is alive, return nil or maybe self depending on rules,
	// but strictly "neighbor" usually implies another player.
	// For simple logic, return nil if no other living neighbor found.
	return nil
}

// Logic: Memento & Persistence

func (g *Game) Snapshot() {
	// Deep copy players
	playersCopy := make([]*Player, len(g.Players))
	for i, p := range g.Players {
		// Start with simple derefence copy
		val := *p
		// Copy slice fields if necessary
		reminders := make([]string, len(p.Reminders))
		copy(reminders, p.Reminders)
		val.Reminders = reminders
		playersCopy[i] = &val
	}

	// Copy log
	logCopy := make([]string, len(g.Log))
	copy(logCopy, g.Log)

	snap := GameSnapshot{
		Players: playersCopy,
		Phase:   g.Phase,
		Log:     logCopy,
	}

	g.History = append(g.History, snap)
}

func (g *Game) Undo() error {
	if len(g.History) == 0 {
		return fmt.Errorf("no history to undo")
	}

	last := g.History[len(g.History)-1]
	g.History = g.History[:len(g.History)-1]

	// Restore
	g.Players = last.Players
	g.Phase = last.Phase
	g.Log = last.Log

	// Auto-save after undo
	g.SaveState()
	return nil
}

func (g *Game) SaveState() error {
	// Simple JSON save
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("game_state.json", data, 0644)
}

func (g *Game) LoadState() error {
	data, err := os.ReadFile("game_state.json")
	if err != nil {
		return err
	}
	return json.Unmarshal(data, g)
}

// Logic: Setup

func GetDistribution(playerCount int) (townsfolk, outsider, minion, demon int) {
	switch playerCount {
	case 5:
		return 3, 0, 1, 1
	case 6:
		return 3, 1, 1, 1
	case 7:
		return 5, 0, 1, 1
	case 8:
		return 5, 1, 1, 1
	case 9:
		return 5, 2, 1, 1
	case 10:
		return 7, 0, 2, 1
	case 11:
		return 7, 1, 2, 1
	case 12:
		return 7, 2, 2, 1
	case 13:
		return 9, 0, 3, 1
	case 14:
		return 9, 1, 3, 1
	case 15:
		return 9, 2, 3, 1
	default:
		// Fallback for weird counts (should validate 5-15)
		if playerCount < 5 {
			return playerCount, 0, 0, 0
		}
		return 9, 2, 3, 1 // Capped at 15-ish logic
	}
}

// Logic: Night Resolution

func (g *Game) ResetNightChanges() {
	for _, p := range g.Players {
		p.ResetNightStatus()
	}
}

func (g *Game) ResolveNightAction(actorName string, target *Player) string {
	// Find actor
	var actor *Player
	for _, p := range g.Players {
		if p.Role.Name == actorName {
			actor = p
			break
		}
	}
	if actor == nil {
		return fmt.Sprintf("Error: Actor %s not found", actorName)
	}

	// Logic based on Role Name
	// Note: Strings should match JSON script exactly.
	switch actorName {
	case "Poisoner":
		target.IsPoisoned = true
		return fmt.Sprintf("Poisoner poisoned %s", target.Name)

	case "Monk":
		if actor.IsPoisoned || actor.IsDrunk {
			return fmt.Sprintf("Monk tried to protect %s but was malfunctioning", target.Name)
		}
		target.IsProtected = true
		return fmt.Sprintf("Monk protected %s", target.Name)

	case "Imp":
		// Check for malfunction
		if actor.IsPoisoned || actor.IsDrunk {
			return fmt.Sprintf("Imp attacked %s but was malfunctioning", target.Name)
		}

		// Check defense
		if target.IsProtected {
			return fmt.Sprintf("Imp attacked %s but they were protected!", target.Name)
		}
		if target.Role.Name == "Soldier" && !target.IsPoisoned && !target.IsDrunk {
			// Soldier cannot be killed by Demon
			return fmt.Sprintf("Imp attacked Soldier %s! No effect.", target.Name)
		}

		// Kill
		target.IsAlive = false
		return fmt.Sprintf("Imp killed %s!", target.Name)

	case "Fortune Teller":
		if actor.IsPoisoned || actor.IsDrunk {
			return fmt.Sprintf("Fortune Teller checked %s (False Info due to malfunction)", target.Name)
		}
		// Basic info logging, actual Yes/No given by storyteller manually usually,
		// but we can log that they checked.
		return fmt.Sprintf("Fortune Teller checked %s", target.Name)

	case "Butler":
		return fmt.Sprintf("Butler chose master %s", target.Name)

	case "Empath":
		// Empath is passive usually, but sometimes checked.
		// Usually no target selection for Empath needed in Night, they just get info.
		// But if supported:
		return "Empath checked neighbors"
	}

	return fmt.Sprintf("%s targeted %s", actorName, target.Name)
}

func (g *Game) ResolveInfoAction(actorName string, p1, p2 *Player, roleName string) string {
	// Find actor
	var actor *Player
	for _, p := range g.Players {
		if p.Role.Name == actorName {
			actor = p
			break
		}
	}
	if actor == nil {
		return fmt.Sprintf("Error: Actor %s not found", actorName)
	}

	// Check malfunction
	if actor.IsPoisoned || actor.IsDrunk {
		return fmt.Sprintf("%s learned that %s or %s is %s (False Info)", actorName, p1.Name, p2.Name, roleName)
	}

	return fmt.Sprintf("%s learned that %s or %s is %s", actorName, p1.Name, p2.Name, roleName)
}

func (g *Game) ResolveFortuneTeller(actor *Player, p1, p2 *Player) string {
	// Check malfunction
	if actor.IsPoisoned || actor.IsDrunk {
		// False info: The storyteller *could* lie, but usually a simple "NO" when it should be "YES" or vice versa is enough.
		// However, since we are automating:
		// Let's just flag it as unreliable.
		// Construct the true answer first.
		hasDemon := g.IsDemonOrRedHerring(p1) || g.IsDemonOrRedHerring(p2)
		result := "NO"
		if hasDemon {
			result = "YES"
		}
		return fmt.Sprintf("Fortune Teller checked %s & %s. Result: %s (FALSE - Is Drunk/Poisoned)", p1.Name, p2.Name, result)
	}

	hasDemon := g.IsDemonOrRedHerring(p1) || g.IsDemonOrRedHerring(p2)
	result := "NO"
	if hasDemon {
		result = "YES"
	}
	return fmt.Sprintf("Fortune Teller checked %s & %s. Result: %s", p1.Name, p2.Name, result)
}

func (g *Game) GetEmpathInfo(empath *Player) string {
	// Find Empath's index
	idx := -1
	for i, p := range g.Players {
		if p == empath {
			idx = i
			break
		}
	}
	if idx == -1 {
		return "Error: Empath not found"
	}

	// Get neighbors
	n1 := g.GetNextLivingNeighbor(idx, true)  // Clockwise
	n2 := g.GetNextLivingNeighbor(idx, false) // Counter-clockwise

	if n1 == nil || n2 == nil {
		return "Not enough neighbors"
	}

	// Calculate true count
	count := 0
	if n1.Role.Type == Minion || n1.Role.Type == Demon {
		count++
	}
	if n1 != n2 { // If only 2 players left, n1 == n2, don't double count?
		// Rules: "neighbors". If 2 players, neighbor is same.
		// Actually "2 alive neighbors". If total alive is 2 (Empath + 1), then neighbor is same.
		// Empath reads "how many of your 2 alive neighbors".
		// If only 2 players alive: Empath + A. A is both CW and CCW neighbor?
		// Usually considered 1 neighbor. But text says "2 alive neighbors".
		// If only 1 neighbor alive?
		// Standard ruling: If only 2 players alive, they effectively have 1 neighbor?
		// No, Blood on the Clocktower rules usually imply adjacent slots.
		// If 3 players: A (Empath) -> B -> C -> A. Neighbors are B and C.
		// If 2 players: A -> B -> A. Neighbor is B (CW) and B (CCW).
		// Rules say "neighbors". Usually implies distinct players?
		// Check wiki/rules if possible. Assuming standard logic: Distinct neighbors unless total < 3?
		// Let's count them individually if they are logically distinct directional neighbors.
		// But in 2 player game, n1 == n2. Is count 1 or 2?
		// "Empath: ...how many of your 2 alive neighbors..."
		// If n1 == n2, it's the same person. It should probably count once?
		// But let's assume standard >2 player game for simplicity.
		if n2.Role.Type == Minion || n2.Role.Type == Demon {
			count++
		}
	} else {
		// Only 1 neighbor total (2 players alive).
		// Is it 1 or 0?
		// Logic: You have two neighborhood slots. Both filled by same person?
		// Usually Empath gets a "0" or "1" in 2p?
		// Let's just implement distinct check.
	}

	// Malfunction
	if empath.IsPoisoned || empath.IsDrunk {
		// Return false info.
		// True info can be 0, 1, 2.
		// False info should be different.
		// Simple random false? Or predictable?
		// Let's just return a generic "Malfunction" string or try to give false number?
		// "Reading: X (FALSE)" is helpful for Storyteller.
		// Storyteller decides false info.
		return fmt.Sprintf("Reading: %d (FALSE - Is Drunk/Poisoned)", (count+1)%3)
	}

	return fmt.Sprintf("Reading: %d", count)
}

// Logic: Fortune Teller
func (g *Game) IsDemonOrRedHerring(p *Player) bool {
	if p == nil {
		return false
	}
	// Normal Demon check
	if p.Role.Type == Demon {
		return true
	}
	// Red Herring check
	if p.IsRedHerring {
		return true
	}
	// TODO: Handle Recluse registering as Evil/Demon if we get to that complexity
	return false
}

// Logic: Manual Edits

func (g *Game) SwapPlayers(i, j int) {
	if i < 0 || i >= len(g.Players) || j < 0 || j >= len(g.Players) {
		return
	}
	g.Players[i], g.Players[j] = g.Players[j], g.Players[i]
}

func (g *Game) SetPlayerRole(idx int, roleName string) error {
	if idx < 0 || idx >= len(g.Players) {
		return fmt.Errorf("invalid player index")
	}

	// Find role definition
	var newRole Role
	found := false
	for _, r := range g.Script.Roles {
		if r.Name == roleName {
			newRole = r
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("role %s not found in script", roleName)
	}

	// Persist status? or reset? usually reset or keep generic status.
	// For simplicity, keep status logic but update role data.
	// Important: Maintain ID/Name, change Role struct.
	g.Players[idx].Role = newRole
	return nil
}
