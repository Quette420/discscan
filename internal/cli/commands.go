package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gsaranov/discscan/internal/finder"
	"github.com/gsaranov/discscan/internal/platform"
	"github.com/gsaranov/discscan/internal/report"
	"github.com/gsaranov/discscan/internal/units"
)

const usage = `discscan — find out what is eating your disk space.

Usage:
  discscan                          interactive menu
  discscan largest [flags] [path]   largest files
  discscan recent  [flags] [path]   large files that appeared recently
  discscan folders [flags] [path]   largest folders
  discscan apps    [flags] [path]   programs, games and app data by total size
  discscan drives                   list drives with free space
  discscan version                  print version

Path defaults to the current directory. Run 'discscan <command> -h'
for command flags.

Examples:
  discscan largest -n 100 C:\
  discscan recent -min 1GB -within 3d D:\
  discscan folders -depth 2 -csv folders.csv C:\Users
  discscan apps -n 50 C:\
`

// common holds flags shared by all scanning commands.
type common struct {
	workers int
	csv     string
	quiet   bool
	verbose bool
	noHints bool
}

func (a *App) command(args []string) int {
	name, rest := args[0], args[1:]

	switch name {
	case "largest", "top", "big":
		return a.cmdLargest(rest)
	case "recent", "new":
		return a.cmdRecent(rest)
	case "folders", "dirs":
		return a.cmdFolders(rest)
	case "apps", "programs", "games":
		return a.cmdApps(rest)
	case "drives":
		return a.cmdDrives()
	case "version", "-v", "--version":
		fmt.Fprintln(a.out, "discscan", a.version)
		return 0
	case "help", "-h", "--help", "/?":
		fmt.Fprint(a.out, usage)
		return 0
	default:
		fmt.Fprintf(a.err, "unknown command %q\n\n%s", name, usage)
		return 2
	}
}

func (a *App) newFlagSet(name, summary string) (*flag.FlagSet, *common) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.err)
	fs.Usage = func() {
		fmt.Fprintf(a.err, "%s\n\nUsage: discscan %s [flags] [path]\n\nFlags:\n", summary, name)
		fs.PrintDefaults()
	}

	c := &common{}
	fs.IntVar(&c.workers, "workers", runtime.NumCPU()*2, "number of parallel directory readers")
	fs.StringVar(&c.csv, "csv", "", "also save results to this CSV file")
	fs.BoolVar(&c.quiet, "q", false, "do not show scan progress")
	fs.BoolVar(&c.verbose, "v", false, "list read errors after the results")
	fs.BoolVar(&c.noHints, "no-hints", false, "hide cleanup hints under results")

	return fs, c
}

// parse parses flags and a single optional path, allowing flags both before
// and after the path ("discscan recent D:\ -min 1GB" works too).
func (a *App) parse(fs *flag.FlagSet, args []string) (string, bool) {
	var paths []string

	for {
		if err := fs.Parse(args); err != nil {
			return "", false
		}
		if fs.NArg() == 0 {
			break
		}
		paths = append(paths, fs.Arg(0))
		args = fs.Args()[1:]
	}

	switch len(paths) {
	case 0:
		return ".", true
	case 1:
		return paths[0], true
	default:
		fmt.Fprintf(a.err, "expected one path, got %d: %s\n", len(paths), strings.Join(paths, ", "))
		return "", false
	}
}

func (a *App) cmdLargest(args []string) int {
	fs, c := a.newFlagSet("largest", "Show the largest files under a path.")
	n := fs.Int("n", 50, "number of files to show")
	minSize := fs.String("min", "0", "ignore files smaller than this (e.g. 100MB, 1.5GB)")

	root, ok := a.parse(fs, args)
	if !ok {
		return 2
	}

	minBytes, err := units.ParseSize(*minSize)
	if err != nil {
		return a.fail(err)
	}

	var rep *finder.FileReport

	err = a.scanWith(c.workers, c.quiet, func(ctx context.Context, env finder.Env) (err error) {
		rep, err = finder.Largest(ctx, env, root, *n, minBytes)
		return err
	})
	if err != nil {
		return a.fail(err)
	}

	a.printer(!c.noHints).Largest(rep)

	return a.finish(c, rep.Summary, func(path string) error { return report.FilesCSV(path, rep) })
}

