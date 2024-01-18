package utils

import (
	"github.com/rs/zerolog"
	"os"
	"time"
)

var log zerolog.Logger

func setLevel(l zerolog.Level) {
	zerolog.SetGlobalLevel(l)
}

func InfoF(format string, v ...interface{}) {
	log.Info().Msgf(format, v...)
}

func DebugF(format string, v ...interface{}) {
	log.Debug().Msgf(format, v...)
}

func ErrorF(format string, v ...interface{}) {
	log.Error().Msgf(format, v...)
}

func WarnF(format string, v ...interface{}) {
	log.Warn().Msgf(format, v...)
}

func PanicF(format string, v ...interface{}) {
	log.Panic().Msgf(format, v...)
}

func init() {
	setLevel(zerolog.DebugLevel)
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log = zerolog.New(output).With().Timestamp().Logger()
}
