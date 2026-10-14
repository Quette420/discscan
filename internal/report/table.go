// Package report renders scan results as console tables and CSV files.
package report

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gsaranov/discscan/internal/finder"
	"github.com/gsaranov/discscan/internal/hints"
	"github.com/gsaranov/discscan/internal/term"
	"github.com/gsaranov/discscan/internal/units"
)

const timeLayout = "2006-01-02 15:04"

// Printer renders reports to a writer.
type Printer struct {
	W     io.Writer
	St    term.Style
	Hints bool // show "what can I do with this" advice under rows
}

// sizeCell colors big sizes so the worst offenders stand out.
func (p Printer) sizeCell(size int64) string {
	cell := fmt.Sprintf("%10s", units.FormatSize(size))

	switch {
	case size >= 10*units.GB:
		return p.St.Red(cell)
	case size >= units.GB:
		return p.St.Yellow(cell)
	default:
		return cell
	}
}

// hintTally sums sizes per suggested action for the footer.
type hintTally map[hints.Action]int64

// hint prints the advice line under a row and records it in the tally.
func (p Printer) hint(h hints.Hint, ok bool, size int64, tally hintTally) {
	if !p.Hints || !ok {
		return
	}

	tally[h.Action] += size

	fmt.Fprintf(p.W, "      %s %s %s\n", p.St.Dim("↳"), p.actionTag(h.Action), p.St.Dim(h.Note))
}

func (p Printer) actionTag(a hints.Action) string {
	tag := fmt.Sprintf("%-7s", a.Label())

	switch a {
	case hints.Delete:
		return p.St.Green(tag)
	case hints.Cache:
		return p.St.Cyan(tag)
	case hints.Move:
		return p.St.Yellow(tag)
	case hints.Careful:
		return p.St.Red(tag)
	default:
		return p.St.Magenta(tag)
	}
}

// actionLabel is actionTag without column padding.
func (p Printer) actionLabel(a hints.Action) string {
	return strings.TrimSpace(p.actionTag(a))
}

// tallyLine prints how much of the listed space each action covers.
func (p Printer) tallyLine(tally hintTally) {
	if !p.Hints || len(tally) == 0 {
		return
	}

	var parts []string

	for _, a := range []hints.Action{hints.Delete, hints.Cache, hints.Move, hints.System, hints.Careful} {
		if size, ok := tally[a]; ok {
			parts = append(parts, p.actionLabel(a)+" "+strings.TrimSpace(units.FormatSize(size)))
		}
	}

	fmt.Fprintf(p.W, "\n%s %s\n", p.St.Bold("Hints:"), strings.Join(parts, p.St.Dim(" · ")))
}

// Largest prints files ranked by size.
func (p Printer) Largest(rep *finder.FileReport) {
	if len(rep.Files) == 0 {
		fmt.Fprintln(p.W, p.St.Dim("No files matched."))
		p.Summary(rep.Summary)
		return
	}

	fmt.Fprintln(p.W, p.St.Bold(fmt.Sprintf("%4s  %10s  %-16s  %s", "#", "SIZE", "MODIFIED", "PATH")))

	tally := hintTally{}

	for i, f := range rep.Files {
		fmt.Fprintf(p.W, "%4d  %s  %s  %s\n",
			i+1,
			p.sizeCell(f.Size),
			p.St.Dim(f.ModTime.Format(timeLayout)),
			f.Path,
		)

		h, ok := hints.ForFile(f.Path)
		p.hint(h, ok, f.Size, tally)
	}

	p.tallyLine(tally)
	p.Summary(rep.Summary)
}

// Recent prints files ordered from newest to oldest.
func (p Printer) Recent(rep *finder.FileReport) {
	if len(rep.Files) == 0 {
		fmt.Fprintln(p.W, p.St.Dim("No large files appeared in this period."))
		p.Summary(rep.Summary)
		return
	}

	now := time.Now()

	fmt.Fprintln(p.W, p.St.Bold(fmt.Sprintf("%4s  %10s  %-16s  %9s  %s", "#", "SIZE", "CHANGED", "AGE", "PATH")))

	var total int64
	tally := hintTally{}

	for i, f := range rep.Files {
		changed := f.Changed()
		total += f.Size

		fmt.Fprintf(p.W, "%4d  %s  %s  %s  %s\n",
			i+1,
			p.sizeCell(f.Size),
			p.St.Dim(changed.Format(timeLayout)),
			p.St.Cyan(fmt.Sprintf("%9s", units.FormatAge(now.Sub(changed)))),
			f.Path,
		)

		h, ok := hints.ForFile(f.Path)
		p.hint(h, ok, f.Size, tally)
	}

	fmt.Fprintf(p.W, "\n%s %s in %d files\n",
		p.St.Bold("Total:"),
		p.St.Yellow(units.FormatSize(total)),
		len(rep.Files),
	)

	p.tallyLine(tally)
	p.Summary(rep.Summary)
}