func (a *App) cmdRecent(args []string) int {
	fs, c := a.newFlagSet("recent", "Show large files that appeared or changed recently, newest first.")
	n := fs.Int("n", 100, "maximum number of files to show")
	minSize := fs.String("min", "100MB", "ignore files smaller than this (e.g. 500MB, 1GB)")
	within := fs.String("within", "7d", "look-back period (e.g. 24h, 3d, 2w)")

	root, ok := a.parse(fs, args)
	if !ok {
		return 2
	}

	minBytes, err := units.ParseSize(*minSize)
	if err != nil {
		return a.fail(err)
	}

	period, err := units.ParseWithin(*within)
	if err != nil {
		return a.fail(err)
	}

	var rep *finder.FileReport

	err = a.scanWith(c.workers, c.quiet, func(ctx context.Context, env finder.Env) (err error) {
		rep, err = finder.Recent(ctx, env, root, *n, minBytes, time.Now().Add(-period))
		return err
	})
	if err != nil {
		return a.fail(err)
	}

	a.printer(!c.noHints).Recent(rep)

	return a.finish(c, rep.Summary, func(path string) error { return report.FilesCSV(path, rep) })
}

func (a *App) cmdFolders(args []string) int {
	fs, c := a.newFlagSet("folders", "Show the largest folders under a path (sizes include subfolders).")
	n := fs.Int("n", 30, "number of folders to show")
	depth := fs.Int("depth", 0, "only list folders up to this many levels below path (0 = any)")

	root, ok := a.parse(fs, args)
	if !ok {
		return 2
	}

	var rep *finder.FolderReport

	err := a.scanWith(c.workers, c.quiet, func(ctx context.Context, env finder.Env) (err error) {
		rep, err = finder.Folders(ctx, env, root, *n, *depth)
		return err
	})
	if err != nil {
		return a.fail(err)
	}

	a.printer(!c.noHints).Folders(rep)

	return a.finish(c, rep.Summary, func(path string) error { return report.FoldersCSV(path, rep) })
}

func (a *App) cmdApps(args []string) int {
	fs, c := a.newFlagSet("apps", "Show program, game and app data folders by total size (counts all small files).")
	n := fs.Int("n", 30, "number of folders to show")

	root, ok := a.parse(fs, args)
	if !ok {
		return 2
	}

	var rep *finder.AppReport

	err := a.scanWith(c.workers, c.quiet, func(ctx context.Context, env finder.Env) (err error) {
		rep, err = finder.Apps(ctx, env, root, *n)
		return err
	})
	if err != nil {
		return a.fail(err)
	}

	a.printer(!c.noHints).Apps(rep)

	return a.finish(c, rep.Summary, func(path string) error { return report.AppsCSV(path, rep) })
}

func (a *App) printer(hints bool) report.Printer {
	return report.Printer{W: a.out, St: a.st, Hints: hints}
}

func (a *App) cmdDrives() int {
	drives, err := platform.Drives()
	if err != nil {
		return a.fail(err)
	}

	a.printDrives(a.out, drives, false)

	return 0
}

// finish prints error details and writes the CSV file if requested.
func (a *App) finish(c *common, s finder.Summary, save func(string) error) int {
	if c.verbose {
		report.Printer{W: a.err, St: a.st}.Errors(s)
	}

	if c.csv != "" {
		if err := save(c.csv); err != nil {
			return a.fail(fmt.Errorf("cannot write CSV: %w", err))
		}

		abs, _ := filepath.Abs(c.csv)
		fmt.Fprintln(a.out, a.st.Green("Saved: ")+abs)
	}

	if s.Interrupted {
		return 130
	}

	return 0
}

func (a *App) fail(err error) int {
	fmt.Fprintln(a.err, a.st.Red("error: ")+err.Error())
	return 1
}

// printDrives renders the drive list; numbered adds menu indexes.
func (a *App) printDrives(w io.Writer, drives []platform.Drive, numbered bool) {
	for i, d := range drives {
		prefix := ""
		if numbered {
			prefix = a.st.Cyan(fmt.Sprintf("  %2d  ", i+1))
		}

		label := d.Label
		if label == "" {
			label = "-"
		}

		usage := ""
		if d.Total > 0 {
			used := float64(d.Used()) / float64(d.Total)
			bar := report.Bar(used, 12)
			if used >= 0.9 {
				bar = a.st.Red(bar)
			} else {
				bar = a.st.Green(bar)
			}

			usage = fmt.Sprintf("%s %3.0f%%  %s free of %s",
				bar,
				used*100,
				units.FormatSize(int64(d.Free)),
				units.FormatSize(int64(d.Total)),
			)
		}

		fmt.Fprintf(w, "%s%s %-14s %-9s %s\n", prefix, a.st.Bold(fmt.Sprintf("%-6s", d.Path)), truncate(label, 14), d.Kind, usage)
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// errBack signals "return to the main menu" from nested prompts.
var errBack = errors.New("back")
