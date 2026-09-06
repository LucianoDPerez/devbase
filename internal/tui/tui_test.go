package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestChecklistToggleAndConfirm(t *testing.T) {
	m := model{title: "t", entries: []Entry{
		{Group: "G", ID: "a", Name: "A", Use: "u", On: true},
		{Group: "G", ID: "b", Name: "B", Use: "u", On: true},
	}}
	// Move down, toggle off, confirm.
	var mod tea.Model = m
	var cmd tea.Cmd
	mod, _ = mod.Update(tea.KeyMsg{Type: tea.KeyDown})
	mod, _ = mod.Update(key(" "))
	mod, _ = mod.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = cmd
	got := mod.(model)
	if !got.done {
		t.Fatal("enter must confirm")
	}
	if got.entries[1].On {
		t.Error("second entry should be toggled off")
	}
	if !got.entries[0].On {
		t.Error("first entry must stay on")
	}
	if sel := SelectedIDs(got.entries); !sel["a"] || sel["b"] {
		t.Errorf("SelectedIDs = %v", sel)
	}
	view := got.View()
	if !strings.Contains(view, "[✓]") || !strings.Contains(view, "[ ]") {
		t.Errorf("view must show both states:\n%s", view)
	}
}

func TestAbort(t *testing.T) {
	var mod tea.Model = model{entries: []Entry{{ID: "a", On: true}}}
	mod, _ = mod.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !mod.(model).aborted {
		t.Error("esc must abort")
	}
}
