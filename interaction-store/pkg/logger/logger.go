package logger

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
)

var (
	once        sync.Once
	initialized = false
	appName     = ""
)

// Init initializes the logger by fetching the log level and app name from the viper configuration
func Init() {

	appName = viper.GetString("APP_NAME")
	logLevel := viper.GetString("APP_LOG_LEVEL")

	if len(appName) == 0 {
		panic("APP_NAME is not set!")
	}
	if len(logLevel) == 0 {
		log.Warn().Msg("Log level not set, defaulting to WARN")
		logLevel = "WARN"
	}
	initLogger(appName, logLevel)
}

func initLogger(appName, logLevel string) {
	if initialized {
		log.Debug().Msgf("Logger already initialized!")
		return
	}
	once.Do(func() {
		setLogLevel(logLevel)
		log.Logger = log.With().Caller().Str("applicationName", appName).Logger()
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "02-01-2006 15:04:05.000",
			FormatLevel: func(i interface{}) string {
				return strings.ToUpper(fmt.Sprintf("%-6s", i))
			},
			FormatMessage: func(i interface{}) string {
				return fmt.Sprintf("%s", i)
			},
			FieldsExclude: []string{
				"applicationName",
			},
			PartsOrder: []string{
				"applicationName",
				zerolog.TimestampFieldName,
				zerolog.LevelFieldName,
				zerolog.CallerFieldName,
				zerolog.MessageFieldName,
			},
		})

		// enable logging caller
		log.Logger = log.With().Caller().Logger()

		// customise caller
		zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
			lineNum := strconv.Itoa(line)
			parts := strings.Split(file, "/")
			if len(parts) == 1 {
				return parts[0] + ":" + lineNum
			}
			return parts[len(parts)-1] + ":" + lineNum
		}

		// add custom hook
		hook := zerolog.Hook(CustomHook{})
		log.Logger = log.Logger.Hook(hook)

		// add stack trace to error
		zerolog.ErrorStackMarshaler = func(err error) interface{} {
			return fmt.Sprintf("%s\n%s", err, debug.Stack())
		}

		initialized = true
		log.Info().Msg("Logger initialized!")
	})
}

// Sets the log level
func setLogLevel(logLevel string) {
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
		log.Panic().Msgf("Incorrect log level - %s", logLevel)
	}
}

type CustomHook struct{}

func (h CustomHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// we can add a custom hook here
	// e.Str("app_name", appName)
}
