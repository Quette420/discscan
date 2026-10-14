package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gsaranov/discscan/internal/finder"
	"github.com/gsaranov/discscan/internal/platform"
	"github.com/gsaranov/discscan/internal/report"
	"github.com/gsaranov/discscan/internal/units"
)

// defaultWorkers lets the scanner choose (2×CPU).
const defaultWorkers = 0

func (a *App) interactive() int {
	p := newPrompter(a.in, a.out, a.st)

	fmt.Fprintf(a.out, "\n  %s %s — see what is eating your disk\n",
		a.st.Bold(a.st.Cyan("discscan")), a.st.Dim(a.version))

	for {
		fmt.Fprintln(a.out)
		fmt.Fprintf(a.out, "  %s  %-20s %s\n", a.st.Cyan("1"), "Largest files", a.st.Dim("the biggest files on a drive"))
		fmt.Fprintf(a.out, "  %s  %-20s %s\n", a.st.Cyan("2"), "Recent large files", a.st.Dim("what filled the drive lately"))
		fmt.Fprintf(a.out, "  %s  %-20s %s\n", a.st.Cyan("3"), "Largest folders", a.st.Dim("where the space goes"))
		fmt.Fprintf(a.out, "  %s  %-20s %s\n", a.st.Cyan("4"), "Programs & games", a.st.Dim("apps, games and app data, all files counted"))
		fmt.Fprintf(a.out, "  %s  %s\n", a.st.Cyan("q"), "Quit")
		fmt.Fprintln(a.out)

		choice, err := p.ask("Choose mode", "1")
		if err != nil {
			return 0
		}

		switch strings.ToLower(choice) {
		case "1":
			err = a.ivLargest(p)
		case "2":
			err = a.ivRecent(p)
		case "3":
			err = a.ivFolders(p)
		case "4":
			err = a.ivApps(p)
		case "q", "quit", "exit", "0":
			return 0
		default:
			fmt.Fprintln(a.out, a.st.Red("  Unknown option: "+choice))
			continue
		}

		switch {
		case errors.Is(err, io.EOF):
			return 0
		case errors.Is(err, errBack):
			continue
		case err != nil:
			fmt.Fprintln(a.err, a.st.Red("error: ")+err.Error())
		}
	}
}

