package logger

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

var instance zerolog.Logger

func Configure(output io.Writer) *zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339
	instance = zerolog.New(colorableWriter(output)).With().Timestamp().Caller().Logger()
	return &instance
}

func ConfigureFile(console io.Writer, path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	consoleOutput := colorableWriter(console)
	if supportsColor(console) {
		consoleOutput = zerolog.ConsoleWriter{
			Out:        consoleOutput,
			TimeFormat: time.RFC3339,
		}
	}

	zerolog.TimeFieldFormat = time.RFC3339
	instance = zerolog.New(zerolog.MultiLevelWriter(consoleOutput, file)).With().Timestamp().Caller().Logger()
	return file, nil
}

func Get() *zerolog.Logger {
	return &instance
}

func Debug() *zerolog.Event {
	return instance.Debug()
}

func Info() *zerolog.Event {
	return instance.Info()
}

func Warn() *zerolog.Event {
	return instance.Warn()
}

func Error() *zerolog.Event {
	return instance.Error()
}

func colorableWriter(output io.Writer) io.Writer {
	switch output {
	case os.Stdout:
		return colorable.NewColorableStdout()
	case os.Stderr:
		return colorable.NewColorableStderr()
	default:
		return output
	}
}

func supportsColor(output io.Writer) bool {
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled || strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}

	file, ok := output.(*os.File)
	if !ok {
		return false
	}

	return isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd())
}

func init() {
	Configure(os.Stdout)
}
