package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

//go:embed all-descriptions.json
var descriptionsJSON []byte

const splash = "  _      __ __       _        ______ __         __ \n | | /| / // /  ___ |/ ___   /_  __// /  ___ _ / /_\n | |/ |/ // _ \\/ _ \\  (_-<    / /  / _ \\/ _ `// __/\n |__/|__//_//_/\\___/ /___/   /_/  /_//_/\\_,_/ \\__/ \n   ___         __                         ___   __ \n  / _ \\ ___   / /__ ___  __ _  ___   ___ /__ \\ / / \n / ___// _ \\ /  '_// -_)/  ' \\/ _ \\ / _ \\ /__//_/  \n/_/    \\___//_/\\_\\ \\__//_/_/_/\\___//_//_/(_) (_)   \n                                                   \n"

type UnFlashFieldMsg string
type PenaltyExpiredMsg struct{}
type TickMsg struct{}

const SCREEN_MENU = 0
const SCREEN_ENDLESS = 1
const SCREEN_TIMED = 2
const SCREEN_GAME_OVER = 3

const GAMETYPE_ENDLESS = 0
const GAMETYPE_TIMED = 1

type Pokemon struct {
	Id          int      `json:"id"`
	Name        string   `json:"name"`
	Description []string `json:"description"`
}

func main() {
	p := tea.NewProgram(createModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func createModel() model {
	var pokes []Pokemon
	if err := json.Unmarshal(descriptionsJSON, &pokes); err != nil {
		panic(err)
	}

	input := textinput.New()
	input.Focus()
	input.CharLimit = 50
	input.SetWidth(50)
	input.SetVirtualCursor(true)
	s := input.Styles()
	s.Cursor.Blink = false
	input.SetStyles(s)

	currentPoke := rand.Intn(len(pokes))
	currentDesc := rand.Intn(len(pokes[currentPoke].Description))

	return model{pokes: pokes, textInput: input, currentPoke: currentPoke, currentDesc: currentDesc, flashingFields: make(map[string]int)}
}

type model struct {
	currentPoke      int
	currentDesc      int
	pokes            []Pokemon
	textInput        textinput.Model
	width            int
	height           int
	skips            int
	misses           int
	time             int
	flashingFields   map[string]int
	penalties        int
	currentPenalty   int
	gameTypeSelector int
	screen           int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			switch m.screen {
			case SCREEN_MENU:
				return m, tea.Quit
			default:
				m.screen = SCREEN_MENU
			}
		case "enter":
			m, cmds = m.handleEnterKey()
		case "tab":
			m, cmds = m.handleTabKey()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case UnFlashFieldMsg:
		current := m.flashingFields[string(msg)]
		m.flashingFields[string(msg)] = max(0, current-1)
	case PenaltyExpiredMsg:
		m.penalties--
		if m.penalties < 1 {
			m.currentPenalty = 0
		}
	case TickMsg:
		if m.screen == SCREEN_TIMED {
			m.time = max(0, m.time-1)
			if m.time < 1 {
				m.screen = SCREEN_GAME_OVER
			} else {
				cmds = append(cmds, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
					return TickMsg{}
				}))
			}
		}
	}

	var tcmd tea.Cmd
	m.textInput, tcmd = m.textInput.Update(msg)
	cmds = append(cmds, tcmd)

	return m, tea.Batch(cmds...)
}

func (m model) handleEnterKey() (model, []tea.Cmd) {
	var cmds []tea.Cmd
	switch m.screen {
	case SCREEN_MENU:
		m.pokes = fetchPokes()
		m.misses = 0
		m.skips = 0
		m = m.selectRandomPoke()
		m.textInput.Reset()
		if m.gameTypeSelector == GAMETYPE_TIMED {
			m.time = 60
			cmds = append(cmds, func() tea.Msg { return TickMsg{} })
			m.screen = SCREEN_TIMED
		} else {
			m.screen = SCREEN_ENDLESS
		}
	case SCREEN_ENDLESS, SCREEN_TIMED:
		if strings.EqualFold(m.pokes[m.currentPoke].Name, m.textInput.Value()) {
			cmds = append(cmds, m.flashField("score"))
			m.pokes = append(m.pokes[:m.currentPoke], m.pokes[m.currentPoke+1:]...)
			m = m.selectRandomPoke()
			m.textInput.Reset()
		} else {
			cmds = append(cmds, m.flashField("misses"))
			m.misses++
		}
	default:
		// no-op
	}

	return m, cmds
}

