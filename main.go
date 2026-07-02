package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"math/rand"
	"strconv"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

//go:embed all-descriptions.json
var descriptionsJSON []byte

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
			if m.pokes[m.currentPoke].Name == m.textInput.Value() {
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

	all := lipgloss.JoinVertical(
		lipgloss.Left,
		m.headerView(),
		m.descriptionView(),
		m.inputView(),
		m.footerView(),
	)
	v := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Bottom, all))

	v.AltScreen = true

	return v
}

func (m model) headerView() string {
	return strconv.Itoa(len(m.pokes))
}

func (m model) descriptionView() string {
	return lipgloss.NewStyle().Width(m.width).Render(m.pokes[m.currentPoke].Description[m.currentDesc])
}

func (m model) inputView() string {
	return m.textInput.View()
}

func (m model) footerView() string {
	style := lipgloss.NewStyle().Faint(true)
	keyStyle := lipgloss.NewStyle().Bold(true).Underline(true).Faint(true)
	return keyStyle.Render("tab") + style.Render(" to skip | ") + keyStyle.Render("esc") + style.Render(" to quit")
}
