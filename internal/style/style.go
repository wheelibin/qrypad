package style

import (
	"math"

	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/colour"
)

const (
	Margin                 = 1
	CurrentStatementHeight = 1
	TitleHeight            = 1
)

func GetBasePanelStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colour.GetTheme().Border.FG)
}

func GetTableHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Background(colour.GetTheme().TableHeader.BG).
		Foreground(colour.GetTheme().TableHeader.FG)
}

func GetSpinnerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(colour.GetTheme().Spinner.FG)
}

func ShortHelp(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center)
}

func Title(width int, active bool) lipgloss.Style {
	theme := colour.GetTheme()
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
		BorderForeground(colour.GetTheme().TableBorder.FG).
		Foreground(colour.GetTheme().Text.FG).
		Align(lipgloss.Left)
}
