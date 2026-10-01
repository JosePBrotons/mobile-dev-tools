package tui

import (
	"strconv"

	"github.com/charmbracelet/lipgloss"
)

func itoa(n int) string { return strconv.Itoa(n) }

func lipglossWidth(w int) lipgloss.Style { return lipgloss.NewStyle().Width(w).MarginRight(2) }

func joinColumns(left, right string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}
