package component

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/stopwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

type loadingPopupKeymap struct {
	cancel key.Binding
}

type LoadingPopupModel struct {
	width     int
	height    int
	spinner   spinner.Model
	stopwatch stopwatch.Model
	keymap    loadingPopupKeymap
	help      help.Model
}

func NewLoadingPopupModel() LoadingPopupModel {
	s := spinner.New()
	s.Spinner = spinner.Meter
	s.Style = style.GetSpinnerStyle()
	return LoadingPopupModel{
		spinner:   s,
		stopwatch: stopwatch.New(),
		help:      makeHelp(),
		keymap: loadingPopupKeymap{
			cancel: keys.DefaultKeyMap.CancelQuery,
		},
	}
}

func (m LoadingPopupModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.stopwatch.Init(),
	)
}

func (m LoadingPopupModel) Update(msg tea.Msg) (LoadingPopupModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case commands.LoadingMsg:
		cmds = append(cmds, m.spinner.Tick)
		cmds = append(cmds, tea.Sequence(m.stopwatch.Reset(), m.stopwatch.Start()))

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.cancel):
			cmds = append(cmds, commands.CancelQuery())
		}
	}

	m.stopwatch, cmd = m.stopwatch.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m LoadingPopupModel) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.cancel,
	})
}

func (m *LoadingPopupModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m LoadingPopupModel) View() string {
	loadingPopupWidth := 20
	spinner := lipgloss.NewStyle().
		MarginRight(2).
		MarginTop(1).
		Render(m.spinner.View())
	stopwatch := style.GetSpinnerStyle().
		Render(m.stopwatch.Elapsed().String())
	spinnerDisplay := lipgloss.JoinHorizontal(lipgloss.Center, spinner, stopwatch)
	helpView := style.ShortHelp(loadingPopupWidth).
		Render(m.helpView())
	loadingPopupContent := lipgloss.JoinVertical(lipgloss.Center, spinnerDisplay, helpView)
	loadingPopup := style.GetBasePanelStyle().
		BorderForeground(theme.GetTheme().Spinner.FG).
		Width(loadingPopupWidth).
		Height(5).Render(loadingPopupContent)

	return loadingPopup
}
