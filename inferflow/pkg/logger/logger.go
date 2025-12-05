package logger

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var applicationName string = ""

const logTemplate string = "%s %v [, ] [] %s inferflow %s\n"

func InitLogger(configs *configs.AppConfigs) {
	logLevel := strings.ToUpper(configs.Configs.ApplicationLogLevel)
	applicationName = configs.Configs.ApplicationName
	switch logLevel {
	case "DEBUG":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "INFO":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "WARN":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "ERROR":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "FATAL":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "PANIC":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	case "DISABLED":
		zerolog.SetGlobalLevel(zerolog.Disabled)
	default:
		Panic(fmt.Sprintf("Incorrect log level %s", logLevel), nil)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	Info("Logger initialized!")
}

func Info(message string) {
	log.Info().Msgf(logTemplate, applicationName, time.Now().Format("02-01-2006 15:04:05.000 -0700"), "INFO", message)
}

func Error(message string, err error) {
	log.Error().AnErr("Error ", err).Msgf(logTemplate, applicationName, time.Now().Format("02-01-2006 15:04:05.000 -0700"), "ERROR", message)
}

func PercentError(message string, err error, loggingPercent int) {
	if loggingPercent == 0 {
		loggingPercent = 10
	}
	randomNumber := rand.Intn(100) + 1
	if randomNumber <= loggingPercent {
		log.Error().AnErr("Error ", err).Msgf(logTemplate, applicationName, time.Now().Format("02-01-2006 15:04:05.000 -0700"), "ERROR", message)
	}
}

func Panic(message string, err error) {
	Error(message, err)
	log.Panic().AnErr("Error", err).Msgf(logTemplate, applicationName, time.Now().Format("02-01-2006 15:04:05.000 -0700"), "PANIC", message)
}