func (m model) handleTabKey() (model, []tea.Cmd) {
	var cmds []tea.Cmd
	switch m.screen {
	case SCREEN_ENDLESS, SCREEN_TIMED:
		m.textInput.Reset()
		cmds = append(cmds, m.flashField("skips"))
		m.currentPoke = rand.Intn(len(m.pokes))
		m.currentDesc = rand.Intn(len(m.pokes[m.currentPoke].Description))
		m.skips++
		if m.screen == SCREEN_TIMED {
			newM, cmd := m.skipPenalty()
			m = newM
			cmds = append(cmds, cmd)
		}
	case SCREEN_MENU:
		if m.gameTypeSelector == GAMETYPE_ENDLESS {
			m.gameTypeSelector = GAMETYPE_TIMED
		} else {
			m.gameTypeSelector = GAMETYPE_ENDLESS
		}
	default:
		// no-op
	}

	return m, cmds
}

func (m model) selectRandomPoke() model {
	m.currentPoke = rand.Intn(len(m.pokes))
	m.currentDesc = rand.Intn(len(m.pokes[m.currentPoke].Description))
	return m
}

func fetchPokes() []Pokemon {
	var pokes []Pokemon
	if err := json.Unmarshal(descriptionsJSON, &pokes); err != nil {
		panic(err)
	}
	return pokes
}

func (m model) flashField(fieldKey string) tea.Cmd {
	m.flashingFields[string(fieldKey)]++
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
		return UnFlashFieldMsg(string(fieldKey))
	})
}

func (m model) skipPenalty() (model, tea.Cmd) {
	m.time = max(m.time-5, 0)
	m.penalties++
	m.currentPenalty = 5
	return m, tea.Tick(800*time.Millisecond, func(t time.Time) tea.Msg {
		return PenaltyExpiredMsg{}
	})
}

func (m model) scoreStr() string {
	return strconv.Itoa(151 - len(m.pokes))
}

func (m model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Loading...")
	}

	switch m.screen {
	case SCREEN_MENU:
		return m.menuView()
	case SCREEN_ENDLESS:
		return m.endlessModeView()
	case SCREEN_GAME_OVER:
		return m.gameOverView()
	case SCREEN_TIMED:
		return m.timedModeView()
	}

	return tea.NewView("Hmmm... are you lost? Try pressing escape")
}

func (m model) gameOverView() tea.View {
	all := lipgloss.JoinVertical(
		lipgloss.Center,
		lipgloss.NewStyle().Foreground(lipgloss.Blue).Render(splash),
		m.gameOverDescription(),
		"",
		"",
		m.footerView(false),
	)

	v := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, all))
	v.AltScreen = true

	return v
}

func (m model) endlessModeView() tea.View {
	all := lipgloss.JoinVertical(
		lipgloss.Center,
		lipgloss.NewStyle().Foreground(lipgloss.Blue).Render(splash),
		m.endlessHeaderView(),
		m.descriptionView(),
		"",
		m.inputView(),
		"",
		"",
		"",
		m.footerView(true),
	)

	v := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, all))
	v.AltScreen = true

	return v
}

func (m model) timedModeView() tea.View {
	all := lipgloss.JoinVertical(
		lipgloss.Center,
		lipgloss.NewStyle().Foreground(lipgloss.Blue).Render(splash),
		m.timedHeaderView(),
		m.descriptionView(),
		"",
		m.inputView(),
		"",
		"",
		"",
		m.footerView(true),
	)

	v := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, all))
	v.AltScreen = true

	return v
}

