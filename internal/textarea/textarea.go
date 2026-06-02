package textarea

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
	qpStyle "github.com/wheelibin/qrypad/internal/style"
)

const (
	minHeight        = 1
	minWidth         = 2
	defaultHeight    = 6
	defaultWidth     = 40
	defaultCharLimit = 400
	defaultMaxHeight = 99
	defaultMaxWidth  = 500
)

// Internal messages for clipboard operations.
type (
	pasteMsg    string
	pasteErrMsg struct{ error } //nolint:errname // Bubble Tea message type, not a standard Go error
)

// KeyMap is the key bindings for different actions within the textarea.
type KeyMap struct {
	CharacterBackward       key.Binding
	CharacterForward        key.Binding
	DeleteAfterCursor       key.Binding
	DeleteBeforeCursor      key.Binding
	DeleteCharacterBackward key.Binding
	DeleteCharacterForward  key.Binding
	DeleteWordBackward      key.Binding
	DeleteWordForward       key.Binding
	InsertNewline           key.Binding
	LineEnd                 key.Binding
	LineNext                key.Binding
	LinePrevious            key.Binding
	LineStart               key.Binding
	Paste                   key.Binding
	WordBackward            key.Binding
	WordForward             key.Binding
	InputBegin              key.Binding
	InputEnd                key.Binding

	UppercaseWordForward  key.Binding
	LowercaseWordForward  key.Binding
	CapitalizeWordForward key.Binding

	TransposeCharacterBackward key.Binding

	PageUp   key.Binding
	PageDown key.Binding

	// Undo and Redo are left unbound by default so the textarea package
	// has no hard dependency on the application's keys package. Callers
	// (e.g. QueryPanelModel) wire them explicitly after textarea.New().
	Undo key.Binding
	Redo key.Binding
}

// DefaultKeyMap is the default set of key bindings for navigating and acting
// upon the textarea.
//
//nolint:gochecknoglobals // default keymap is required for Bubble Tea component
var DefaultKeyMap = KeyMap{
	CharacterForward:        key.NewBinding(key.WithKeys("right", "ctrl+f")),
	CharacterBackward:       key.NewBinding(key.WithKeys("left", "ctrl+b")),
	WordForward:             key.NewBinding(key.WithKeys("alt+right", "alt+f")),
	WordBackward:            key.NewBinding(key.WithKeys("alt+left", "alt+b")),
	LineNext:                key.NewBinding(key.WithKeys("down", "ctrl+n")),
	LinePrevious:            key.NewBinding(key.WithKeys("up", "ctrl+p")),
	DeleteWordBackward:      key.NewBinding(key.WithKeys("alt+backspace", "ctrl+w")),
	DeleteWordForward:       key.NewBinding(key.WithKeys("alt+delete", "alt+d")),
	DeleteAfterCursor:       key.NewBinding(key.WithKeys("ctrl+k")),
	DeleteBeforeCursor:      key.NewBinding(key.WithKeys("ctrl+u")),
	InsertNewline:           key.NewBinding(key.WithKeys("enter", "ctrl+m")),
	DeleteCharacterBackward: key.NewBinding(key.WithKeys("backspace", "ctrl+h")),
	DeleteCharacterForward:  key.NewBinding(key.WithKeys("delete", "ctrl+d")),
	LineStart:               key.NewBinding(key.WithKeys("home", "ctrl+a")),
	LineEnd:                 key.NewBinding(key.WithKeys("end", "ctrl+e")),
	Paste:                   key.NewBinding(key.WithKeys("ctrl+v")),
	InputBegin:              key.NewBinding(key.WithKeys("alt+<", "ctrl+home")),
	InputEnd:                key.NewBinding(key.WithKeys("alt+>", "ctrl+end")),

	CapitalizeWordForward: key.NewBinding(key.WithKeys("alt+c")),
	LowercaseWordForward:  key.NewBinding(key.WithKeys("alt+l")),
	UppercaseWordForward:  key.NewBinding(key.WithKeys("alt+u")),

	TransposeCharacterBackward: key.NewBinding(key.WithKeys("ctrl+t")),

	PageUp:   key.NewBinding(key.WithKeys("pgup")),
	PageDown: key.NewBinding(key.WithKeys("pgdown")),
}

// LineInfo is a helper for keeping track of line information regarding
// soft-wrapped lines.
type LineInfo struct {
	// Width is the number of columns in the line.
	Width int
	// CharWidth is the number of characters in the line to account for
	// double-width runes.
	CharWidth int
	// Height is the number of rows in the line.
	Height int
	// StartColumn is the index of the first column of the line.
	StartColumn int
	// ColumnOffset is the number of columns that the cursor is offset from the
	// start of the line.
	ColumnOffset int
	// RowOffset is the number of rows that the cursor is offset from the start
	// of the line.
	RowOffset int
	// CharOffset is the number of characters that the cursor is offset
	// from the start of the line. This will generally be equivalent to
	// ColumnOffset, but will be different there are double-width runes before
	// the cursor.
	CharOffset int
}

// Style that will be applied to the text area.
//
// Style can be applied to focused and unfocused states to change the styles
// depending on the focus state.
//
// For an introduction to styling with Lip Gloss see:
// https://github.com/charmbracelet/lipgloss
type Style struct {
	Base             lipgloss.Style
	CursorLine       lipgloss.Style
	CursorLineNumber lipgloss.Style
	EndOfBuffer      lipgloss.Style
	LineNumber       lipgloss.Style
	Placeholder      lipgloss.Style
	Prompt           lipgloss.Style
	Text             lipgloss.Style
}

// line is the input to the text wrapping function. This is stored in a struct
// so that it can be hashed and memoized.
type line struct {
	runes []rune
	width int
}

// Hash returns a hash of the line.
func (w line) Hash() string {
	v := fmt.Sprintf("%s:%d", string(w.runes), w.width)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(v)))
}

