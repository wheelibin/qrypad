package colour

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
)

//go:embed themes/*
var themes embed.FS

const defaultThemeName = "catppuccin-mocha"

var (
	theme     Theme
	themeOnce sync.Once
	themeErr  error
)

type TC struct {
	FG lipgloss.Color
	BG lipgloss.Color
}

func (tc *TC) UnmarshalJSON(data []byte) error {
	// Expecting: {"fg": "#fff", "bg": "#000"}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if fg, ok := raw["fg"]; ok {
		tc.FG = lipgloss.Color(fg)
	}
	if bg, ok := raw["bg"]; ok {
		tc.BG = lipgloss.Color(bg)
	}
	return nil
}

type Theme struct {
	ThemeName string

	Border                TC `json:"border"`
	BorderActive          TC `json:"border:active"`
	CurrentStatement      TC `json:"currentStatement"`
	DatabaseSwitcherPopup TC `json:"databaseSwitcherPopup"`
	Error                 TC `json:"error"`
	HelpPopup             TC `json:"helpPopup"`
	HelpDesc              TC `json:"helpDesc"`
	HelpKey               TC `json:"helpKey"`
	PanelTitle            TC `json:"panelTitle"`
	PanelTitleActive      TC `json:"panelTitle:active"`
	RowDetailsPopup       TC `json:"rowDetailsPopup"`
	Spinner               TC `json:"spinner"`
	StatusBar             TC `json:"statusBar"`
	TableBorder           TC `json:"tableBorder"`
	TableHeader           TC `json:"tableHeader"`
	Text                  TC `json:"text"`
	TitleBar              TC `json:"titleBar"`
}

func GetTheme() Theme {
	if err := LoadTheme(); err != nil {
		log.Fatal(err)
	}

	return theme
}

func LoadTheme() error {
	viper.SetDefault("theme", defaultThemeName)
	themeName := viper.GetString("theme")

	themeOnce.Do(func() {
		data, err := themes.ReadFile(fmt.Sprintf("themes/%s.json", themeName))
		if err != nil {
			fallback, _ := themes.ReadFile(fmt.Sprintf("themes/%s.json", defaultThemeName))
			themeErr = json.Unmarshal(fallback, &theme)
			theme.ThemeName = defaultThemeName
			registerChromaStyle()
			return
		}
		themeErr = json.Unmarshal(data, &theme)
		theme.ThemeName = themeName
		registerChromaStyle()
	})

	if themeErr != nil {
		return themeErr
	}
	return nil
}

func registerChromaStyle() {
	if _, found := styles.Registry[theme.ThemeName]; !found {
		styles.Register(chroma.MustNewStyle(theme.ThemeName, chroma.StyleEntries{
			chroma.Literal:     string(theme.Text.FG),
			chroma.Name:        string(theme.Text.FG),
			chroma.Comment:     fmt.Sprintf("italic %s bg:%s", theme.PanelTitle.FG, theme.PanelTitle.BG),
			chroma.Keyword:     fmt.Sprintf("bold %s", theme.BorderActive.FG),
			chroma.Operator:    string(theme.Text.FG),
			chroma.String:      string(theme.DatabaseSwitcherPopup.BG),
			chroma.Number:      string(theme.HelpPopup.BG),
			chroma.Punctuation: string(theme.Text.FG),
		}))
	}
}