func (m model) menuView() tea.View {
	splashView := lipgloss.JoinVertical(
		lipgloss.Center,
		lipgloss.NewStyle().Foreground(lipgloss.Blue).Render(splash),
		m.gametypeSelectorView(),
		"",
		lipgloss.NewStyle().Faint(true).Underline(true).Render("enter")+
			lipgloss.NewStyle().Faint(true).Render(" to begin, ")+
			lipgloss.NewStyle().Faint(true).Underline(true).Render("tab")+
			lipgloss.NewStyle().Faint(true).Render(" to switch mode"),
	)
	v := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, splashView))

	v.AltScreen = true
	return v
}

func (m model) gametypeSelectorView() string {
	if m.gameTypeSelector == GAMETYPE_ENDLESS {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#008800")).Render("[endless]") +
			" | " +
			lipgloss.NewStyle().Faint(true).Render(" time attack ")
	} else {
		return lipgloss.NewStyle().Faint(true).Render(" endless ") +
			" | " +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#008800")).Render("[time attack]")
	}
}

func (m model) endlessHeaderView() string {
	score := "score: " + m.scoreStr()
	skips := "skips: " + strconv.Itoa(m.skips)
	misses := "misses: " + strconv.Itoa(m.misses)
	flashSkips := m.flashingFields["skips"] > 0
	flashMisses := m.flashingFields["misses"] > 0
	flashScore := m.flashingFields["score"] > 0

	var skipStyle lipgloss.Style
	if flashSkips {
		skipStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e09500"))
	} else {
		skipStyle = lipgloss.NewStyle().Faint(true)
	}

	var scoreStyle lipgloss.Style
	if flashScore {
		scoreStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#33cc66"))
	} else {
		scoreStyle = lipgloss.NewStyle().Faint(true)
	}

	var missesStyle lipgloss.Style
	if flashMisses {
		missesStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	} else {
		missesStyle = lipgloss.NewStyle().Faint(true)
	}

	return scoreStyle.Render(score) + " | " +
		skipStyle.Render(skips) + " | " +
		missesStyle.Render(misses)
}

func (m model) timedHeaderView() string {
	score := "score: " + m.scoreStr()
	var scoreStyle lipgloss.Style
	flashScore := m.flashingFields["score"] > 0
	flashMisses := m.flashingFields["misses"] > 0

	if flashScore {
		scoreStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#33cc66"))
	} else if flashMisses {
		scoreStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	} else {
		scoreStyle = lipgloss.NewStyle().Faint(true)
	}

	timer := lipgloss.NewStyle().Faint(true).Render("time: " + strconv.Itoa(m.time))

	if m.currentPenalty > 0 {
		timer += " -" + lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render(strconv.Itoa(m.currentPenalty))
	}

	return scoreStyle.Render(score) + " | " + timer
}

func (m model) gameOverDescription() string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		PaddingLeft(4).
		PaddingRight(4).
		PaddingTop(1).
		PaddingBottom(1).
		Width(min(m.width, 80)).
		Align(lipgloss.Center).
		Render("game over!\n\nfinal score: " + m.scoreStr() + "\nmisses: " + strconv.Itoa(m.misses) + "\nskips: " + strconv.Itoa(m.skips))
}

func (m model) descriptionView() string {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).PaddingLeft(4).PaddingRight(4).Width(min(m.width, 80)).Align(lipgloss.Center).Render(m.pokes[m.currentPoke].Description[m.currentDesc])
}

func (m model) inputView() string {
	return lipgloss.NewStyle().Width(len(m.textInput.Value()) + 2).Render(m.textInput.View())
}

func (m model) footerView(showTabSkip bool) string {
	style := lipgloss.NewStyle().Faint(true)
	keyStyle := lipgloss.NewStyle().Bold(true).Underline(true).Faint(true)
	footer := keyStyle.Render("esc") + style.Render(" to quit")

	if showTabSkip {
		footer = keyStyle.Render("tab") + style.Render(" to skip | ") + footer
	}

	return footer
}