// Model is the Bubble Tea model for this text area element.
//
//nolint:recvcheck // Bubble Tea model: Init/View use value receiver, mutating methods use pointer receiver
type Model struct {
	Err error

	// General settings.
	cache *memoCache[line, [][]rune]

	// Prompt is printed at the beginning of each line.
	//
	// When changing the value of Prompt after the model has been
	// initialized, ensure that SetWidth() gets called afterwards.
	//
	// See also SetPromptFunc().
	Prompt string

	// Placeholder is the text displayed when the user
	// hasn't entered anything yet.
	Placeholder string

	// ShowLineNumbers, if enabled, causes line numbers to be printed
	// after the prompt.
	ShowLineNumbers bool

	// EndOfBufferCharacter is displayed at the end of the input.
	EndOfBufferCharacter rune

	// KeyMap encodes the keybindings recognized by the widget.
	KeyMap KeyMap

	// Styling. FocusedStyle and BlurredStyle are used to style the textarea in
	// focused and blurred states.
	FocusedStyle Style
	BlurredStyle Style
	// style is the current styling to use.
	// It is used to abstract the differences in focus state when styling the
	// model, since we can simply assign the set of styles to this variable
	// when switching focus states.
	style *Style

	// Cursor is the text area cursor.
	Cursor cursor.Model

	// CharLimit is the maximum number of characters this input element will
	// accept. If 0 or less, there's no limit.
	CharLimit int

	// MaxHeight is the maximum height of the text area in rows. If 0 or less,
	// there's no limit.
	MaxHeight int

	// MaxWidth is the maximum width of the text area in columns. If 0 or less,
	// there's no limit.
	MaxWidth int

	// If promptFunc is set, it replaces Prompt as a generator for
	// prompt strings at the beginning of each line.
	promptFunc func(line int) string

	// promptWidth is the width of the prompt.
	promptWidth int

	// width is the maximum number of characters that can be displayed at once.
	// If 0 or less this setting is ignored.
	width int

	// height is the maximum number of lines that can be displayed at once. It
	// essentially treats the text field like a vertically scrolling viewport
	// if there are more lines than the permitted height.
	height int

	// Underlying text value.
	value [][]rune

	// focus indicates whether user input focus should be on this input
	// component. When false, ignore keyboard input and hide the cursor.
	focus bool

	// Cursor column.
	Col int

	// Cursor Row.
	Row int

	// Last character offset, used to maintain state when the cursor is moved
	// vertically such that we can maintain the same navigating position.
	lastCharOffset int

	// lineNumberFormat is the format string used to display line numbers.
	lineNumberFormat string

	// viewport is the vertically-scrollable viewport of the multi-line text
	// input.
	viewport *viewport.Model

	// rune sanitizer for input.
	rsan Sanitizer

	// history tracks undo/redo snapshots. Initialised lazily by
	// ensureHistory() on first mutation. Lazy because New() returns
	// Model by value; binding a history to &m inside New would point
	// at a soon-to-be-copied local.
	history *history
}

// New creates a new model with default settings.
func New() Model {
	vp := viewport.New()
	vp.KeyMap = viewport.KeyMap{}
	cur := cursor.New()

	focusedStyle, blurredStyle := DefaultStyles(true)

	m := Model{
		CharLimit:            defaultCharLimit,
		MaxHeight:            defaultMaxHeight,
		MaxWidth:             defaultMaxWidth,
		Prompt:               lipgloss.ThickBorder().Left + " ",
		style:                &blurredStyle,
		FocusedStyle:         focusedStyle,
		BlurredStyle:         blurredStyle,
		cache:                newMemoCache[line, [][]rune](defaultMaxHeight),
		EndOfBufferCharacter: '~',
		ShowLineNumbers:      true,
		Cursor:               cur,
		KeyMap:               DefaultKeyMap,

		value:            make([][]rune, minHeight, defaultMaxHeight),
		focus:            false,
		Col:              0,
		Row:              0,
		lineNumberFormat: "%3v ",

		viewport: &vp,
	}

	m.SetHeight(defaultHeight)
	m.SetWidth(defaultWidth)

	return m
}

// DefaultStyles returns the default styles for focused and blurred states for
// the textarea. Pass isDark=true for dark backgrounds, false for light.
func DefaultStyles(isDark bool) (Style, Style) {
	lightDark := lipgloss.LightDark(isDark)
	focused := Style{
		Base:             lipgloss.NewStyle(),
		CursorLine:       lipgloss.NewStyle().Background(lightDark(lipgloss.Color("255"), lipgloss.Color("0"))),
		CursorLineNumber: lipgloss.NewStyle().Foreground(lightDark(lipgloss.Color("240"), lipgloss.Color("240"))),
		EndOfBuffer:      lipgloss.NewStyle().Foreground(lightDark(lipgloss.Color("254"), lipgloss.Color("0"))),
		LineNumber:       lipgloss.NewStyle().Foreground(lightDark(lipgloss.Color("249"), lipgloss.Color("7"))),
		Placeholder:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Prompt:           lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
		Text:             lipgloss.NewStyle(),
	}
	blurred := Style{
		Base:             lipgloss.NewStyle(),
		CursorLine:       lipgloss.NewStyle().Foreground(lightDark(lipgloss.Color("245"), lipgloss.Color("7"))),
		CursorLineNumber: lipgloss.NewStyle().Foreground(lightDark(lipgloss.Color("249"), lipgloss.Color("7"))),
		EndOfBuffer:      lipgloss.NewStyle().Foreground(lightDark(lipgloss.Color("254"), lipgloss.Color("0"))),
		LineNumber:       lipgloss.NewStyle().Foreground(lightDark(lipgloss.Color("249"), lipgloss.Color("7"))),
		Placeholder:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Prompt:           lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
		Text:             lipgloss.NewStyle().Foreground(lightDark(lipgloss.Color("245"), lipgloss.Color("7"))),
	}

	return focused, blurred
}

// SetValue sets the value of the text input and resets undo/redo history
// so the newly-loaded buffer is the new baseline.
func (m *Model) SetValue(s string) {
	m.setValueInternal(s)
	// Reset history to a clean baseline around the new value.
	m.history = newHistory(m)
}

// ReplaceValue replaces the entire buffer contents as a single undoable
// edit. Use this for programmatic replacements that the user should be
// able to revert with Ctrl+Z (e.g. accepting an autocomplete suggestion),
// in contrast to SetValue which resets history (used when the new value
// is a fresh baseline from disk, $EDITOR, or a connection switch).
func (m *Model) ReplaceValue(s string) {
	m.ensureHistory()
	m.history.recordPre(m, opOther)
	m.setValueInternal(s)
}

// InsertString inserts a string at the cursor position.
func (m *Model) InsertString(s string) {
	m.insertRunesFromUserInput([]rune(s))
}

// InsertRune inserts a rune at the cursor position.
func (m *Model) InsertRune(r rune) {
	m.insertRunesFromUserInput([]rune{r})
}

// Value returns the value of the text input.
func (m Model) Value() string {
	if m.value == nil {
		return ""
	}

	var v strings.Builder
	for _, l := range m.value {
		v.WriteString(string(l))
		v.WriteByte('\n')
	}

	return strings.TrimSuffix(v.String(), "\n")
}

