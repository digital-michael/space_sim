package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"syscall"

	"golang.org/x/term"
)

// lineReader wraps an io.Reader and provides readline-style line input.
//
// When the reader is a TTY it enables raw terminal mode and supports:
//   - Up/down arrows to navigate command history
//   - ESC to clear the current line
//   - Backspace/DEL to delete the last character
//   - TAB to cycle through completions supplied by TabCompleter
//   - Ctrl-C and Ctrl-D to signal EOF
//
// When the reader is not a TTY (pipes, test strings) it falls back to a
// plain bufio.Scanner-based line reader so existing tests are unaffected.
type lineReader struct {
	in      io.Reader
	scanner *bufio.Scanner // non-nil only on the dumb (non-TTY) path
	history []string
	histIdx int
	draft   string
	// TabCompleter returns a list of completions for the given partial input.
	// Nil means no completion. Called only on the raw TTY path.
	TabCompleter func(partial string) []string
	tabMatches   []string // current cycle candidates; nil = cycle not active
	tabIdx       int      // index into tabMatches for next TAB hit
}

func newLineReader(in io.Reader) *lineReader {
	lr := &lineReader{in: in}
	if _, isTTY := ttyFd(in); !isTTY {
		lr.scanner = bufio.NewScanner(in)
	}
	return lr
}

// readLine prints prompt and returns the next trimmed line. Returns io.EOF when
// the input is exhausted or the user signals an exit (Ctrl-C, Ctrl-D, EOF).
func (lr *lineReader) readLine(prompt string) (string, error) {
	if lr.scanner != nil {
		return lr.readDumb(prompt)
	}
	f := lr.in.(*os.File)
	fd := int(f.Fd())
	return lr.readRaw(f, fd, prompt)
}

// appendHistory adds a non-empty line to the in-memory history.
func (lr *lineReader) appendHistory(line string) {
	if line != "" {
		lr.history = append(lr.history, line)
	}
}

// ── dumb path (non-TTY) ──────────────────────────────────────────────────────

func (lr *lineReader) readDumb(prompt string) (string, error) {
	fmt.Print(prompt)
	if lr.scanner.Scan() {
		return lr.scanner.Text(), nil
	}
	if err := lr.scanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

// ── raw TTY path ─────────────────────────────────────────────────────────────

func (lr *lineReader) readRaw(f *os.File, fd int, prompt string) (string, error) {
	old, err := term.MakeRaw(fd)
	if err != nil {
		// Fall back to dumb if we can't enter raw mode.
		lr.scanner = bufio.NewScanner(lr.in)
		return lr.readDumb(prompt)
	}
	defer term.Restore(fd, old) //nolint:errcheck

	fmt.Print(prompt)

	var buf []byte
	lr.histIdx = len(lr.history)
	lr.draft = ""

	b := make([]byte, 1)
	for {
		if _, err := f.Read(b); err != nil {
			if len(buf) > 0 {
				return string(buf), nil
			}
			return "", io.EOF
		}

		switch b[0] {
		case 0x03: // Ctrl-C
			fmt.Print("\r\n")
			return "", io.EOF

		case 0x04: // Ctrl-D — EOF only on empty line (bash convention)
			if len(buf) == 0 {
				fmt.Print("\r\n")
				return "", io.EOF
			}

		case 0x0d, 0x0a: // Enter
			fmt.Print("\r\n")
			line := string(buf)
			lr.appendHistory(line)
			lr.tabMatches = nil
			return line, nil

		case 0x09: // TAB — cycle completions
			if lr.TabCompleter == nil {
				continue
			}
			partial := string(buf)
			if lr.tabMatches == nil {
				lr.tabMatches = lr.TabCompleter(partial)
				lr.tabIdx = 0
			}
			if len(lr.tabMatches) == 0 {
				// No matches — ring bell.
				fmt.Print("\x07")
				lr.tabMatches = nil
				continue
			}
			lr.replaceLine(prompt, &buf, lr.tabMatches[lr.tabIdx])
			lr.tabIdx = (lr.tabIdx + 1) % len(lr.tabMatches)
			continue

		case 0x15: // Ctrl-U — clear line
			lr.tabMatches = nil
			lr.eraseLine(prompt, &buf)

		case 0x7f, 0x08: // Backspace / DEL
			lr.tabMatches = nil
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				fmt.Print("\b \b")
			}

		case 0x1b: // ESC — either clear line or start of CSI sequence
			lr.tabMatches = nil
			next, ok := peekByte(fd, f)
			if !ok {
				// Bare ESC — clear.
				lr.eraseLine(prompt, &buf)
				continue
			}
			if next != '[' {
				// ESC + something other than '[' — clear, discard extra byte.
				lr.eraseLine(prompt, &buf)
				continue
			}
			// CSI: read the final byte.
			if _, err := f.Read(b); err != nil {
				continue
			}
			switch b[0] {
			case 'A': // Up arrow
				lr.historyPrev(prompt, &buf)
			case 'B': // Down arrow
				lr.historyNext(prompt, &buf)
				// Left/right/etc. — silently ignore to avoid corrupting the buffer.
			}

		default:
			if b[0] >= 0x20 { // printable ASCII
				lr.tabMatches = nil // any printable char breaks the TAB cycle
				buf = append(buf, b[0])
				fmt.Print(string(b[0]))
			}
		}
	}
}

// eraseLine clears the current input line and re-prints the prompt.
func (lr *lineReader) eraseLine(prompt string, buf *[]byte) {
	fmt.Printf("\r\x1b[K%s", prompt) // CR + erase-to-EOL + prompt
	*buf = (*buf)[:0]
}

// replaceLine replaces the visible input with newContent.
func (lr *lineReader) replaceLine(prompt string, buf *[]byte, newContent string) {
	fmt.Printf("\r\x1b[K%s%s", prompt, newContent)
	*buf = []byte(newContent)
}

func (lr *lineReader) historyPrev(prompt string, buf *[]byte) {
	if lr.histIdx == len(lr.history) {
		lr.draft = string(*buf)
	}
	if lr.histIdx > 0 {
		lr.histIdx--
		lr.replaceLine(prompt, buf, lr.history[lr.histIdx])
	}
}

func (lr *lineReader) historyNext(prompt string, buf *[]byte) {
	if lr.histIdx >= len(lr.history) {
		return
	}
	lr.histIdx++
	var next string
	if lr.histIdx == len(lr.history) {
		next = lr.draft
	} else {
		next = lr.history[lr.histIdx]
	}
	lr.replaceLine(prompt, buf, next)
}

// ── helpers ──────────────────────────────────────────────────────────────────

// ttyFd returns the file descriptor and true when r is an *os.File that is a
// terminal.
func ttyFd(r io.Reader) (int, bool) {
	f, ok := r.(*os.File)
	if !ok {
		return 0, false
	}
	fd := int(f.Fd())
	return fd, term.IsTerminal(fd)
}

// peekByte temporarily sets the file descriptor to non-blocking mode and
// attempts one byte read. Returns (byte, true) if a byte was available
// immediately, (0, false) if the read would block (bare ESC).
func peekByte(fd int, f *os.File) (byte, bool) {
	if err := syscall.SetNonblock(fd, true); err != nil {
		return 0, false
	}
	var b [1]byte
	n, _ := f.Read(b[:])
	syscall.SetNonblock(fd, false) //nolint:errcheck
	if n == 0 {
		return 0, false
	}
	return b[0], true
}
