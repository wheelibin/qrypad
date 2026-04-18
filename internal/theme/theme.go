package theme

import (
	"embed"
	"encoding/json"
	"fmt"
	"image/color"
	"log/slog"
	"maps"
	"sync"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/spf13/viper"
)

//go:embed themes/*
var themes embed.FS

const defaultThemeName = "catppuccin-mocha"

var (
	theme     Theme     //nolint:gochecknoglobals // singleton theme loaded once via sync.Once
	themeOnce sync.Once //nolint:gochecknoglobals // sync.Once for thread-safe single init
	errTheme  error     //nolint:gochecknoglobals // error from singleton theme init
)

type TC struct {
	FG color.Color `json:"fg,omitempty" mapstructure:"fg,omitempty"`
	BG color.Color `json:"bg,omitempty" mapstructure:"bg,omitempty"`
}

func (tc *TC) UnmarshalJSON(data []byte) error {
	// Expecting: {"fg": "#fff", "bg": "#000"}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("error unmarshalling theme color: %w", err)
	}
	if fg, ok := raw["fg"]; ok {
		tc.FG = lipgloss.Color(fg)
	}
	if bg, ok := raw["bg"]; ok {
		tc.BG = lipgloss.Color(bg)
	}
	return nil
}

func (tc TC) MarshalJSON() ([]byte, error) {
	raw := map[string]string{}
	if tc.FG != nil {
		raw["fg"] = colorStr(tc.FG)
	}
	if tc.BG != nil {
		raw["bg"] = colorStr(tc.BG)
	}
	return json.Marshal(raw)
}

type Theme struct {
	ThemeName               string `json:"themeName,omitempty"               mapstructure:"themeName,omitempty"`
	Border                  *TC    `json:"border,omitempty"                  mapstructure:"border,omitempty"`
	BorderActive            *TC    `json:"border:active,omitempty"           mapstructure:"borderActive,omitempty"`
	CurrentStatement        *TC    `json:"currentStatement,omitempty"        mapstructure:"currentStatement,omitempty"`
	DatabaseSwitcherPopup   *TC    `json:"databaseSwitcherPopup,omitempty"   mapstructure:"databaseSwitcherPopup,omitempty"`
	ConnectionSwitcherPopup *TC    `json:"connectionSwitcherPopup,omitempty" mapstructure:"connectionSwitcherPopup,omitempty"`
	Error                   *TC    `json:"error,omitempty"                   mapstructure:"error,omitempty"`
	HelpPopup               *TC    `json:"helpPopup,omitempty"               mapstructure:"helpPopup,omitempty"`
	HelpDesc                *TC    `json:"helpDesc,omitempty"                mapstructure:"helpDesc,omitempty"`
	HelpKey                 *TC    `json:"helpKey,omitempty"                 mapstructure:"helpKey,omitempty"`
	PanelTitle              *TC    `json:"panelTitle,omitempty"              mapstructure:"panelTitle,omitempty"`
	PanelTitleActive        *TC    `json:"panelTitle:active,omitempty"       mapstructure:"panelTitleActive,omitempty"`
	RowDetailsPopup         *TC    `json:"rowDetailsPopup,omitempty"         mapstructure:"rowDetailsPopup,omitempty"`
	Spinner                 *TC    `json:"spinner,omitempty"                 mapstructure:"spinner,omitempty"`
	StatusBar               *TC    `json:"statusBar,omitempty"               mapstructure:"statusBar,omitempty"`
	TableBorder             *TC    `json:"tableBorder,omitempty"             mapstructure:"tableBorder,omitempty"`
	TableHeader             *TC    `json:"tableHeader,omitempty"             mapstructure:"tableHeader,omitempty"`
	Text                    *TC    `json:"text,omitempty"                    mapstructure:"text,omitempty"`
	TitleBar                *TC    `json:"titleBar,omitempty"                mapstructure:"titleBar,omitempty"`
	TitleBarAlt             *TC    `json:"titleBarAlt,omitempty"             mapstructure:"titleBarAlt,omitempty"`
}

