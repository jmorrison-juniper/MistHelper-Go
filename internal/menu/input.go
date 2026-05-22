// Package menu provides the interactive TUI menu, entry registry, and dispatcher.
// All user input flows through SafeInput so EOF from SSH/container restarts is handled cleanly.
package menu

import (
	"bufio"    // for bufio.Reader -- buffered line-at-a-time stdin reads
	"fmt"      // for fmt.Print -- display the prompt before blocking
	"io"       // for io.EOF -- sentinel for clean session termination
	"log/slog" // for slog.Info / slog.Debug -- structured logging (Go 1.21+)
	"strings"  // for strings.TrimRight -- strip \r\n from the returned line
)

// SafeInput reads one line from reader, trimming whitespace.
// Returns ("", io.EOF) when the reader is exhausted (SSH session closed, container restart).
// term is the io.Writer where prompts are displayed (os.Stdout for local, SSH channel for remote).
// context is a label used in log messages to identify which prompt caused the EOF.
func SafeInput(reader *bufio.Reader, term io.Writer, prompt string, context string) (string, error) {
	_, _ = fmt.Fprint(term, prompt)                         // Display the prompt to the terminal writer (local stdout or SSH channel)
	slog.Info("waiting for user input", "context", context) // Log the read attempt so operators can trace interactive sessions
	line, err := reader.ReadString('\n')                    // Block until the user presses Enter or the session closes
	if err != nil {                                         // Non-nil error means EOF or an unexpected I/O failure
		if err == io.EOF { // EOF is a normal, expected end of an SSH session or container restart
			slog.Info("EOF detected", "context", context) // Log the clean termination so operators know why input stopped
			return "", io.EOF                             // Return the io.EOF sentinel -- caller decides whether to stop or retry
		}
		return "", fmt.Errorf("reading input in %s: %w", context, err) // Wrap unexpected read errors with context for the caller
	}
	result := strings.TrimRight(line, "\r\n")                                    // Strip trailing carriage-return and newline from the raw line
	slog.Debug("received user input", "context", context, "length", len(result)) // Log after read so callers can see input length without logging secrets
	return result, nil                                                           // Return the cleaned input string to the caller
}
