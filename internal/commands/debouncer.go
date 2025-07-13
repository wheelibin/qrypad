package commands

import (
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Debouncer struct {
	mu     sync.Mutex
	timers map[string]*time.Timer
	delay  time.Duration
	out    chan tea.Msg
}

func NewDebouncer(delay time.Duration, out chan tea.Msg) *Debouncer {
	return &Debouncer{
		delay:  delay,
		timers: make(map[string]*time.Timer),
		out:    out,
	}
}

func (d *Debouncer) Trigger(key string, cmd tea.Cmd) tea.Cmd {
	d.mu.Lock()
	defer d.mu.Unlock()

	if existing, ok := d.timers[key]; ok {
		existing.Stop()
	}

	d.timers[key] = time.AfterFunc(d.delay, func() {
		msg := cmd()
		d.out <- msg
	})

	return nil
}
