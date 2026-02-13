package tracing

import (
	"context"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	tcr "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var once sync.Once
var initialized bool = false
var tp *trace.TracerProvider

const (
	endpointEnv   = "OTEL_EXPORTER_OTLP_ENDPOINT"
	samplerArgEnv = "OTEL_TRACES_SAMPLER_ARG"
)

func Init() {
	if initialized {
		log.Warn().Msgf("Tracing already initialized!")
		return
	}
	once.Do(
		func() {
			ctx := context.Background()

			serviceName := viper.GetString("APP_NAME")
			if serviceName == "" {
				log.Fatal().Msg("APP_NAME cannot be empty!!!")
			}

			collectorURL := os.Getenv(endpointEnv)
			if collectorURL == "" {
				log.Fatal().Msg(endpointEnv + " env is not set!!!")
			}

			exporter, err := otlptrace.New(ctx,
				otlptracegrpc.NewClient(
					otlptracegrpc.WithInsecure(),
					otlptracegrpc.WithEndpoint(collectorURL),
				),
			)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to create OTLP trace exporter")
			}

			resources, err := resource.New(ctx,
				resource.WithAttributes(
					attribute.String("service.name", serviceName),
					attribute.String("telemetry.sdk.language", "go"),
					attribute.String("telemetry.sdk.version", "v1.34.0"),
				),
			)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to create OTLP resource")
			}

			viper.SetDefault(samplerArgEnv, 0.1)
			samplingRatio := viper.GetFloat64(samplerArgEnv)

			tp = trace.NewTracerProvider(
				trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(samplingRatio))),
				trace.WithBatcher(exporter),
				trace.WithResource(resources),
			)

			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
			log.Info().
				Str("collectorURL", collectorURL).
				Str("serviceName", serviceName).
				Float64("samplingRatio", samplingRatio).
				Msg("Tracer initialized!")
			initialized = true
		},
	)
}

// GetTracer returns a tracer instance. If the tracer provider is not initialized, it returns a noop tracer.
// Pass the name of the package from where the tracer is being used.
// Example: GetTracer("github.com/my-org/my-repo/my-package")
func GetTracer(name string) tcr.Tracer {
	if tp == nil {
		return noop.NewTracerProvider().Tracer(name)
	}
	return tp.Tracer(name)
}

func ShutdownTracer() {
	log.Info().Msg("Tracer shuting down...")
	if tp == nil {
		log.Warn().Msg("Tracer provider is nil, nothing to shutdown")
		return
	}
	if err := tp.Shutdown(context.Background()); err != nil {
		log.Warn().Err(err)
		return
	}
	log.Info().Msg("Tracer shutdown complete!!!")
}
