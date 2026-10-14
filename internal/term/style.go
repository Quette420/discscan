// Package term contains small terminal helpers: colors and a progress line.
package term

import (
	"os"

	"github.com/gsaranov/discscan/internal/platform"
)

// Style applies ANSI colors when enabled and is a no-op otherwise.
type Style struct {
	on bool
}

// NewStyle enables colors for interactive terminals unless NO_COLOR is set.
func NewStyle() Style {
	if os.Getenv("NO_COLOR") != "" {
		return Style{}
	}
	return Style{on: platform.EnableColors()}
}

func (s Style) wrap(code, text string) string {
	if !s.on || text == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func (s Style) Bold(t string) string    { return s.wrap("1", t) }
func (s Style) Dim(t string) string     { return s.wrap("2", t) }
func (s Style) Red(t string) string     { return s.wrap("31", t) }
func (s Style) Green(t string) string   { return s.wrap("32", t) }
func (s Style) Yellow(t string) string  { return s.wrap("33", t) }
func (s Style) Cyan(t string) string    { return s.wrap("36", t) }
func (s Style) Magenta(t string) string { return s.wrap("35", t) }
