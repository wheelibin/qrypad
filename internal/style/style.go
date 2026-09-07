package style

import (
	"image/color"
	"math"

	"charm.land/lipgloss/v2"

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

func GetTableHighlightStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(theme.GetTheme().CurrentStatement.BG).
		Foreground(theme.GetTheme().CurrentStatement.FG)
}

func GetTableBorderForeground() color.Color {
	return theme.GetTheme().TableBorder.FG
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

// ResultCellStyle returns a lipgloss.Style with the foreground color for the
// given abstract type category. Used by the results panel and row detail popup
// to color-code cell values by their database column type.
func ResultCellStyle(category string) lipgloss.Style {
	t := theme.GetTheme()
	var fg color.Color
	switch category {
	case "number":
		fg = t.SyntaxNumber.FG
	case "string":
		fg = t.SyntaxString.FG
	case "boolean":
		fg = t.SyntaxKeyword.FG
	case "datetime":
		fg = t.SyntaxLiteral.FG
	case "binary":
		fg = t.SyntaxComment.FG
	default:
		fg = t.Text.FG
	}
	if fg == nil {
		fg = t.Text.FG
	}
	return lipgloss.NewStyle().Foreground(fg)
}

// NullStyle returns the style used for SQL NULL values in the results grid.
func NullStyle() lipgloss.Style {
	t := theme.GetTheme()
	fg := t.SyntaxComment.FG
	if fg == nil {
		fg = t.Text.FG
	}
	return lipgloss.NewStyle().Foreground(fg)
}
