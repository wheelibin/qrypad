package style

import (
	"math"

	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/theme"
)

const (
	Margin                 = 1
	CurrentStatementHeight = 1
	TitleHeight            = 1
)

func GetBasePanelStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.GetTheme().Border.FG)
}

func GetTableHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Background(theme.GetTheme().TableHeader.BG).
		Foreground(theme.GetTheme().TableHeader.FG)
}

func GetSpinnerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(theme.GetTheme().Spinner.FG)
}

func ShortHelp(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center)
}

func Title(width int, active bool) lipgloss.Style {
	theme := theme.GetTheme()
	title := lipgloss.NewStyle().
		Background(theme.PanelTitle.BG).
		Foreground(theme.PanelTitle.FG).
		Width(width).
		Height(1).
		MarginLeft(1).
		PaddingLeft(1).
		Bold(true)
	if active {
		title = title.Background(theme.PanelTitleActive.BG).Foreground(theme.PanelTitleActive.FG)
	}
	return title
}

func GetSpan(span int, total int) int {
	if span == 12 {
		return total
	}
	oneCell := float64(total) / float64(12)
	result := math.Ceil(oneCell * float64(span))
	return int(result)
}

func TableColumn() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderForeground(theme.GetTheme().TableBorder.FG).
		Foreground(theme.GetTheme().Text.FG).
		Align(lipgloss.Left)
}

func WindowTooSmall(w, h int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(w).
		Height(h).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(theme.GetTheme().Error.FG)
}