// Folders prints folders ranked by total size with their share of the scan.
func (p Printer) Folders(rep *finder.FolderReport) {
	if len(rep.Folders) == 0 {
		fmt.Fprintln(p.W, p.St.Dim("No folders found."))
		p.Summary(rep.Summary)
		return
	}

	fmt.Fprintln(p.W, p.St.Bold(fmt.Sprintf("%4s  %10s  %-17s  %10s  %s", "#", "SIZE", "SHARE", "FILES", "PATH")))

	tally := hintTally{}

	for i, f := range rep.Folders {
		fmt.Fprintf(p.W, "%4d  %s  %s  %10s  %s\n",
			i+1,
			p.sizeCell(f.Size),
			p.share(f.Size, rep.Counters.Bytes),
			units.FormatCount(f.Files),
			f.Path,
		)

		h, ok := hints.ForDir(f.Path)
		p.hint(h, ok, f.Size, tally)
	}

	p.Summary(rep.Summary)
}

// Apps prints programs, games and app data folders ranked by size.
func (p Printer) Apps(rep *finder.AppReport) {
	if len(rep.Apps) == 0 {
		fmt.Fprintln(p.W, p.St.Dim("No program or game folders found."))
		p.Summary(rep.Summary)
		return
	}

	fmt.Fprintln(p.W, p.St.Bold(fmt.Sprintf("%4s  %10s  %-17s  %10s  %-19s  %s", "#", "SIZE", "SHARE", "FILES", "KIND", "PATH")))

	tally := hintTally{}

	for i, a := range rep.Apps {
		fmt.Fprintf(p.W, "%4d  %s  %s  %10s  %s  %s\n",
			i+1,
			p.sizeCell(a.Size),
			p.share(a.Size, rep.Counters.Bytes),
			units.FormatCount(a.Files),
			p.St.Dim(fmt.Sprintf("%-19s", a.Kind)),
			a.Path,
		)

		h, ok := hints.ForDir(a.Path)
		p.hint(h, ok, a.Size, tally)
	}

	p.tallyLine(tally)
	p.Summary(rep.Summary)
}

// share renders a bar and a percentage of total, 17 columns wide.
func (p Printer) share(size, total int64) string {
	fraction := 0.0
	if total > 0 {
		fraction = float64(size) / float64(total)
	}

	return fmt.Sprintf("%s %5.1f%%", p.St.Cyan(Bar(fraction, 10)), fraction*100)
}

// Summary prints scan statistics below a table.
func (p Printer) Summary(s finder.Summary) {
	c := s.Counters

	line := fmt.Sprintf("Scanned %s files in %s folders (%s) under %s in %s",
		units.FormatCount(c.Files),
		units.FormatCount(c.Dirs),
		units.FormatSize(c.Bytes),
		s.Root,
		s.Elapsed.Round(10*time.Millisecond),
	)

	fmt.Fprintln(p.W)
	fmt.Fprintln(p.W, p.St.Dim(line))

	if s.Interrupted {
		fmt.Fprintln(p.W, p.St.Yellow("Scan was interrupted — results are partial."))
	}

	if c.Errors > 0 {
		fmt.Fprintln(p.W, p.St.Dim(fmt.Sprintf(
			"%s items could not be read (usually access denied; run as Administrator to include them).",
			units.FormatCount(c.Errors),
		)))
	}
}

// Errors prints the collected error samples.
func (p Printer) Errors(s finder.Summary) {
	for _, e := range s.Errors {
		fmt.Fprintln(p.W, p.St.Dim("  "+e))
	}

	if more := s.Counters.Errors - int64(len(s.Errors)); more > 0 {
		fmt.Fprintln(p.W, p.St.Dim(fmt.Sprintf("  …and %s more", units.FormatCount(more))))
	}
}

// Bar renders a fraction in [0, 1] as a fixed-width bar.
func Bar(fraction float64, width int) string {
	fraction = min(max(fraction, 0), 1)
	filled := int(fraction*float64(width) + 0.5)

	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
