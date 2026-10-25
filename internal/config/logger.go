package config

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

const formatJSON = "json"

func InitLogger(cfg Logger) zerolog.Logger {
	level := zerolog.InfoLevel
	if newLevel, err := zerolog.ParseLevel(cfg.LogLevel); err == nil {
		level = newLevel
	}

	var out io.Writer = os.Stdout
	if cfg.LogFormat != formatJSON {
		out = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.StampMicro}
	}

	return zerolog.New(out).Level(level).
		With().Caller().Timestamp().
		Logger()
}