// Length returns the number of characters currently in the text input.
func (m *Model) Length() int {
	var l int
	for _, row := range m.value {
		l += uniseg.StringWidth(string(row))
	}
	// We add len(m.value) to include the newline characters.
	return l + len(m.value) - 1
}

// LineCount returns the number of lines that are currently in the text input.
func (m *Model) LineCount() int {
	return len(m.value)
}

// Line returns the line position.
func (m Model) Line() int {
	return m.Row
}

// CursorDown moves the cursor down by one line.
// Returns whether or not the cursor blink should be reset.
func (m *Model) CursorDown() {
	li := m.LineInfo()
	charOffset := max(m.lastCharOffset, li.CharOffset)
	m.lastCharOffset = charOffset

	if li.RowOffset+1 >= li.Height && m.Row < len(m.value)-1 {
		m.Row++
		m.Col = 0
	} else {
		// Move the cursor to the start of the next line so that we can get
		// the line information. We need to add 2 columns to account for the
		// trailing space wrapping.
		m.Col = min(li.StartColumn+li.Width+2, len(m.value[m.Row])-1)
	}

	nli := m.LineInfo()
	m.Col = nli.StartColumn

	if nli.Width <= 0 {
		return
	}

	offset := 0
	for offset < charOffset {
		if m.Row >= len(m.value) || m.Col >= len(m.value[m.Row]) || offset >= nli.CharWidth-1 {
			break
		}
		offset += xansi.StringWidth(string(m.value[m.Row][m.Col]))
		m.Col++
	}
}

// CursorUp moves the cursor up by one line.
func (m *Model) CursorUp() {
	li := m.LineInfo()
	charOffset := max(m.lastCharOffset, li.CharOffset)
	m.lastCharOffset = charOffset

	if li.RowOffset <= 0 && m.Row > 0 {
		m.Row--
		m.Col = len(m.value[m.Row])
	} else {
		// Move the cursor to the end of the previous line.
		// This can be done by moving the cursor to the start of the line and
		// then subtracting 2 to account for the trailing space we keep on
		// soft-wrapped lines.
		m.Col = li.StartColumn - 2
	}

	nli := m.LineInfo()
	m.Col = nli.StartColumn

	if nli.Width <= 0 {
		return
	}

	offset := 0
	for offset < charOffset {
		if m.Col >= len(m.value[m.Row]) || offset >= nli.CharWidth-1 {
			break
		}
		offset += xansi.StringWidth(string(m.value[m.Row][m.Col]))
		m.Col++
	}
}

// PageUp moves the cursor up by the height of the viewport.
func (m *Model) PageUp() {
	for range m.height {
		m.CursorUp()
	}
}

// PageDown moves the cursor down by the height of the viewport.
func (m *Model) PageDown() {
	for range m.height {
		m.CursorDown()
	}
}

// SetCursor moves the cursor to the given position. If the position is
// out of bounds the cursor will be moved to the start or end accordingly.
func (m *Model) SetCursor(col int) {
	m.Col = clamp(col, 0, len(m.value[m.Row]))
	// Any time that we move the cursor horizontally we need to reset the last
	// offset so that the horizontal position when navigating is adjusted.
	m.lastCharOffset = 0
}

// CursorStart moves the cursor to the start of the input field.
func (m *Model) CursorStart() {
	m.SetCursor(0)
}

// CursorEnd moves the cursor to the end of the input field.
func (m *Model) CursorEnd() {
	m.SetCursor(len(m.value[m.Row]))
}

// Focused returns the focus state on the model.
func (m Model) Focused() bool {
	return m.focus
}

// Focus sets the focus state on the model. When the model is in focus it can
// receive keyboard input and the cursor will be hidden.
func (m *Model) Focus() tea.Cmd {
	m.focus = true
	m.style = &m.FocusedStyle
	return m.Cursor.Focus()
}

// Blur removes the focus state on the model. When the model is blurred it can
// not receive keyboard input and the cursor will be hidden.
func (m *Model) Blur() {
	m.focus = false
	m.style = &m.BlurredStyle
	m.Cursor.Blur()
}

// Reset sets the input to its default state with no input.
func (m *Model) Reset() {
	startCap := m.MaxHeight
	if startCap <= 0 {
		startCap = defaultMaxHeight
	}
	m.value = make([][]rune, minHeight, startCap)
	m.Col = 0
	m.Row = 0
	m.viewport.GotoTop()
	m.SetCursor(0)
}

// LineInfo returns the number of characters from the start of the
// (soft-wrapped) line and the (soft-wrapped) line width.
func (m Model) LineInfo() LineInfo {
	grid := m.memoizedWrap(m.value[m.Row], m.width)

	// Find out which line we are currently on. This can be determined by the
	// m.col and counting the number of runes that we need to skip.
	var counter int
	for i, line := range grid {
		// We've found the line that we are on
		if counter+len(line) == m.Col && i+1 < len(grid) {
			// We wrap around to the next line if we are at the end of the
			// previous line so that we can be at the very beginning of the row
			return LineInfo{
				CharOffset:   0,
				ColumnOffset: 0,
				Height:       len(grid),
				RowOffset:    i + 1,
				StartColumn:  m.Col,
				Width:        len(grid[i+1]),
				CharWidth:    uniseg.StringWidth(string(line)),
			}
		}

		if counter+len(line) >= m.Col {
			return LineInfo{
				CharOffset:   uniseg.StringWidth(string(line[:max(0, m.Col-counter)])),
				ColumnOffset: m.Col - counter,
				Height:       len(grid),
				RowOffset:    i,
				StartColumn:  counter,
				Width:        len(line),
				CharWidth:    uniseg.StringWidth(string(line)),
			}
		}

		counter += len(line)
	}
	return LineInfo{}
}

// Width returns the width of the textarea.
func (m Model) Width() int {
	return m.width
}

// SetWidth sets the width of the textarea to fit exactly within the given width.
// This means that the textarea will account for the width of the prompt and
// whether or not line numbers are being shown.
//
// Ensure that SetWidth is called after setting the Prompt and ShowLineNumbers,
// It is important that the width of the textarea be exactly the given width
// and no more.
func (m *Model) SetWidth(w int) {
	if m.MaxWidth > 0 {
		m.viewport.SetWidth(clamp(w, minWidth, m.MaxWidth))
	} else {
		m.viewport.SetWidth(max(w, minWidth))
	}

	// Since the width of the textarea input is dependent on the width of the
	// prompt and line numbers, we need to calculate it by subtracting.
	inputWidth := w
	if m.ShowLineNumbers {
		inputWidth -= uniseg.StringWidth(fmt.Sprintf(m.lineNumberFormat, 0))
	}

	// Account for base style borders and padding.
	inputWidth -= m.style.Base.GetHorizontalFrameSize()

	if m.promptFunc == nil {
		m.promptWidth = uniseg.StringWidth(m.Prompt)
	}

	inputWidth -= m.promptWidth
	if m.MaxWidth > 0 {
		m.width = clamp(inputWidth, minWidth, m.MaxWidth)
	} else {
		m.width = max(inputWidth, minWidth)
	}
}

