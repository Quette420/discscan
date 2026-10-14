package term

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gsaranov/discscan/internal/scan"
	"github.com/gsaranov/discscan/internal/units"
)

// Progress redraws a single status line with live scan counters.
type Progress struct {
	w     io.Writer
	st    Style
	stats *scan.Stats
	start time.Time
	done  chan struct{}
	exit  chan struct{}
	width int
}

// StartProgress begins rendering progress for stats to w (usually stderr).
func StartProgress(w io.Writer, st Style, stats *scan.Stats) *Progress {
	p := &Progress{
		w:     w,
		st:    st,
		stats: stats,
		start: time.Now(),
		done:  make(chan struct{}),
		exit:  make(chan struct{}),
	}

	go p.loop()

	return p
}

// Stop clears the status line and waits for the renderer to finish.
func (p *Progress) Stop() {
	close(p.done)
	<-p.exit
	p.print("")
}

func (p *Progress) loop() {
	defer close(p.exit)

	// Braille spinner on modern terminals, ASCII on legacy consoles.
	spinner := []string{"|", "/", "-", `\`}
	if p.st.on {
		spinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	}
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	for i := 0; ; i++ {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			c := p.stats.Snapshot()
			line := fmt.Sprintf("%s Scanning… %s files · %s folders · %s · %s",
				p.st.Cyan(spinner[i%len(spinner)]),
				units.FormatCount(c.Files),
				units.FormatCount(c.Dirs),
				units.FormatSize(c.Bytes),
				time.Since(p.start).Round(time.Second),
			)
			p.print(line + p.st.Dim("  (Ctrl+C to stop)"))
		}
	}
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// print overwrites the current line, padding with spaces so a shorter line
// fully covers a longer previous one without relying on ANSI erase codes.
func (p *Progress) print(line string) {
	n := utf8.RuneCountInString(ansi.ReplaceAllString(line, ""))
	pad := ""
	if p.width > n {
		pad = strings.Repeat(" ", p.width-n)
	}
	p.width = n

	fmt.Fprint(p.w, "\r"+line+pad+"\r")
}
