package logger

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/trace"
)

var (
	once        sync.Once
	initialized = false
	appName     = ""
	signalChan  = make(chan os.Signal, 1)
)

// InitLogger initializes the logger with the given app name and log level
// This can be used when you want to initialize the logger with config file
func InitLogger(appName, logLevel string) {
	if len(appName) == 0 {
		panic("Application name is not set!")
	}
	if len(logLevel) == 0 {
		log.Warn().Msg("Log level not set, defaulting to WARN")
		logLevel = "WARN"
	}

	initLogger(appName, logLevel)

}

// Init initializes the logger by fetching the log level and app name from the viper configuration
func Init() {

	appName = viper.GetString("APP_NAME")
	logLevel := viper.GetString("APP_LOG_LEVEL")

	//check
	if len(appName) == 0 {
		panic("APP_NAME is not set!")

	}
	if len(logLevel) == 0 {
		panic("APP_LOG_LEVEL is not set!")
	}
	InitLogger(appName, logLevel)
}

func initLogger(appName, logLevel string) {
	rbSize := -1
	drainingInterval := 5 * time.Millisecond
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	if viper.IsSet("LOG_RB_SIZE") {
		rbSize = viper.GetInt("LOG_RB_SIZE")
		drainingInterval = viper.GetDuration("LOG_RB_DRAINING_INTERVAL")
	}

	if initialized {
		log.Debug().Msgf("Logger already initialized!")
		return
	}
	once.Do(func() {

		setLogLevel(logLevel)
		var dropWarnOnce sync.Once

		// target_log_pattern = %d{yyyy-MM-dd HH:mm:ss.SSS} - [%-5level] - [%logger{36}:%M:%L] - [%pid, %thread] - (,)|(%X{trace_id},%X{span_id}) - [%X{MEESHO-USER-ID}] - () - %msg%n%throwable
		// eg = 2024-08-01 14:38:51.316 - [ERROR] - [com.meesho.edgeproxy.filters.outbound.impl.Status4xxWhiteListingFilter::] - [1, reactor-http-epoll-1] - (6aadf4e41ac5b14c,6aadf4e41ac5b14c)|(e67f0226042b4c38088be7ebd2da4c0a,3288c6dea9bb3cc5) - [] - () -  401 UNAUTHORIZED from downstream path /api/1.0/config
		log.Logger = log.With().
			Caller().
			Str("processInfo", fmt.Sprintf("- [%d, ] -", os.Getpid())).
			Logger()

		var w io.Writer
		var closer io.Closer
		if rbSize > 0 {
			metric.Incr("log_rb_initialized", []string{})
			log.Info().Msgf("Initializing logger with ring buffer size: %d", rbSize)
			dw := diode.NewWriter(os.Stdout, rbSize, drainingInterval, func(missed int) {
				metric.Count("log_rb_dropped", int64(missed), []string{})
				dropWarnOnce.Do(func() {
					fmt.Fprintf(os.Stderr, "Error from Logger: dropping logs due to buffer overflow\n")
				})
			})
			w = dw
			closer = dw
			go func() {
				<-signalChan
				fmt.Fprintf(os.Stdout, "Received signal, closing logger\n")
				_ = closer.Close()
			}()
		} else {
			w = os.Stdout
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:           w,
			NoColor:       true,
			TimeFormat:    "2006-01-02 15:04:05.000",
			FormatLevel:   func(i interface{}) string { return strings.ToUpper(fmt.Sprintf("- [%-5s] -", i)) },
			FormatCaller:  func(i interface{}) string { return fmt.Sprintf("%s", i) },
			FormatMessage: func(i interface{}) string { return fmt.Sprintf("%s", i) },
			FieldsExclude: []string{
				"processInfo",
				"traceInfo",
				"extraInfo",
			},
			PartsOrder: []string{
				zerolog.TimestampFieldName,
				zerolog.LevelFieldName,
				zerolog.CallerFieldName,
				"processInfo",
				"traceInfo",
				"extraInfo",
				zerolog.MessageFieldName,
			},
		})

		// enable logging caller
		log.Logger = log.With().Caller().Logger()

		// As a standard practice, we are logging [file_name:method_name:line_number], as method name is not available
		// we are logging [file_name::line_number]
		zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
			parts := strings.Split(file, "/")
			if len(parts) == 1 {
				return fmt.Sprintf("[%s::%d]", parts[0], line)
			}
			return fmt.Sprintf("[%s::%d]", parts[len(parts)-1], line)
		}

		// Adding hooks, to add trace info and custom info
		traceHook := zerolog.Hook(TraceHook{})
		customHook := zerolog.Hook(CustomHook{})
		log.Logger = log.Logger.Hook(traceHook, customHook)

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

type TraceHook struct{}

type CustomHook struct{}

func (h TraceHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	ctx := e.GetCtx()
	span := trace.SpanFromContext(ctx)
	spanContext := span.SpanContext()
	traceId := ""
	spanId := ""
	if spanContext.HasTraceID() {
		traceId = spanContext.TraceID().String()
	}
	if spanContext.HasSpanID() {
		spanId = spanContext.SpanID().String()
	}
	e.Str("traceInfo", fmt.Sprintf("(,)|(%s,%s)", traceId, spanId))
}

// We can use the custom hook to add any extra info to the log
func (h CustomHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	// contextId := e.GetCtx().Value("context_id")
	contextId := ""

	extraInfoMap := make(map[string]string)
	// extraInfoMap["tenant"] := e.GetCtx().Value("tenant")

	e.Str("extraInfo", fmt.Sprintf("- [%s] - (%s) -", contextId, mapToString(extraInfoMap)))
}

func mapToString(fields map[string]string) string {
	var builder strings.Builder
	for key, value := range fields {
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(value)
		builder.WriteString(",")
	}
	// Remove the trailing comma
	result := builder.String()
	if len(result) > 0 {
		result = result[:len(result)-1]
	}
	return result
}