// SetPromptFunc supersedes the Prompt field and sets a dynamic prompt
// instead.
// If the function returns a prompt that is shorter than the
// specified promptWidth, it will be padded to the left.
// If it returns a prompt that is longer, display artifacts
// may occur; the caller is responsible for computing an adequate
// promptWidth.
func (m *Model) SetPromptFunc(promptWidth int, fn func(lineIdx int) string) {
	m.promptFunc = fn
	m.promptWidth = promptWidth
}

// Height returns the current height of the textarea.
func (m Model) Height() int {
	return m.height
}

// CursorViewRow returns the cursor's row position relative to the visible
// viewport (0-indexed from the top of the visible area).
func (m Model) CursorViewRow() int {
	return m.cursorLineNumber() - m.viewport.YOffset()
}

// SetHeight sets the height of the textarea.
func (m *Model) SetHeight(h int) {
	if m.MaxHeight > 0 {
		m.height = clamp(h, minHeight, m.MaxHeight)
		m.viewport.SetHeight(clamp(h, minHeight, m.MaxHeight))
	} else {
		m.height = max(h, minHeight)
		m.viewport.SetHeight(max(h, minHeight))
	}
}

// Update is the Bubble Tea update loop.
//
//nolint:gocyclo,cyclop // Bubble Tea message handling inherently requires complex switch statements
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focus {
		m.Cursor.Blur()
		return m, nil
	}

	// Used to determine if the cursor should blink.
	oldRow, oldCol := m.cursorLineNumber(), m.Col

	cmds := make([]tea.Cmd, 0, 2)

	if m.value[m.Row] == nil {
		m.value[m.Row] = make([]rune, 0)
	}

	if m.MaxHeight > 0 && m.MaxHeight != m.cache.Capacity() {
		m.cache = newMemoCache[line, [][]rune](m.MaxHeight)
	}

	switch msg := msg.(type) {
	case tea.PasteMsg:
		m.ensureHistory()
		m.history.recordPre(&m, opOther)
		m.insertRunesFromUserInput([]rune(msg.Content))
	case tea.KeyPressMsg:
		// Hoisted: every branch below needs history initialised (either to
		// call recordPre or to touch lastOp). Initialise once.
		m.ensureHistory()
		switch {
		case key.Matches(msg, m.KeyMap.Undo):
			m.history.undo(&m)
		case key.Matches(msg, m.KeyMap.Redo):
			m.history.redo(&m)
		case key.Matches(msg, m.KeyMap.DeleteAfterCursor):
			m.history.recordPre(&m, opOther)
			m.Col = clamp(m.Col, 0, len(m.value[m.Row]))
			if m.Col >= len(m.value[m.Row]) {
				m.mergeLineBelow(m.Row)
				break
			}
			m.deleteAfterCursor()
		case key.Matches(msg, m.KeyMap.DeleteBeforeCursor):
			m.history.recordPre(&m, opOther)
			m.Col = clamp(m.Col, 0, len(m.value[m.Row]))
			if m.Col <= 0 {
				m.mergeLineAbove(m.Row)
				break
			}
			m.deleteBeforeCursor()
		case key.Matches(msg, m.KeyMap.DeleteCharacterBackward):
			m.history.recordPre(&m, opDeleteChar)
			m.Col = clamp(m.Col, 0, len(m.value[m.Row]))
			if m.Col <= 0 {
				m.mergeLineAbove(m.Row)
				break
			}
			if len(m.value[m.Row]) > 0 {
				m.value[m.Row] = append(m.value[m.Row][:max(0, m.Col-1)], m.value[m.Row][m.Col:]...)
				if m.Col > 0 {
					m.SetCursor(m.Col - 1)
				}
			}
		case key.Matches(msg, m.KeyMap.DeleteCharacterForward):
			m.history.recordPre(&m, opDeleteChar)
			if len(m.value[m.Row]) > 0 && m.Col < len(m.value[m.Row]) {
				m.value[m.Row] = append(m.value[m.Row][:m.Col], m.value[m.Row][m.Col+1:]...)
			}
			if m.Col >= len(m.value[m.Row]) {
				m.mergeLineBelow(m.Row)
				break
			}
		case key.Matches(msg, m.KeyMap.DeleteWordBackward):
			m.history.recordPre(&m, opOther)
			if m.Col <= 0 {
				m.mergeLineAbove(m.Row)
				break
			}
			m.deleteWordLeft()
		case key.Matches(msg, m.KeyMap.DeleteWordForward):
			m.history.recordPre(&m, opOther)
			m.Col = clamp(m.Col, 0, len(m.value[m.Row]))
			if m.Col >= len(m.value[m.Row]) {
				m.mergeLineBelow(m.Row)
				break
			}
			m.deleteWordRight()
		case key.Matches(msg, m.KeyMap.InsertNewline):
			m.history.recordPre(&m, opOther)
			if m.MaxHeight > 0 && len(m.value) >= m.MaxHeight {
				return m, nil
			}
			m.Col = clamp(m.Col, 0, len(m.value[m.Row]))
			m.splitLine(m.Row, m.Col)
		case key.Matches(msg, m.KeyMap.LineEnd):
			m.history.lastOp = opNone
			m.CursorEnd()
		case key.Matches(msg, m.KeyMap.LineStart):
			m.history.lastOp = opNone
			m.CursorStart()
		case key.Matches(msg, m.KeyMap.CharacterForward):
			m.history.lastOp = opNone
			m.characterRight()
		case key.Matches(msg, m.KeyMap.LineNext):
			m.history.lastOp = opNone
			m.CursorDown()
		case key.Matches(msg, m.KeyMap.WordForward):
			m.history.lastOp = opNone
			m.wordRight()
		case key.Matches(msg, m.KeyMap.Paste):
			m.history.lastOp = opNone
			return m, Paste
		case key.Matches(msg, m.KeyMap.CharacterBackward):
			m.history.lastOp = opNone
			m.characterLeft(false /* insideLine */)
		case key.Matches(msg, m.KeyMap.LinePrevious):
			m.history.lastOp = opNone
			m.CursorUp()
		case key.Matches(msg, m.KeyMap.WordBackward):
			m.history.lastOp = opNone
			m.wordLeft()
		case key.Matches(msg, m.KeyMap.InputBegin):
			m.history.lastOp = opNone
			m.moveToBegin()
		case key.Matches(msg, m.KeyMap.InputEnd):
			m.history.lastOp = opNone
			m.moveToEnd()
		case key.Matches(msg, m.KeyMap.LowercaseWordForward):
			m.history.recordPre(&m, opOther)
			m.lowercaseRight()
		case key.Matches(msg, m.KeyMap.UppercaseWordForward):
			m.history.recordPre(&m, opOther)
			m.uppercaseRight()
		case key.Matches(msg, m.KeyMap.CapitalizeWordForward):
			m.history.recordPre(&m, opOther)
			m.capitalizeRight()
		case key.Matches(msg, m.KeyMap.TransposeCharacterBackward):
			m.history.recordPre(&m, opOther)
			m.transposeLeft()
		case key.Matches(msg, m.KeyMap.PageUp):
			m.history.lastOp = opNone
			m.PageUp()
		case key.Matches(msg, m.KeyMap.PageDown):
			m.history.lastOp = opNone
			m.PageDown()

		default:
			runes := []rune(msg.Text)
			op := opOther
			if len(runes) == 1 && !isWhitespaceRune(runes[0]) {
				op = opInsertChar
			}
			m.history.recordPre(&m, op)
			m.insertRunesFromUserInput(runes)
		}

	case pasteMsg:
		m.ensureHistory()
		m.history.recordPre(&m, opOther)
		m.insertRunesFromUserInput([]rune(msg))

	case pasteErrMsg:
		m.Err = msg
	}

	vp, cmd := m.viewport.Update(msg)
	m.viewport = &vp
	cmds = append(cmds, cmd)

	newRow, newCol := m.cursorLineNumber(), m.Col
	m.Cursor, cmd = m.Cursor.Update(msg)
	if (newRow != oldRow || newCol != oldCol) && m.Cursor.Mode() == cursor.CursorBlink {
		m.Cursor.IsBlinked = false
		cmd = m.Cursor.Blink()
	}
	cmds = append(cmds, cmd)

	m.repositionView()

	return m, tea.Batch(cmds...)
}

