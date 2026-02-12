package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Shared RCON pool
	rconPool := NewRCONPool(cfg.RCON.Host, cfg.RCON.Port, cfg.RCON.Password)
	defer rconPool.Close()

	// K8s client
	k8s := NewK8sClient(cfg.Factorio.Namespace)

	// OTel metric exporter
	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure())
	if err != nil {
		log.Fatalf("metric exporter: %v", err)
	}
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(cfg.Metrics.Interval))),
	)
	defer meterProvider.Shutdown(ctx)

	// OTel log exporter
	logExporter, err := otlploggrpc.New(ctx, otlploggrpc.WithInsecure())
	if err != nil {
		log.Fatalf("log exporter: %v", err)
	}
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
	)
	defer loggerProvider.Shutdown(ctx)
	logger := loggerProvider.Logger(cfg.OTel.ServiceName)

	// Load Lua scripts
	collectLua := mustReadFile("/lua/collect.lua")
	registerEventsLua := mustReadFile("/lua/register_events.lua")
	pollEventsLua := mustReadFile("/lua/poll_events.lua")

	// Components
	var wg sync.WaitGroup

	// 1. Metrics collector
	if cfg.Metrics.Enabled {
		collector, err := NewCollector(rconPool, collectLua, meterProvider)
		if err != nil {
			log.Fatalf("collector: %v", err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			collector.Run(ctx, cfg.Metrics.Interval)
		}()
	}

	// 2. Log tailer + subscribers
	tailer := NewLogTailer(cfg.Factorio.PodLabel, k8s)

	otelSub := &OTelLogSubscriber{logger: logger, cfg: &cfg}
	tailer.Subscribe(otelSub)

	// 3. Discord channel (optional)
	var channels []Channel
	if cfg.Discord.Enabled {
		dc, err := NewDiscordChannel(cfg.Discord.BotToken, cfg.Discord.ChannelID, &cfg)
		if err != nil {
			log.Fatalf("discord: %v", err)
		}
		channels = append(channels, dc)
	}

	// 4. Bridge
	bridge := NewBridge(rconPool, channels)
	bridgeSub := &BridgeSubscriber{events: bridge.Events()}
	tailer.Subscribe(bridgeSub)

	// 5. Event poller
	if cfg.Events.Enabled {
		poller := NewEventPoller(rconPool, registerEventsLua, pollEventsLua, cfg.Events.PollInterval)
		poller.Subscribe(otelSub)
		poller.Subscribe(bridgeSub)
		wg.Add(1)
		go func() {
			defer wg.Done()
			poller.Run(ctx)
		}()
	}

	// Start goroutines
	wg.Add(1)
	go func() {
		defer wg.Done()
		tailer.Run(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		bridge.FanOutEvents(ctx)
	}()

	for _, ch := range channels {
		wg.Add(1)
		go func(c Channel) {
			defer wg.Done()
			if err := c.Start(ctx); err != nil {
				log.Printf("channel %s: %v", c.Name(), err)
			}
		}(ch)

		wg.Add(1)
		go func(c Channel) {
			defer wg.Done()
			bridge.HandleInbound(ctx, c)
		}(ch)
	}

	channelNames := make([]string, len(channels))
	for i, ch := range channels {
		channelNames[i] = ch.Name()
	}
	log.Printf("factorio-exporter started (metrics=%v, events=%v, channels=%v)",
		cfg.Metrics.Enabled, cfg.Events.Enabled, channelNames)

	wg.Wait()
	log.Println("shutting down")
}

func mustReadFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
