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

var BasePanelStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder())

var TableHeaderStyle = lipgloss.NewStyle().Bold(true)

func Title(width int, active bool) lipgloss.Style {
	title := lipgloss.NewStyle().
		Background(colour.PanelTitleBG).
		Width(width).
		Height(1).
		// MarginBottom(1).
		MarginLeft(1).
		PaddingLeft(1).
		Bold(true)
	if active {
		title = title.Background(colour.PanelTitleActiveBG).Foreground(colour.PanelTitleActiveFG)
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
