package model

type Player struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Role    Role   `json:"role"`
	IsAlive bool   `json:"is_alive"`

	// Game State
	UsedGhostVote        bool     `json:"used_ghost_vote"` // Has used their ghost vote?
	Reminders            []string `json:"reminders"`
	RegistrationOverride string   `json:"registration_override"` // "Townsfolk", "Outsider", "Minion", "Demon" or empty

	// Status Flags
	IsPoisoned   bool `json:"is_poisoned"`
	IsDrunk      bool `json:"is_drunk"`
	IsProtected  bool `json:"is_protected"`
	IsRedHerring bool `json:"is_red_herring"` // For Fortune Teller
}

func NewPlayer(id int, name string) *Player {
	return &Player{
		ID:      id,
		Name:    name,
		IsAlive: true,
	}
}

func (p *Player) ResetNightStatus() {
	p.IsPoisoned = false
	p.IsDrunk = false
	p.IsProtected = false
}
