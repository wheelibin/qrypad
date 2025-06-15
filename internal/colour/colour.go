package colour

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

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
	Text             TC `json:"text"`
	Border           TC `json:"border"`
	BorderActive     TC `json:"border.active"`
	PanelTitle       TC `json:"panelTitle"`
	PanelTitleActive TC `json:"panelTitle.active"`
	CurrentStatement TC `json:"currentStatement"`
	Spinner          TC `json:"spinner"`
	StatusBar        TC `json:"statusBar"`
	TitleBar         TC `json:"titleBar"`
	Error            TC `json:"error"`
	PopupTable       TC `json:"popupTable"`
	DatabaseSwitcher TC `json:"databaseSwitcher"`
	Help             TC `json:"help"`
	TableHeader      TC `json:"tableHeader"`
}

func GetTheme() Theme {
	if err := LoadTheme(); err != nil {
		log.Fatal(err)
	}
	return theme
}

func LoadTheme() error {
	themeOnce.Do(func() {
		file, err := os.Open("internal/colour/themes/catppucin-mocha.json")
		if err != nil {
			themeErr = err
			return
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		themeErr = decoder.Decode(&theme)
	})

	if themeErr != nil {
		return themeErr
	}
	return nil
}