// View renders the text area in its current state.
func (m Model) View() string {
	if m.Value() == "" && m.Row == 0 && m.Col == 0 && m.Placeholder != "" {
		return m.placeholderView()
	}
	m.Cursor.TextStyle = m.style.CursorLine

	var s strings.Builder
	var style lipgloss.Style
	lineInfo := m.LineInfo()

	var newLines int

	displayLine := 0
	for l, line := range m.value {
		wrappedLines := m.memoizedWrap(line, m.width)

		if m.Row == l {
			style = m.style.CursorLine
		} else {
			style = m.style.Text
		}

		for wl, wrappedLine := range wrappedLines {
			prompt := m.getPromptString(displayLine)
			prompt = m.style.Prompt.Render(prompt)
			s.WriteString(style.Render(prompt))
			displayLine++

			if m.ShowLineNumbers {
				if wl == 0 {
					if m.Row == l {
						s.WriteString(style.Render(m.style.CursorLineNumber.Render(fmt.Sprintf(m.lineNumberFormat, l+1))))
					} else {
						s.WriteString(style.Render(m.style.LineNumber.Render(fmt.Sprintf(m.lineNumberFormat, l+1))))
					}
				} else {
					if m.Row == l {
						s.WriteString(style.Render(m.style.CursorLineNumber.Render(fmt.Sprintf(m.lineNumberFormat, " "))))
					} else {
						s.WriteString(style.Render(m.style.LineNumber.Render(fmt.Sprintf(m.lineNumberFormat, " "))))
					}
				}
			}

			strwidth := uniseg.StringWidth(string(wrappedLine))
			padding := m.width - strwidth
			// If the trailing space causes the line to be wider than the
			// width, we should not draw it to the screen since it will result
			// in an extra space at the end of the line which can look off when
			// the cursor line is showing.
			if strwidth > m.width {
				// The character causing the line to be wider than the width is
				// guaranteed to be a space since any other character would
				// have been wrapped.
				wrappedLine = []rune(strings.TrimSuffix(string(wrappedLine), " "))
				padding -= m.width - strwidth
			}

			if m.Row == l && lineInfo.RowOffset == wl {
				// Highlight the full wrapped line first, then split the styled
				// output around the cursor position. This ensures tokens that
				// span the cursor (e.g. SELECT with cursor in the middle) are
				// highlighted correctly.
				fullStyled := qpStyle.HighlightText(string(wrappedLine))
				before, after := qpStyle.SplitStyledLine(fullStyled, lineInfo.ColumnOffset)

				s.WriteString(before)
				if m.Col >= len(line) && lineInfo.CharOffset >= m.width {
					m.Cursor.SetChar(" ")
					s.WriteString(m.Cursor.View())
				} else {
					cursorChar := string(wrappedLine[lineInfo.ColumnOffset])
					m.Cursor.SetChar(cursorChar)

					// Extract the full Chroma styling (bold, colour, etc.) for
					// the character under the cursor.
					styledChar := qpStyle.ExtractStyledChar(fullStyled, lineInfo.ColumnOffset)

					if m.Cursor.IsBlinked {
						// Blink-out (no block): write the Chroma-styled character
						// directly so that all attributes (bold, colour) are
						// preserved. Falling through to Cursor.View() would only
						// apply CursorLine's style, dropping Chroma attributes.
						if styledChar != "" {
							s.WriteString(styledChar)
						} else {
							s.WriteString(m.Cursor.View())
						}
					} else {
						// Blink-in (block visible): cursor.View() renders
						// Style.Reverse(true) which was always correct.
						s.WriteString(m.Cursor.View())
					}

					s.WriteString(after)
				}
			} else {
				s.WriteString(qpStyle.HighlightText(string(wrappedLine)))
			}
			// s.WriteString(style.Render(strings.Repeat(" ", max(0, padding))))
			s.WriteString(strings.Repeat(" ", max(0, padding)))
			s.WriteRune('\n')
			newLines++
		}
	}

	// Always show at least `m.Height` lines at all times.
	// To do this we can simply pad out a few extra new lines in the view.
	for range m.height {
		prompt := m.getPromptString(displayLine)
		prompt = m.style.Prompt.Render(prompt)
		s.WriteString(prompt)
		displayLine++

		if m.ShowLineNumbers {
			lineNumber := m.style.EndOfBuffer.Render((fmt.Sprintf(m.lineNumberFormat, string(m.EndOfBufferCharacter))))
			s.WriteString(lineNumber)
		}
		s.WriteRune('\n')
	}

	m.viewport.SetContent(s.String())
	return m.style.Base.Render(m.viewport.View())
}

