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
	logger := e.LoggerPlugin.Logger.NewScope("OpenTelemetryPlugin")
	endpointUrl := e.ConfigProvider.Manager.GetString("otl.exporter.jaeger.endpoint")
	serviceName := e.ConfigProvider.Manager.GetString("otl.name")
	logger.WithFields(map[string]interface{}{
		"exporter.jaeger.endpoint": endpointUrl,
		"serviceName":              serviceName,
	}).Info("otl config")
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

func (o *OpenTelemetryPlugin) GetPluginConfig() map[string]interface{} {
	return map[string]interface{}{
		"enabled":         true,
		"exporter.jaeger": "configured",
	}
}
