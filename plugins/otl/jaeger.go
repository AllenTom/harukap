package otl

import (
	"github.com/allentom/harukap"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func newDefaultJaegerExporter(e *harukap.HarukaAppEngine) (*trace.TracerProvider, error) {
	endpointUrl := e.ConfigProvider.Manager.GetString("otl.exporter.jaeger.endpoint")
	serviceName := e.ConfigProvider.Manager.GetString("otl.name")
	exp, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(endpointUrl),
		),
	)
	if err != nil {
		return nil, err
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)
	return tp, nil
}
