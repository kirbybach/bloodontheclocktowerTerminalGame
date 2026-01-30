package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorPrimary   = lipgloss.Color("#7D56F4") // Purple
	ColorSecondary = lipgloss.Color("#EE6FF8") // Pink
	ColorText      = lipgloss.Color("#FAFAFA") // White
	ColorSubtext   = lipgloss.Color("#A1A1A1") // Gray
	ColorSuccess   = lipgloss.Color("#43BF6D") // Green
	ColorError     = lipgloss.Color("#E74C3C") // Red
	ColorGold      = lipgloss.Color("#F1C40F") // Gold for townsfolk/good
	ColorEvil      = lipgloss.Color("#E74C3C") // Red for evil

	// Role Specific Colors
	ColorTownsfolk = lipgloss.Color("#2980B9") // Blue
	ColorOutsider  = lipgloss.Color("#ECF0F1") // White
	ColorMinion    = lipgloss.Color("#E67E22") // Orange-Red (User asked for Red, but distinct from Demon?)
	// Actually user asked for Red for Minion, Dark Red for Demon.
	ColorMinionRed = lipgloss.Color("#E74C3C") // Red
	ColorDemonRed  = lipgloss.Color("#C0392B") // Dark Red

	// Grid Styles
	StyleGridHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(ColorSubtext).
			Padding(0, 1)

	StyleCell = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(ColorText)

	StyleSelected = lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderLeft(true).
			BorderForeground(ColorSecondary).
			Foreground(ColorSecondary).
			Bold(true)

	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorSubtext).
			MarginTop(1)
)
