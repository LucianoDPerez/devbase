// Package tui hosts the interactive setup checklist: a curated catalog
// grouped by tool kind, everything selected by default, space toggles.
package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Entry is one catalog row.
type Entry struct {
	Group string // MCPs, Reglas, Skills, Herramientas
	ID    string // stable selector id, e.g. "tool:semgrep", "stack:js", "mcp:context7"
	Name  string
	Use   string // one-line use case
	Note  string // e.g. "already installed" (informational only)
	On    bool
}

var (
	groupStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	onStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	offStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	nameStyle  = lipgloss.NewStyle().Bold(true)
	useStyle   = lipgloss.NewStyle().Faint(true)
	curStyle   = lipgloss.NewStyle().Background(lipgloss.Color("8")).Bold(true)
	titleStyle = lipgloss.NewStyle().Bold(true).Underline(true)
)

type model struct {
	title   string
	entries []Entry
	cursor  int
	done    bool
	aborted bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.aborted = true
			return m, tea.Quit
		case "enter":
			m.done = true
			return m, tea.Quit
		case " ", "x":
			m.entries[m.cursor].On = !m.entries[m.cursor].On
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.title) + "\n\n")
	lastGroup := ""
	for i, e := range m.entries {
		if e.Group != lastGroup {
			b.WriteString(groupStyle.Render("◆ "+e.Group) + "\n")
			lastGroup = e.Group
		}
		box := offStyle.Render("[ ]")
		if e.On {
			box = onStyle.Render("[✓]")
		}
		line := fmt.Sprintf("  %s %s  %s", box, nameStyle.Render(e.Name), useStyle.Render(e.Use))
		if e.Note != "" {
			line += "  " + useStyle.Render("("+e.Note+")")
		}
		if i == m.cursor && !m.done {
			line = curStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + useStyle.Render("↑↓ mover · espacio/X alternar · enter confirmar · q salir") + "\n")
	return b.String()
}

// Select runs the interactive checklist. Without a TTY it returns entries
// untouched (everything default-on) so pipes and CI keep working.
func Select(title string, entries []Entry) ([]Entry, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return entries, nil
	}
	p := tea.NewProgram(model{title: title, entries: entries})
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	m := final.(model)
	if m.aborted {
		return nil, fmt.Errorf("cancelled by user")
	}
	return m.entries, nil
}

// SelectedIDs returns the ids left on.
func SelectedIDs(entries []Entry) map[string]bool {
	out := map[string]bool{}
	for _, e := range entries {
		if e.On {
			out[e.ID] = true
		}
	}
	return out
}
