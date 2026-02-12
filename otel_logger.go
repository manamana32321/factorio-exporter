package main

import (
	"context"
	"time"

	otellog "go.opentelemetry.io/otel/log"
)

// OTelLogSubscriber sends GameEvents as structured OTel log records (→ Loki).
type OTelLogSubscriber struct {
	logger otellog.Logger
	cfg    *Config
}

func (s *OTelLogSubscriber) OnLogEvent(event GameEvent) {
	if !s.cfg.lokiEventAllowed(event.Type) {
		return
	}

	var attrs []otellog.KeyValue
	if event.Player != "" {
		attrs = append(attrs, otellog.String("player", event.Player))
	}
	if event.Message != "" {
		attrs = append(attrs, otellog.String("message", event.Message))
	}
	for k, v := range event.Extra {
		attrs = append(attrs, otellog.String(k, v))
	}

	logEvent(s.logger, event.Type, attrs...)
}

func logEvent(logger otellog.Logger, event string, attrs ...otellog.KeyValue) {
	var r otellog.Record
	r.SetTimestamp(time.Now())
	r.SetBody(otellog.StringValue(event))
	r.AddAttributes(attrs...)
	logger.Emit(context.Background(), r)
}