func (a *App) ivLargest(p *prompter) error {
	root, err := a.pickRoot(p)
	if err != nil {
		return err
	}

	n, err := p.askInt("How many files to show", 50, 1, 100000)
	if err != nil {
		return err
	}

	minSize, err := askParsed(p, "Ignore files smaller than", "0", units.ParseSize)
	if err != nil {
		return err
	}

	var rep *finder.FileReport

	err = a.scanWith(defaultWorkers, false, func(ctx context.Context, env finder.Env) (err error) {
		rep, err = finder.Largest(ctx, env, root, n, minSize)
		return err
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(a.out)
	a.printer(true).Largest(rep)

	return a.actions(p, filePaths(rep), func(path string) error { return report.FilesCSV(path, rep) }, csvName("largest", root))
}

func (a *App) ivRecent(p *prompter) error {
	root, err := a.pickRoot(p)
	if err != nil {
		return err
	}

	minSize, err := askParsed(p, "Minimum file size (e.g. 500MB, 1GB)", "100MB", units.ParseSize)
	if err != nil {
		return err
	}

	period, err := askParsed(p, "Appeared within (e.g. 24h, 3d, 2w)", "7d", units.ParseWithin)
	if err != nil {
		return err
	}

	n, err := p.askInt("Maximum files to show", 100, 1, 100000)
	if err != nil {
		return err
	}

	var rep *finder.FileReport

	err = a.scanWith(defaultWorkers, false, func(ctx context.Context, env finder.Env) (err error) {
		rep, err = finder.Recent(ctx, env, root, n, minSize, time.Now().Add(-period))
		return err
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(a.out)
	a.printer(true).Recent(rep)

	return a.actions(p, filePaths(rep), func(path string) error { return report.FilesCSV(path, rep) }, csvName("recent", root))
}

func (a *App) ivFolders(p *prompter) error {
	root, err := a.pickRoot(p)
	if err != nil {
		return err
	}

	n, err := p.askInt("How many folders to show", 30, 1, 100000)
	if err != nil {
		return err
	}

	depth, err := p.askInt("Max depth below the root (0 = any)", 3, 0, 1000)
	if err != nil {
		return err
	}

	var rep *finder.FolderReport

	err = a.scanWith(defaultWorkers, false, func(ctx context.Context, env finder.Env) (err error) {
		rep, err = finder.Folders(ctx, env, root, n, depth)
		return err
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(a.out)
	a.printer(true).Folders(rep)

	paths := make([]string, len(rep.Folders))
	for i, f := range rep.Folders {
		paths[i] = f.Path
	}

	return a.actions(p, paths, func(path string) error { return report.FoldersCSV(path, rep) }, csvName("folders", root))
}

func (a *App) ivApps(p *prompter) error {
	root, err := a.pickRoot(p)
	if err != nil {
		return err
	}

	n, err := p.askInt("How many to show", 30, 1, 100000)
	if err != nil {
		return err
	}

	var rep *finder.AppReport

	err = a.scanWith(defaultWorkers, false, func(ctx context.Context, env finder.Env) (err error) {
		rep, err = finder.Apps(ctx, env, root, n)
		return err
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(a.out)
	a.printer(true).Apps(rep)

	paths := make([]string, len(rep.Apps))
	for i, app := range rep.Apps {
		paths[i] = app.Path
	}

	return a.actions(p, paths, func(path string) error { return report.AppsCSV(path, rep) }, csvName("apps", root))
}

// pickRoot lets the user choose a drive or type any folder path.
func (a *App) pickRoot(p *prompter) (string, error) {
	drives, err := platform.Drives()
	if err != nil {
		fmt.Fprintln(a.err, a.st.Red("cannot list drives: ")+err.Error())
	}

	fmt.Fprintln(a.out)
	a.printDrives(a.out, drives, true)
	fmt.Fprintf(a.out, "  %s  %s\n", a.st.Cyan(fmt.Sprintf("%2s", "p")), "Enter a folder path")
	fmt.Fprintf(a.out, "  %s  %s\n\n", a.st.Cyan(fmt.Sprintf("%2s", "b")), "Back")

	def := "p"
	if len(drives) > 0 {
		def = "1"
	}

	for {
		choice, err := p.ask("Choose drive", def)
		if err != nil {
			return "", err
		}

		switch strings.ToLower(choice) {
		case "b":
			return "", errBack
		case "p":
			path, err := p.ask("Folder path", "")
			if err != nil {
				return "", err
			}

			root, err := finder.ResolveRoot(strings.Trim(path, `"`))
			if err != nil {
				fmt.Fprintln(a.out, a.st.Red("  "+err.Error()))
				continue
			}

			return root, nil
		}

		if i, err := strconv.Atoi(choice); err == nil && i >= 1 && i <= len(drives) {
			return drives[i-1].Path, nil
		}

		// Accept a typed drive letter ("d", "D:") or a full path as well.
		if root, err := finder.ResolveRoot(driveOrPath(choice)); err == nil {
			return root, nil
		}

		fmt.Fprintln(a.out, a.st.Red("  Unknown choice: "+choice))
	}
}

func driveOrPath(s string) string {
	if len(s) == 1 && unicode.IsLetter(rune(s[0])) {
		return strings.ToUpper(s) + `:\`
	}
	return strings.Trim(s, `"`)
}

// actions lets the user reveal results in the file manager or save a CSV.
func (a *App) actions(p *prompter, paths []string, save func(string) error, defaultCSV string) error {
	if len(paths) == 0 {
		return nil
	}

	hint := a.st.Dim(fmt.Sprintf("[1-%d] show in %s · [s] save CSV · [Enter] menu", len(paths), platform.FileManager))

	for {
		fmt.Fprintln(a.out)

		choice, err := p.ask(hint, "")
		if err != nil {
			return err
		}

		switch strings.ToLower(choice) {
		case "":
			return nil

		case "s":
			name, err := p.ask("Save as", defaultCSV)
			if err != nil {
				return err
			}

			if err := save(name); err != nil {
				fmt.Fprintln(a.out, a.st.Red("  cannot save: "+err.Error()))
				continue
			}

			abs, _ := filepath.Abs(name)
			fmt.Fprintln(a.out, a.st.Green("  Saved: ")+abs)

		default:
			i, err := strconv.Atoi(choice)
			if err != nil || i < 1 || i > len(paths) {
				fmt.Fprintln(a.out, a.st.Red("  Unknown choice: "+choice))
				continue
			}

			if err := platform.Reveal(paths[i-1]); err != nil {
				fmt.Fprintln(a.out, a.st.Red("  cannot open: "+err.Error()))
			}
		}
	}
}

func filePaths(rep *finder.FileReport) []string {
	paths := make([]string, len(rep.Files))
	for i, f := range rep.Files {
		paths[i] = f.Path
	}
	return paths
}

// csvName builds a default file name such as discscan-largest-C-20261014-1530.csv.
func csvName(mode, root string) string {
	tag := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, filepath.Base(root))

	if tag == "" {
		tag = strings.TrimRight(root, `:\/`)
	}
	if tag == "" {
		tag = "root"
	}

	return fmt.Sprintf("discscan-%s-%s-%s.csv", mode, tag, time.Now().Format("20060102-1504"))
}