// Blink returns the blink command for the cursor.
func Blink() tea.Msg {
	return cursor.Blink()
}

// Paste is a command for pasting from the clipboard into the text input.
func Paste() tea.Msg {
	str, err := clipboard.ReadAll()
	if err != nil {
		return pasteErrMsg{err}
	}
	return pasteMsg(str)
}

func wrap(runes []rune, width int) [][]rune {
	var (
		lines  = [][]rune{{}}
		word   = []rune{}
		row    int
		spaces int
	)

	// Word wrap the runes
	for _, r := range runes {
		if unicode.IsSpace(r) {
			spaces++
		} else {
			word = append(word, r)
		}

		if spaces > 0 {
			if uniseg.StringWidth(string(lines[row]))+uniseg.StringWidth(string(word))+spaces > width {
				row++
				lines = append(lines, []rune{})
				lines[row] = append(lines[row], word...)
				lines[row] = append(lines[row], repeatSpaces(spaces)...)
				spaces = 0
				word = nil
			} else {
				lines[row] = append(lines[row], word...)
				lines[row] = append(lines[row], repeatSpaces(spaces)...)
				spaces = 0
				word = nil
			}
		} else {
			// If the last character is a double-width rune, then we may not be able to add it to this line
			// as it might cause us to go past the width.
			lastCharLen := xansi.StringWidth(string(word[len(word)-1]))
			if uniseg.StringWidth(string(word))+lastCharLen > width {
				// If the current line has any content, let's move to the next
				// line because the current word fills up the entire line.
				if len(lines[row]) > 0 {
					row++
					lines = append(lines, []rune{})
				}
				lines[row] = append(lines[row], word...)
				word = nil
			}
		}
	}

	if uniseg.StringWidth(string(lines[row]))+uniseg.StringWidth(string(word))+spaces >= width {
		lines = append(lines, []rune{})
		lines[row+1] = append(lines[row+1], word...)
		// We add an extra space at the end of the line to account for the
		// trailing space at the end of the previous soft-wrapped lines so that
		// behaviour when navigating is consistent and so that we don't need to
		// continually add edges to handle the last line of the wrapped input.
		spaces++
		lines[row+1] = append(lines[row+1], repeatSpaces(spaces)...)
	} else {
		lines[row] = append(lines[row], word...)
		spaces++
		lines[row] = append(lines[row], repeatSpaces(spaces)...)
	}

	return lines
}

func repeatSpaces(n int) []rune {
	return []rune(strings.Repeat(string(' '), n))
}

func clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}

// --- bufferState implementation for use by *history ---
// Model satisfies bufferState in this package only. These helpers let
// the history package mutate the buffer without going through SetValue
// (which would reset history).

// SetValueRaw is the bufferState counterpart of setValueInternal;
// called only from history.restore.
func (m *Model) SetValueRaw(s string) {
	m.setValueInternal(s)
}

// CursorPos reports the current cursor position.
func (m *Model) CursorPos() (int, int) {
	return m.Row, m.Col
}

// SetCursorPos places the cursor at (row, col), clamping col to the
// length of the target row as a safety net.
func (m *Model) SetCursorPos(row, col int) {
	if row < 0 {
		row = 0
	}
	if row >= len(m.value) {
		row = len(m.value) - 1
	}
	m.Row = row
	rowLen := len(m.value[m.Row])
	if col < 0 {
		col = 0
	}
	if col > rowLen {
		col = rowLen
	}
	m.Col = col
}

// setValueInternal replaces the buffer contents without touching undo
// history. Used both by SetValue (which then resets history) and by the
// history restore path (which must preserve history).
func (m *Model) setValueInternal(s string) {
	m.Reset()
	m.InsertString(s)
}

// ensureHistory initialises the undo/redo history lazily. Lazy because
// New() returns Model by value; binding a history to &m inside New
// would point at a soon-to-be-copied local.
func (m *Model) ensureHistory() {
	if m.history == nil {
		m.history = newHistory(m)
	}
}

// insertRunesFromUserInput inserts runes at the current cursor position.
func (m *Model) insertRunesFromUserInput(runes []rune) {
	// Clean up any special characters in the input provided by the
	// clipboard. This avoids bugs due to e.g. tab characters and
	// whatnot.
	runes = m.san().Sanitize(runes)

	var availSpace int
	if m.CharLimit > 0 {
		availSpace = m.CharLimit - m.Length()
		// If the char limit's been reached, cancel.
		if availSpace <= 0 {
			return
		}
		// If there's not enough space to paste the whole thing cut the pasted
		// runes down so they'll fit.
		if availSpace < len(runes) {
			runes = runes[:availSpace]
		}
	}

	// Split the input into lines.
	var lines [][]rune
	lstart := 0
	for i := range runes {
		if runes[i] == '\n' {
			// Queue a line to become a new row in the text area below.
			// Beware to clamp the max capacity of the slice, to ensure no
			// data from different rows get overwritten when later edits
			// will modify this line.
			lines = append(lines, runes[lstart:i:i])
			lstart = i + 1
		}
	}
	if lstart <= len(runes) {
		// The last line did not end with a newline character.
		// Take it now.
		lines = append(lines, runes[lstart:])
	}

	// Obey the maximum height limit.
	if m.MaxHeight > 0 && len(m.value)+len(lines)-1 > m.MaxHeight {
		allowedHeight := max(0, m.MaxHeight-len(m.value)+1)
		lines = lines[:allowedHeight]
	}

	if len(lines) == 0 {
		// Nothing left to insert.
		return
	}

	// Save the remainder of the original line at the current
	// cursor position.
	tail := make([]rune, len(m.value[m.Row][m.Col:]))
	copy(tail, m.value[m.Row][m.Col:])

	// Paste the first line at the current cursor position.
	m.value[m.Row] = append(m.value[m.Row][:m.Col], lines[0]...)
	m.Col += len(lines[0])

	if numExtraLines := len(lines) - 1; numExtraLines > 0 {
		// Add the new lines.
		// We try to reuse the slice if there's already space.
		var newGrid [][]rune
		if cap(m.value) >= len(m.value)+numExtraLines {
			// Can reuse the extra space.
			newGrid = m.value[:len(m.value)+numExtraLines]
		} else {
			// No space left; need a new slice.
			newGrid = make([][]rune, len(m.value)+numExtraLines)
			copy(newGrid, m.value[:m.Row+1])
		}
		// Add all the rows that were after the cursor in the original
		// grid at the end of the new grid.
		copy(newGrid[m.Row+1+numExtraLines:], m.value[m.Row+1:])
		m.value = newGrid
		// Insert all the new lines in the middle.
		for _, l := range lines[1:] {
			m.Row++
			m.value[m.Row] = l
			m.Col = len(l)
		}
	}

	// Finally add the tail at the end of the last line inserted.
	m.value[m.Row] = append(m.value[m.Row], tail...)

	m.SetCursor(m.Col)
}

