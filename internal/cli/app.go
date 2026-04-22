package cli

import (
	"io"
	"os"
)

type Streams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

func DefaultStreams() Streams {
	return Streams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
}

type App struct {
	Streams Streams
	Verbose bool
}

func (a *App) verboseWriter() io.Writer {
	if a.Verbose {
		return a.Streams.ErrOut
	}
	return nil
}
