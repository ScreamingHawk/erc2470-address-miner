package logger

import (
	"io"
	"log"
	"os"
)

// Log flags
const (
	LstdFlags     = log.LstdFlags
	Lmicroseconds = log.Lmicroseconds
)

// Logger wraps the standard log.Logger with additional functionality
type Logger struct {
	*log.Logger
}

// New creates a new logger
func New() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// NewWriter creates a new logger that writes to the provided writer
func NewWriter(w io.Writer) *Logger {
	return &Logger{
		Logger: log.New(w, "", log.LstdFlags),
	}
}

// SetOutput sets the output destination for the logger
func (l *Logger) SetOutput(w io.Writer) {
	l.Logger.SetOutput(w)
}

// SetFlags sets the output flags for the logger
func (l *Logger) SetFlags(flag int) {
	l.Logger.SetFlags(flag)
}