// san initializes or retrieves the rune sanitizer.
func (m *Model) san() Sanitizer {
	if m.rsan == nil {
		// Textinput has all its input on a single line so collapse
		// newlines/tabs to single spaces.
		m.rsan = NewSanitizer()
	}
	return m.rsan
}

// deleteBeforeCursor deletes all text before the cursor. Returns whether or
// not the cursor blink should be reset.
func (m *Model) deleteBeforeCursor() {
	m.value[m.Row] = m.value[m.Row][m.Col:]
	m.SetCursor(0)
}

// deleteAfterCursor deletes all text after the cursor. Returns whether or not
// the cursor blink should be reset. If input is masked delete everything after
// the cursor so as not to reveal word breaks in the masked input.
func (m *Model) deleteAfterCursor() {
	m.value[m.Row] = m.value[m.Row][:m.Col]
	m.SetCursor(len(m.value[m.Row]))
}

// transposeLeft exchanges the runes at the cursor and immediately
// before. No-op if the cursor is at the beginning of the line.  If
// the cursor is not at the end of the line yet, moves the cursor to
// the right.
func (m *Model) transposeLeft() {
	if m.Col == 0 || len(m.value[m.Row]) < 2 {
		return
	}
	if m.Col >= len(m.value[m.Row]) {
		m.SetCursor(m.Col - 1)
	}
	m.value[m.Row][m.Col-1], m.value[m.Row][m.Col] = m.value[m.Row][m.Col], m.value[m.Row][m.Col-1]
	if m.Col < len(m.value[m.Row]) {
		m.SetCursor(m.Col + 1)
	}
}

// deleteWordLeft deletes the word left to the cursor. Returns whether or not
// the cursor blink should be reset.
func (m *Model) deleteWordLeft() {
	if m.Col == 0 || len(m.value[m.Row]) == 0 {
		return
	}

	// Linter note: it's critical that we acquire the initial cursor position
	// here prior to altering it via SetCursor() below. As such, moving this
	// call into the corresponding if clause does not apply here.
	oldCol := m.Col //nolint:ifshort

	m.SetCursor(m.Col - 1)
	for unicode.IsSpace(m.value[m.Row][m.Col]) {
		if m.Col <= 0 {
			break
		}
		// ignore series of whitespace before cursor
		m.SetCursor(m.Col - 1)
	}

	for m.Col > 0 {
		if !unicode.IsSpace(m.value[m.Row][m.Col]) {
			m.SetCursor(m.Col - 1)
		} else {
			if m.Col > 0 {
				// keep the previous space
				m.SetCursor(m.Col + 1)
			}
			break
		}
	}

	if oldCol > len(m.value[m.Row]) {
		m.value[m.Row] = m.value[m.Row][:m.Col]
	} else {
		m.value[m.Row] = append(m.value[m.Row][:m.Col], m.value[m.Row][oldCol:]...)
	}
}

// deleteWordRight deletes the word right to the cursor.
func (m *Model) deleteWordRight() {
	if m.Col >= len(m.value[m.Row]) || len(m.value[m.Row]) == 0 {
		return
	}

	oldCol := m.Col

	for m.Col < len(m.value[m.Row]) && unicode.IsSpace(m.value[m.Row][m.Col]) {
		// ignore series of whitespace after cursor
		m.SetCursor(m.Col + 1)
	}

	for m.Col < len(m.value[m.Row]) {
		if !unicode.IsSpace(m.value[m.Row][m.Col]) {
			m.SetCursor(m.Col + 1)
		} else {
			break
		}
	}

	if m.Col > len(m.value[m.Row]) {
		m.value[m.Row] = m.value[m.Row][:oldCol]
	} else {
		m.value[m.Row] = append(m.value[m.Row][:oldCol], m.value[m.Row][m.Col:]...)
	}

	m.SetCursor(oldCol)
}

// characterRight moves the cursor one character to the right.
func (m *Model) characterRight() {
	if m.Col < len(m.value[m.Row]) {
		m.SetCursor(m.Col + 1)
	} else if m.Row < len(m.value)-1 {
		m.Row++
		m.CursorStart()
	}
}

// characterLeft moves the cursor one character to the left.
// If insideLine is set, the cursor is moved to the last
// character in the previous line, instead of one past that.
func (m *Model) characterLeft(insideLine bool) {
	if m.Col == 0 && m.Row != 0 {
		m.Row--
		m.CursorEnd()
		if !insideLine {
			return
		}
	}
	if m.Col > 0 {
		m.SetCursor(m.Col - 1)
	}
}

// wordLeft moves the cursor one word to the left. Returns whether or not the
// cursor blink should be reset. If input is masked, move input to the start
// so as not to reveal word breaks in the masked input.
func (m *Model) wordLeft() {
	for {
		m.characterLeft(true /* insideLine */)
		if m.Col < len(m.value[m.Row]) && !unicode.IsSpace(m.value[m.Row][m.Col]) {
			break
		}
	}

	for m.Col > 0 {
		if unicode.IsSpace(m.value[m.Row][m.Col-1]) {
			break
		}
		m.SetCursor(m.Col - 1)
	}
}

// wordRight moves the cursor one word to the right. Returns whether or not the
// cursor blink should be reset. If the input is masked, move input to the end
// so as not to reveal word breaks in the masked input.
func (m *Model) wordRight() {
	m.doWordRight(func(int, int) { /* nothing */ })
}

