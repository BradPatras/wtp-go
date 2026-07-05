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

const splash = "  _      __ __                ______ __         __  \n | | /| / // /  ___ ( )___   /_  __// /  ___ _ / /_ \n | |/ |/ // _ \\/ _ \\|/(_-<    / /  / _ \\/ _ `// __/ \n |__/|__//_//_/\\___/ /___/   /_/  /_//_/\\_,_/ \\__/  \n     ___         __     __                  ___   __\n    / _ \\ ___   / /__ _/_/ __ _  ___   ___ /__ \\ / /\n   / ___// _ \\ /  '_// -_)/  ' \\/ _ \\ / _ \\ /__//_/ \n  /_/    \\___//_/\\_\\ \\__//_/_/_/\\___//_//_/(_) (_)  \n                                                    \n"

// Message used to make a field "flash"
type FlashFieldMsg string
type UnFlashFieldMsg string

type Pokemon struct {
	Id          int      `json:"id"`
	Name        string   `json:"name"`
	Description []string `json:"description"`
}

func main() {
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

	p := tea.NewProgram(model{pokes: pokes, textInput: input, currentPoke: currentPoke, currentDesc: currentDesc, flashingFields: make(map[string]int)})
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type model struct {
	isStarted      bool
	currentPoke    int
	currentDesc    int
	pokes          []Pokemon
	textInput      textinput.Model
	width          int
	height         int
	skips          int
	misses         int
	flashingFields map[string]int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if !m.isStarted {
				m.isStarted = true
			} else if strings.EqualFold(m.pokes[m.currentPoke].Name, m.textInput.Value()) {
				cmds = append(cmds, m.flashField("score"))
				m.pokes = append(m.pokes[:m.currentPoke], m.pokes[m.currentPoke+1:]...)
				m.currentPoke = rand.Intn(len(m.pokes))
				m.currentDesc = rand.Intn(len(m.pokes[m.currentPoke].Description))
				m.textInput.Reset()
			} else {
				cmds = append(cmds, m.flashField("misses"))
				m.misses++
			}
		case "tab":
			m.skips++
			cmds = append(cmds, m.flashField("skips"))
			m.currentPoke = rand.Intn(len(m.pokes))
			m.currentDesc = rand.Intn(len(m.pokes[m.currentPoke].Description))
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case UnFlashFieldMsg:
		current := m.flashingFields[string(msg)]
		m.flashingFields[string(msg)] = max(0, current-1)
	}

	var tcmd tea.Cmd
	m.textInput, tcmd = m.textInput.Update(msg)
	cmds = append(cmds, tcmd)

	return m, tea.Batch(cmds...)
}

func (m model) flashField(fieldKey string) tea.Cmd {
	m.flashingFields[string(fieldKey)]++
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
		return UnFlashFieldMsg(string(fieldKey))
	})
}

func (m model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Loading...")
	}

	if !m.isStarted {
		splashView := lipgloss.JoinVertical(
			lipgloss.Center,
			lipgloss.NewStyle().Foreground(lipgloss.Blue).Render(splash),
			lipgloss.NewStyle().Faint(true).Underline(true).Render("enter")+lipgloss.NewStyle().Faint(true).Render(" to begin"),
		)
		v := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, splashView))

		v.AltScreen = true
		return v
	} else {
		all := lipgloss.JoinVertical(
			lipgloss.Center,
			lipgloss.NewStyle().Foreground(lipgloss.Blue).Render(splash),
			m.headerView(),
			m.descriptionView(),
			"",
			m.inputView(),
			"",
			"",
			"",
			m.footerView(),
		)

		v := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, all))
		v.AltScreen = true

		return v
	}
}

func (m model) headerView() string {
	score := "score: " + strconv.Itoa(151-len(m.pokes))
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

func (m model) descriptionView() string {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).PaddingLeft(4).PaddingRight(4).Width(min(m.width, 80)).Align(lipgloss.Center).Render(m.pokes[m.currentPoke].Description[m.currentDesc])
}

func (m model) inputView() string {
	return lipgloss.NewStyle().Width(len(m.textInput.Value()) + 2).Render(m.textInput.View())
}

func (m model) footerView() string {
	style := lipgloss.NewStyle().Faint(true)
	keyStyle := lipgloss.NewStyle().Bold(true).Underline(true).Faint(true)
	return keyStyle.Render("tab") + style.Render(" to skip | ") + keyStyle.Render("esc") + style.Render(" to quit")
}
