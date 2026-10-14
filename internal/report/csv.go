package report

import (
	"encoding/csv"
	"os"
	"strconv"
	"time"

	"github.com/gsaranov/discscan/internal/finder"
	"github.com/gsaranov/discscan/internal/hints"
	"github.com/gsaranov/discscan/internal/units"
)

// FilesCSV writes a file list to path.
func FilesCSV(path string, rep *finder.FileReport) error {
	rows := [][]string{{"Rank", "SizeBytes", "Size", "Modified", "Created", "Path", "Hint", "HintNote"}}

	for i, f := range rep.Files {
		created := ""
		if !f.Created.IsZero() {
			created = f.Created.Format(time.DateTime)
		}

		rows = append(rows, append([]string{
			strconv.Itoa(i + 1),
			strconv.FormatInt(f.Size, 10),
			units.FormatSize(f.Size),
			f.ModTime.Format(time.DateTime),
			created,
			f.Path,
		}, hintCells(hints.ForFile(f.Path))...))
	}

	return writeCSV(path, rows)
}

// FoldersCSV writes a folder list to path.
func FoldersCSV(path string, rep *finder.FolderReport) error {
	rows := [][]string{{"Rank", "SizeBytes", "Size", "Files", "Depth", "Path", "Hint", "HintNote"}}

	for i, f := range rep.Folders {
		rows = append(rows, append([]string{
			strconv.Itoa(i + 1),
			strconv.FormatInt(f.Size, 10),
			units.FormatSize(f.Size),
			strconv.FormatInt(f.Files, 10),
			strconv.Itoa(f.Depth),
			f.Path,
		}, hintCells(hints.ForDir(f.Path))...))
	}

	return writeCSV(path, rows)
}

// AppsCSV writes a program/game folder list to path.
func AppsCSV(path string, rep *finder.AppReport) error {
	rows := [][]string{{"Rank", "SizeBytes", "Size", "Files", "Kind", "Path", "Hint", "HintNote"}}

	for i, a := range rep.Apps {
		rows = append(rows, append([]string{
			strconv.Itoa(i + 1),
			strconv.FormatInt(a.Size, 10),
			units.FormatSize(a.Size),
			strconv.FormatInt(a.Files, 10),
			a.Kind,
			a.Path,
		}, hintCells(hints.ForDir(a.Path))...))
	}

	return writeCSV(path, rows)
}

func hintCells(h hints.Hint, ok bool) []string {
	if !ok {
		return []string{"", ""}
	}
	return []string{h.Action.Label(), h.Note}
}

// writeCSV writes semicolon-separated UTF-8 with a BOM, which Excel on
// Windows opens correctly out of the box (including non-ASCII paths).
func writeCSV(path string, rows [][]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	if _, err := f.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		f.Close()
		return err
	}

	w := csv.NewWriter(f)
	w.Comma = ';'

	if err := w.WriteAll(rows); err != nil {
		f.Close()
		return err
	}

	return f.Close()
}
