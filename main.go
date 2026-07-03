package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"math/rand"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

//go:embed all-descriptions.json
var descriptionsJSON []byte

const splash = " _       ____                    __  __          __     \n| |     / / /_  ____ ( )_____   / /_/ /_  ____ _/ /_    \n| | /| / / __ \\/ __ \\|// ___/  / __/ __ \\/ __ `/ __/    \n| |/ |/ / / / / /_/ / (__  )  / /_/ / / / /_/ / /_      \n|__/|__/_/ /_/\\____/ /____/   \\__/_/ /_/\\__,_/\\__/      \n    ____        __    __                      ___  __   \n   / __ \\____  / /___/_/ ____ ___  ____  ____/__ \\/ /   \n  / /_/ / __ \\/ //_/ _ \\/ __ `__ \\/ __ \\/ __ \\/ _/ /    \n / ____/ /_/ / ,< /  __/ / / / / / /_/ / / / /_//_/     \n/_/    \\____/_/|_|\\___/_/ /_/ /_/\\____/_/ /_(_)(_)      \n                                                        "

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
	p := tea.NewProgram(model{pokes: pokes, textInput: input})
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type model struct {
	isStarted   bool
	currentPoke int
	currentDesc int
	pokes       []Pokemon
	textInput   textinput.Model
	width       int
	height      int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if !m.isStarted {
				m.isStarted = true
			} else if strings.ToLower(m.pokes[m.currentPoke].Name) == strings.ToLower(m.textInput.Value()) {
				m.pokes = append(m.pokes[:m.currentPoke], m.pokes[m.currentPoke+1:]...)
				m.currentPoke = rand.Intn(len(m.pokes))
				m.currentDesc = rand.Intn(len(m.pokes[m.currentPoke].Description))
				m.textInput.Reset()
			}
		case "tab":
			m.currentPoke = rand.Intn(len(m.pokes))
			m.currentDesc = rand.Intn(len(m.pokes[m.currentPoke].Description))
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
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
	return strconv.Itoa(len(m.pokes))
}

func (m model) descriptionView() string {
	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(m.pokes[m.currentPoke].Description[m.currentDesc])
}

func (m model) inputView() string {
	return m.textInput.View()
}

func (m model) footerView() string {
	style := lipgloss.NewStyle().Faint(true)
	keyStyle := lipgloss.NewStyle().Bold(true).Underline(true).Faint(true)
	return keyStyle.Render("tab") + style.Render(" to skip | ") + keyStyle.Render("esc") + style.Render(" to quit")
}