func (m *Model) doWordRight(fn func(charIdx int, pos int)) {
	// Skip spaces forward.
	for m.Col >= len(m.value[m.Row]) || unicode.IsSpace(m.value[m.Row][m.Col]) {
		if m.Row == len(m.value)-1 && m.Col == len(m.value[m.Row]) {
			// End of text.
			break
		}
		m.characterRight()
	}

	charIdx := 0
	for m.Col < len(m.value[m.Row]) {
		if unicode.IsSpace(m.value[m.Row][m.Col]) {
			break
		}
		fn(charIdx, m.Col)
		m.SetCursor(m.Col + 1)
		charIdx++
	}
}

// uppercaseRight changes the word to the right to uppercase.
func (m *Model) uppercaseRight() {
	m.doWordRight(func(_ int, i int) {
		m.value[m.Row][i] = unicode.ToUpper(m.value[m.Row][i])
	})
}

// lowercaseRight changes the word to the right to lowercase.
func (m *Model) lowercaseRight() {
	m.doWordRight(func(_ int, i int) {
		m.value[m.Row][i] = unicode.ToLower(m.value[m.Row][i])
	})
}

// capitalizeRight changes the word to the right to title case.
func (m *Model) capitalizeRight() {
	m.doWordRight(func(charIdx int, i int) {
		if charIdx == 0 {
			m.value[m.Row][i] = unicode.ToTitle(m.value[m.Row][i])
		}
	})
}

// repositionView repositions the view of the viewport based on the defined
// scrolling behavior.
func (m *Model) repositionView() {
	minRow := m.viewport.YOffset()
	maxRow := minRow + m.viewport.Height() - 1

	if row := m.cursorLineNumber(); row < minRow {
		m.viewport.ScrollUp(minRow - row)
	} else if row > maxRow {
		m.viewport.ScrollDown(row - maxRow)
	}
}

// moveToBegin moves the cursor to the beginning of the input.
func (m *Model) moveToBegin() {
	m.Row = 0
	m.SetCursor(0)
}

// moveToEnd moves the cursor to the end of the input.
func (m *Model) moveToEnd() {
	m.Row = len(m.value) - 1
	m.SetCursor(len(m.value[m.Row]))
}

func (m Model) getPromptString(displayLine int) string {
	prompt := m.Prompt
	if m.promptFunc == nil {
		return prompt
	}
	prompt = m.promptFunc(displayLine)
	pl := uniseg.StringWidth(prompt)
	if pl < m.promptWidth {
		prompt = fmt.Sprintf("%*s%s", m.promptWidth-pl, "", prompt)
	}
	return prompt
}

// placeholderView returns the prompt and placeholder view, if any.
func (m Model) placeholderView() string {
	var (
		s     strings.Builder
		p     = xansi.Truncate(m.Placeholder, m.width, "...")
		style = m.style.Placeholder.Inline(true)
	)

	prompt := m.getPromptString(0)
	prompt = m.style.Prompt.Render(prompt)
	s.WriteString(m.style.CursorLine.Render(prompt))

	if m.ShowLineNumbers {
		s.WriteString(m.style.CursorLine.Render(m.style.CursorLineNumber.Render((fmt.Sprintf(m.lineNumberFormat, 1)))))
	}

	m.Cursor.TextStyle = m.style.Placeholder
	m.Cursor.SetChar(string(p[0]))
	s.WriteString(m.style.CursorLine.Render(m.Cursor.View()))

	// The rest of the placeholder text
	s.WriteString(m.style.CursorLine.Render(style.Render(p[1:] + strings.Repeat(" ", max(0, m.width-uniseg.StringWidth(p))))))

	// The rest of the new lines
	for i := 1; i < m.height; i++ {
		s.WriteRune('\n')
		prompt := m.getPromptString(i)
		prompt = m.style.Prompt.Render(prompt)
		s.WriteString(prompt)

		if m.ShowLineNumbers {
			eob := m.style.EndOfBuffer.Render((fmt.Sprintf(m.lineNumberFormat, string(m.EndOfBufferCharacter))))
			s.WriteString(eob)
		}
	}

	m.viewport.SetContent(s.String())
	return m.style.Base.Render(m.viewport.View())
}

func (m Model) memoizedWrap(runes []rune, width int) [][]rune {
	input := line{runes: runes, width: width}
	if v, ok := m.cache.Get(input); ok {
		return v
	}
	v := wrap(runes, width)
	m.cache.Set(input, v)
	return v
}

// cursorLineNumber returns the line number that the cursor is on.
// This accounts for soft wrapped lines.
func (m Model) cursorLineNumber() int {
	line := 0
	for i := range m.Row {
		// Calculate the number of lines that the current line will be split
		// into.
		line += len(m.memoizedWrap(m.value[i], m.width))
	}
	line += m.LineInfo().RowOffset
	return line
}

// mergeLineBelow merges the current line the cursor is on with the line below.
func (m *Model) mergeLineBelow(row int) {
	if row >= len(m.value)-1 {
		return
	}

	// To perform a merge, we will need to combine the two lines and then
	m.value[row] = append(m.value[row], m.value[row+1]...)

	// Shift all lines up by one
	for i := row + 1; i < len(m.value)-1; i++ {
		m.value[i] = m.value[i+1]
	}

	// And, remove the last line
	if len(m.value) > 0 {
		m.value = m.value[:len(m.value)-1]
	}
}

// mergeLineAbove merges the current line the cursor is on with the line above.
func (m *Model) mergeLineAbove(row int) {
	if row <= 0 {
		return
	}

	m.Col = len(m.value[row-1])
	m.Row--

	// To perform a merge, we will need to combine the two lines and then
	m.value[row-1] = append(m.value[row-1], m.value[row]...)

	// Shift all lines up by one
	for i := row; i < len(m.value)-1; i++ {
		m.value[i] = m.value[i+1]
	}

	// And, remove the last line
	if len(m.value) > 0 {
		m.value = m.value[:len(m.value)-1]
	}
}

func (m *Model) splitLine(row, col int) {
	// To perform a split, take the current line and keep the content before
	// the cursor, take the content after the cursor and make it the content of
	// the line underneath, and shift the remaining lines down by one
	head, tailSrc := m.value[row][:col], m.value[row][col:]
	tail := make([]rune, len(tailSrc))
	copy(tail, tailSrc)

	m.value = append(m.value[:row+1], m.value[row:]...)

	m.value[row] = head
	m.value[row+1] = tail

	m.Col = 0
	m.Row++
}

// Compile-time assertion that *Model implements bufferState.
var _ bufferState = (*Model)(nil)

// isWhitespaceRune returns true for runes that should break a typing-run
// coalesce. Newlines are normally handled by the InsertNewline case, but
// we include them here as a safety net if any input sequence delivers one
// through the default arm.
func isWhitespaceRune(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}
