package otl

import (
	"github.com/allentom/harukap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type OpenTelemetryPlugin struct {
	Provider trace.TracerProvider
}

func NewOpenTelemetryPlugin(Provider trace.TracerProvider) *OpenTelemetryPlugin {
	return &OpenTelemetryPlugin{
		Provider: Provider,
	}
}

func (o *OpenTelemetryPlugin) OnInit(e *harukap.HarukaAppEngine) error {
	if o.Provider == nil {
		exporter, err := newDefaultJaegerExporter(e)
		if err != nil {
			return err
		}
		otel.SetTracerProvider(exporter)
	} else {
		otel.SetTracerProvider(o.Provider)
	}
	return nil
}
