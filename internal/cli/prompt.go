package cli

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gsaranov/discscan/internal/term"
)

// prompter reads answers to interactive questions.
type prompter struct {
	in  *bufio.Reader
	out io.Writer
	st  term.Style
}

func newPrompter(in io.Reader, out io.Writer, st term.Style) *prompter {
	return &prompter{in: bufio.NewReader(in), out: out, st: st}
}

// ask shows a question with an optional default and returns the trimmed
// answer. An empty answer yields the default; closed input yields io.EOF.
func (p *prompter) ask(question, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(p.out, "%s %s: ", question, p.st.Dim("["+def+"]"))
	} else {
		fmt.Fprintf(p.out, "%s: ", question)
	}

	line, err := p.in.ReadString('\n')
	line = strings.TrimSpace(line)

	if err != nil && line == "" {
		fmt.Fprintln(p.out)
		return "", io.EOF
	}

	if line == "" {
		return def, nil
	}

	return line, nil
}

// askParsed repeats the question until parse accepts the answer.
func askParsed[T any](p *prompter, question, def string, parse func(string) (T, error)) (T, error) {
	for {
		answer, err := p.ask(question, def)
		if err != nil {
			var zero T
			return zero, err
		}

		v, err := parse(answer)
		if err == nil {
			return v, nil
		}

		fmt.Fprintln(p.out, p.st.Red("  "+err.Error()))
	}
}

func (p *prompter) askInt(question string, def, lo, hi int) (int, error) {
	return askParsed(p, question, strconv.Itoa(def), func(s string) (int, error) {
		n, err := strconv.Atoi(s)
		if err != nil || n < lo || n > hi {
			return 0, fmt.Errorf("enter a number from %d to %d", lo, hi)
		}
		return n, nil
	})
}