func BlankTheme() Theme {
	return Theme{
		ThemeName:             "",
		Border:                &TC{},
		BorderActive:          &TC{},
		CurrentStatement:      &TC{},
		DatabaseSwitcherPopup: &TC{},
		ConnectionSwitcherPopup: &TC{},
		Error:                 &TC{},
		HelpPopup:             &TC{},
		HelpDesc:              &TC{},
		HelpKey:               &TC{},
		PanelTitle:            &TC{},
		PanelTitleActive:      &TC{},
		RowDetailsPopup:       &TC{},
		Spinner:               &TC{},
		StatusBar:             &TC{},
		TableBorder:           &TC{},
		TableHeader:           &TC{},
		Text:                  &TC{},
		TitleBar:              &TC{},
		TitleBarAlt:           &TC{},
	}
}

func GetTheme() Theme {
	if err := LoadTheme(); err != nil {
		slog.Error("fatal error loading theme", "error", err)
		panic(err)
	}
	return theme
}

func LoadTheme() error {
	viper.SetDefault("theme", defaultThemeName)
	themeName := viper.GetString("theme.name")

	themeOnce.Do(func() {
		data, err := themes.ReadFile(fmt.Sprintf("themes/%s.json", themeName))
		if err != nil {
			fallback, _ := themes.ReadFile(fmt.Sprintf("themes/%s.json", defaultThemeName))
			errTheme = json.Unmarshal(fallback, &theme)
			theme.ThemeName = defaultThemeName
			applyConfigOverrides()
			registerChromaStyle()
			return
		}
		errTheme = json.Unmarshal(data, &theme)
		theme.ThemeName = themeName
		applyConfigOverrides()
		registerChromaStyle()
	})

	if errTheme != nil {
		return fmt.Errorf("error loading theme: %w", errTheme)
	}
	return nil
}

func applyConfigOverrides() {
	var overrides Theme
	if sub := viper.Sub("theme"); sub != nil {
		if err := sub.Unmarshal(&overrides); err != nil {
			slog.Error("fatal error applying theme overrides", "error", err)
			panic(err)
		}
		if overrides.ThemeName != "" {
			theme = BlankTheme()
		} else {
			overrides.ThemeName = theme.ThemeName
		}
		ot, err := mergeStructs(theme, overrides)
		if err != nil {
			slog.Error("fatal error merging theme structs", "error", err)
			panic(err)
		}
		theme = ot
	}
}

// colorStr converts a color.Color to a hex string suitable for Chroma style entries.
// In lipgloss v2, Color() returns color.Color (interface) instead of a string type.
func colorStr(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}

// registerChromaStyle builds and registers a chroma style using the
// theme colors, if the theme doesn't already exist
func registerChromaStyle() {
	if _, found := styles.Registry[theme.ThemeName]; !found {
		styles.Register(chroma.MustNewStyle(theme.ThemeName, chroma.StyleEntries{
			chroma.Literal:     colorStr(theme.Text.FG),
			chroma.Name:        colorStr(theme.Text.FG),
			chroma.Comment:     fmt.Sprintf("italic %s bg:%s", colorStr(theme.PanelTitle.FG), colorStr(theme.PanelTitle.BG)),
			chroma.Keyword:     fmt.Sprintf("bold %s", colorStr(theme.BorderActive.FG)),
			chroma.Operator:    colorStr(theme.Text.FG),
			chroma.String:      colorStr(theme.DatabaseSwitcherPopup.BG),
			chroma.Number:      colorStr(theme.HelpPopup.BG),
			chroma.Punctuation: colorStr(theme.Text.FG),
		}))
	}
}

func mergeStructs[T any](base, override T) (T, error) {
	var result T
	baseJSON, err := json.Marshal(base)
	if err != nil {
		return result, fmt.Errorf("error marshalling base struct: %w", err)
	}
	overrideJSON, err := json.Marshal(override)
	if err != nil {
		return result, fmt.Errorf("error marshalling override struct: %w", err)
	}

	var merged map[string]any
	if err := json.Unmarshal(baseJSON, &merged); err != nil {
		return result, fmt.Errorf("error unmarshalling base JSON: %w", err)
	}

	var overrideMap map[string]any
	if err := json.Unmarshal(overrideJSON, &overrideMap); err != nil {
		return result, fmt.Errorf("error unmarshalling override JSON: %w", err)
	}

	maps.Copy(merged, overrideMap)

	finalJSON, err := json.Marshal(merged)
	if err != nil {
		return result, fmt.Errorf("error marshalling merged struct: %w", err)
	}

	err = json.Unmarshal(finalJSON, &result)
	if err != nil {
		return result, fmt.Errorf("error unmarshalling final struct: %w", err)
	}
	return result, nil
}
