// Package textarea provides a utility function for processing Key messages containing runes.
package textarea

import (
	"unicode/utf8"
)

// NewSanitizer constructs a rune sanitizer.
func NewSanitizer(opts ...Option) *sanitizer {
	s := sanitizer{
		replaceNewLine: []rune("\n"),
		replaceTab:     []rune("    "),
	}
	for _, o := range opts {
		s = o(s)
	}
	return &s
}

// Option is the type of option that can be passed to Sanitize().
type Option func(sanitizer) sanitizer

// ReplaceTabs replaces tabs by the specified string.
func ReplaceTabs(tabRepl string) Option {
	return func(s sanitizer) sanitizer {
		s.replaceTab = []rune(tabRepl)
		return s
	}
}

// ReplaceNewlines replaces newline characters by the specified string.
func ReplaceNewlines(nlRepl string) Option {
	return func(s sanitizer) sanitizer {
		s.replaceNewLine = []rune(nlRepl)
		return s
	}
}

func (s *sanitizer) Sanitize(runes []rune) []rune {
	// dstrunes are where we are storing the result.
	dstrunes := runes[:0:len(runes)]
	// copied indicates whether dstrunes is an alias of runes
	// or a copy. We need a copy when dst moves past src.
	// We use this as an optimization to avoid allocating
	// a new rune slice in the common case where the output
	// is smaller or equal to the input.
	copied := false

	for src, r := range runes {
		switch r {
		case utf8.RuneError:
			// skip

		case '\r', '\n':
			if len(dstrunes)+len(s.replaceNewLine) > src && !copied {
				dst := len(dstrunes)
				dstrunes = make([]rune, dst, len(runes)+len(s.replaceNewLine))
				copy(dstrunes, runes[:dst])
				copied = true
			}
			dstrunes = append(dstrunes, s.replaceNewLine...)

		case '\t':
			if len(dstrunes)+len(s.replaceTab) > src && !copied {
				dst := len(dstrunes)
				dstrunes = make([]rune, dst, len(runes)+len(s.replaceTab))
				copy(dstrunes, runes[:dst])
				copied = true
			}
			dstrunes = append(dstrunes, s.replaceTab...)

		// case unicode.IsControl(r):
		// Other control characters: skip.

		default:
			// Keep the character.
			dstrunes = append(dstrunes, runes[src])
		}
	}
	return dstrunes
}

type sanitizer struct {
	replaceNewLine []rune
	replaceTab     []rune
}
