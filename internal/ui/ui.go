// Package ui centralizes terminal presentation: one palette, one set of
// symbols, graceful plain-text fallback when stdout is not a TTY or
// NO_COLOR is set (handled by lipgloss).
package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	dimStyle   = lipgloss.NewStyle().Faint(true)
	headStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	verdictOK  = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("2"))
	verdictBad = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true).Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("1"))
)

// Ok renders a success row: "  ✓ name  detail".
func Ok(name, detail string) string {
	return fmt.Sprintf("  %s %-14s %s", okStyle.Render("✓"), name, dimStyle.Render(detail))
}

// Warn renders a skipped/pending row.
func Warn(name, detail string) string {
	return fmt.Sprintf("  %s %-14s %s", warnStyle.Render("○"), name, dimStyle.Render(detail))
}

// Fail renders an error row.
func Fail(name, detail string) string {
	return fmt.Sprintf("  %s %-14s %s", errStyle.Render("✗"), name, dimStyle.Render(detail))
}

// Section renders a cyan section header.
func Section(name string) string {
	return headStyle.Render("── " + name + " ──")
}

// Verdict renders the boxed VERIFIED / BLOCKED stamp.
func Verdict(pass bool, sha string) string {
	if pass {
		return verdictOK.Render("✓ VERIFIED @ " + sha)
	}
	return verdictBad.Render("✗ BLOCKED @ " + sha)
}

// Dim renders faint secondary text.
func Dim(s string) string {
	return dimStyle.Render(s)
}
