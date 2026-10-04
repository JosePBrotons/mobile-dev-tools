package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#4F46E5", Dark: "#8B87FF"})
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#8A8A8A", Dark: "#6C6C6C"})
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#0E9F6E", Dark: "#04B575"})
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#F5B93A"})
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#C62828", Dark: "#FF6B6B"})
	boldStyle  = lipgloss.NewStyle().Bold(true)
)

// clip cuts a line to width cells so the layout never wraps by accident.
func clip(s string, width int) string {
	if width <= 0 {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}

// window returns the slice of lines that fits in height and keeps focus
// roughly centered.
func window(lines []string, focus, height int) []string {
	if height <= 0 || len(lines) <= height {
		return lines
	}
	start := max(0, min(focus-height/2, len(lines)-height))
	return lines[start : start+height]
}
