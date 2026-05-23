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
	line, err := readLineAnyEOL(reader)                     // Block until Enter using either LF or CR line endings (SSH PTY can send CR)
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

// readLineAnyEOL reads bytes until LF or CR and returns the line without the terminator.
// SSH PTY sessions may send Enter as CR-only, while local terminals often send LF or CRLF.
func readLineAnyEOL(reader *bufio.Reader) (string, error) {
	var builder strings.Builder // Collect input bytes until an end-of-line terminator is encountered
	for {                      // Keep reading one byte at a time until CR, LF, or EOF
		currentByte, err := reader.ReadByte() // Read next byte from buffered input stream
		if err != nil {                        // Non-nil means EOF or unexpected read failure
			if err == io.EOF { // EOF may happen with or without pending input
				if builder.Len() > 0 { // Return partial input when stream ends after typed characters
					return builder.String(), nil // Treat partial line at EOF as valid user input
				}
				return "", io.EOF // No pending input at EOF -- propagate clean session termination
			}
			return "", err // Propagate unexpected read errors to SafeInput for wrapping
		}
		if currentByte == '\n' { // LF terminates the line on most local terminals
			return builder.String(), nil // Return collected bytes without terminator
		}
		if currentByte == '\r' { // CR terminates the line in many SSH PTY configurations
			nextBytes, peekErr := reader.Peek(1) // Check whether CR is followed by LF (CRLF sequence)
			if peekErr == nil && len(nextBytes) == 1 && nextBytes[0] == '\n' { // Detect CRLF pair safely
				_, _ = reader.ReadByte() // Consume the LF after CR so next read starts at fresh input
			}
			return builder.String(), nil // Return collected bytes without CR terminator
		}
		if currentByte == '\x00' && builder.Len() > 0 { // Some PTY stacks send CR-NUL, where NUL marks end of line
			return builder.String(), nil // Treat NUL as Enter terminator when user already typed characters
		}
		builder.WriteByte(currentByte) // Append regular character byte to the in-progress line buffer
	}
}
