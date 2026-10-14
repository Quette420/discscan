// Package cli wires the finder, report and platform packages into the
// discscan command-line interface: subcommands and the interactive menu.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"

	"github.com/gsaranov/discscan/internal/finder"
	"github.com/gsaranov/discscan/internal/scan"
	"github.com/gsaranov/discscan/internal/term"
)

// App holds process-wide state for one discscan invocation.
type App struct {
	version string
	in      io.Reader
	out     io.Writer
	err     io.Writer
	st      term.Style
	intr    *interrupter
}

// Run executes discscan with the given arguments (without the program name)
// and returns the process exit code.
func Run(version string, args []string) int {
	a := &App{
		version: version,
		in:      os.Stdin,
		out:     os.Stdout,
		err:     os.Stderr,
		st:      term.NewStyle(),
		intr:    newInterrupter(),
	}

	if len(args) == 0 {
		return a.interactive()
	}

	return a.command(args)
}

// scanWith runs fn with a cancellable context and a live progress line.
// Ctrl+C during the scan stops it and keeps the partial results.
func (a *App) scanWith(workers int, quiet bool, fn func(context.Context, finder.Env) error) error {
	ctx, done := a.intr.scanContext()
	defer done()

	stats := new(scan.Stats)

	var progress *term.Progress
	if !quiet {
		progress = term.StartProgress(a.err, a.st, stats)
	}

	err := fn(ctx, finder.Env{Workers: workers, Stats: stats})

	if progress != nil {
		progress.Stop()
	}

	return err
}

// interrupter routes Ctrl+C: during a scan it cancels the scan, otherwise
// it exits the program.
type interrupter struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

func newInterrupter() *interrupter {
	it := &interrupter{}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	go func() {
		for range ch {
			it.mu.Lock()
			cancel := it.cancel
			it.cancel = nil
			it.mu.Unlock()

			if cancel != nil {
				cancel()
				continue
			}

			fmt.Fprintln(os.Stderr)
			os.Exit(130)
		}
	}()

	return it
}

func (it *interrupter) scanContext() (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	it.mu.Lock()
	it.cancel = cancel
	it.mu.Unlock()

	return ctx, func() {
		it.mu.Lock()
		it.cancel = nil
		it.mu.Unlock()
		cancel()
	}
}
