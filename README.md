# Blood on the Clocktower Game Manager

A robust, CLI-based Storyteller helper for running *Blood on the Clocktower* games, built with Go and the [Charm](https://charm.sh) stack.

## ✨ Features

- **Setup Wizard**: Interactive, form-based setup for selecting scripts, player counts, and names (powered by `huh`).
- **The Grimoire**: A clean, responsive list view of the town square (powered by `bubbletea` & `lipgloss`).
    - **Status Tracking**: Toggle players between Alive/Dead states.
    - **Phase Management**: Switch between Day and Night phases.
    - **Registration Override**: Handle **Spy/Recluse** logic by overriding how a player registers to game effects (Townsfolk, Outsider, Minion, Demon).
- **Automated Night Phase**:
    - **Guided Walkthrough**: Steps through the night sequence based on the script and character order.
    - **Action Logic**: Handles Poisoner, Monk, Imp, etc., with automatic state updates.
    - **Fortune Teller**: Dedicated logic for Red Herrings and "Yes/No" signal generation (accounting for Poison/Drunk).
- **Resilience**:
    - **Auto-Save**: Game state persists to `game_state.json` on every action.
    - **Undo System**: Infinite generic undo stack (`u` key) to correct Storyteller mistakes.
- **Script Support**:
    - Includes *Trouble Brewing* out of the box.
    - Supports loading custom scripts via JSON.
- **Smart Logic**:
    - Correctly handles circular adjacency logic (skipping dead players for Empath/Chef checks).
    - **Malfunction Handling**: Automatically flags info as "False/Malfunction" in logs if the actor is Drunk or Poisoned.

## 🚀 Getting Started

### Prerequisites
- Go 1.21+ installed.

### Installation & Run
clone the repository and run:

```bash
go run main.go
```

Or build a binary:

```bash
go build -o clocktower
./clocktower
```

## 🎮 Controls

### Global / Overview
| Key | Action |
| :--- | :--- |
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `Enter` | Toggle Player Life/Death |
| `n` | Next Phase (Day/Night) |
| `u` | Undo last action |
| `e` | **Edit Mode** (Move players, Change roles) |
| `i` | View Role Info (Ability & Reminders) |
| `g` | Toggle **Ghost Vote** (Dead players only) |
| `R` | Cycle **Registration Override** (Spy/Recluse) |
| `q` | Quit |
| `Ctrl+n` | Wipe Game & Quit |

### Edit Mode (`e`)
| Key | Action |
| :--- | :--- |
| `Shift+K` | Move Player Up (Swap) |
| `Shift+J` | Move Player Down (Swap) |
| `Enter` / `r` | Change Role |
| `Esc` | Exit Edit Mode |

### Night Phase
| Key | Action |
| :--- | :--- |
| `Enter` | Confirm Action / Select Target |
| `→` / `l` | Skip / Next Step |
| `f` | Set **Red Herring** (Fortune Teller only) |
| `Esc` | Cancel / Back |

## 🛠️ Tech Stack

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**: The fun, functional One Model to Rule Them All.
- **[Huh?](https://github.com/charmbracelet/huh)**: Lightweight, accessible terminal forms.
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)**: Style definitions for nice terminal layouts.

## 📂 Project Structure

```
├── main.go           # Entry point
├── model/            # Game logic, state, and persistence
├── tui/              # UI components (Setup, Grimoire, Styles)
└── data/scripts/     # JSON script definitions
```
