package model

type RoleType string

const (
	Townsfolk RoleType = "Townsfolk"
	Outsider  RoleType = "Outsider"
	Minion    RoleType = "Minion"
	Demon     RoleType = "Demon"
	Traveler  RoleType = "Traveler"
)

type ActionType string

const (
	ActionNone         ActionType = "None"
	ActionSelectPlayer ActionType = "SelectPlayer"
	ActionSelectRole   ActionType = "SelectRole"
	ActionYesNo        ActionType = "YesNo"
	ActionInfoToken    ActionType = "InfoToken" // Select 2 players + 1 Role (e.g. Washerwoman)
)

type Role struct {
	Name       string     `json:"name"`
	Type       RoleType   `json:"type"`
	Ability    string     `json:"ability"`
	ActionType ActionType `json:"action_type"`
	Reminders  []string   `json:"reminders"`
}
