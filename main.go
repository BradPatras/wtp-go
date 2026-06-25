package main

import (
	_ "embed"
	"encoding/json"
	"log"
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
	isStarted bool
	pokes     []Pokemon
	textInput textinput.Model
	width     int
	height    int
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
		case "ctrl+z":
			return m, tea.Suspend
		case "enter":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) processGuess(guess string) {

}

func (m model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Loading...")
	}

	// style := lipgloss.NewStyle().Foreground(lipgloss.Color("#6cacd4"))
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
	return m.pokes[150].Name
}

func (m model) inputView() string {
	return m.textInput.View()
}

func (m model) footerView() string {
	return "esc to quit"
}
