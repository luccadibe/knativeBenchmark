package main

import (
	"flag"
	"log/slog"
	"os"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/luccadibe/knativeBenchmark/pkg/config"
	"github.com/luccadibe/knativeBenchmark/pkg/connection"
	"github.com/luccadibe/knativeBenchmark/pkg/generator"
	"github.com/luccadibe/knativeBenchmark/pkg/store"
)

func main() {

	configPath := flag.String("config", "config.yaml", "path to config file")
	rps := flag.Float64("rps", 0, "replace config rps with another value")
	pingEndpoints := flag.Bool("ping", false, "ping endpoints")
	devMode := flag.Bool("dev", false, "development mode - use localhost:8080")
	cloudEventMode := flag.Bool("event", false, "cloud event mode - generate cloud events")
	coldStartMode := flag.Bool("cold-start", false, "cold start mode - send requests to trigger cold start")
	prefix := flag.String("prefix", "workload-generator", "prefix for log file")
	flag.Parse()
	logFile := store.GetLogFileWriter(*prefix, "/logs")
	defer logFile.Close()

	logger := slog.New(slog.NewTextHandler(logFile, nil))
	logger.Info("Loading configuration", "configPath", *configPath, "devMode", *devMode)
	cfg, err := config.Load(*configPath, *devMode)
	if err != nil {
		logger.Error("Failed to load config", "error", err)
	}

	if *rps > 0 {
		cfg.Rate.RequestsPerSecond = *rps
		logger.Info("Overriding RPS", "rps", *rps)
	}
	logger.Info("Loaded configuration", "config", cfg)

	pool := connection.NewPool(cfg.BaseURL, cfg.Rate.MaxIdleConns, cfg.Rate.MaxIdleConnsPerHost, cfg.Rate.IdleConnTimeout, cfg.Rate.Timeout)

	if *pingEndpoints {
		ping(cfg, logger, pool)
	}

	if *cloudEventMode {
		// get K_SINK from env
		logger.Info("K_SINK", "K_SINK", os.Getenv("K_SINK"))

		kSink := os.Getenv("K_SINK")
		if kSink == "" {
			logger.Error("K_SINK is not set")
			os.Exit(1)
		}
		cfg.Targets[0].URL = kSink

		event := cloudevents.NewEvent()
		event.SetID(cfg.Targets[0].Headers["ce-id"])
		event.SetSource(cfg.Targets[0].Headers["ce-source"])
		event.SetType(cfg.Targets[0].Headers["ce-type"])
		event.SetDataContentType(cfg.Targets[0].Headers["Content-Type"])
		event.SetData(cfg.Targets[0].Headers["Content-Type"], cfg.Targets[0].Body)
		logger.Info("Event", "event", event)
		gen := generator.NewCloudEventGenerator(cfg, &event, pool, logger)

		logger.Info("Generator initialized")
		err = gen.Start()
		if err != nil {
			logger.Error("Generator failed", "error", err)
		}
		gen.Stop()
	} else if *coldStartMode {
		gen := generator.New(cfg, logger, pool)
		logger.Info("Generator initialized")
		err = gen.StartColdStart()
		if err != nil {
			logger.Error("Cold start failed", "error", err)
		}
		gen.Stop()
	} else {
		gen := generator.New(cfg, logger, pool)
		logger.Info("Generator initialized")

		err = gen.Start()
		if err != nil {
			logger.Error("Generator failed", "error", err)
		}
		gen.Stop()
	}
}

func ping(cfg *config.Config, logger *slog.Logger, pool connection.Pool) {
	for _, target := range cfg.Targets {
		resp, err := pool.Get(target)
		if err != nil {
			logger.Error("Failed to ping endpoint", "target", target.URL, "error", err)
			panic(err)
		}
		defer resp.Response.Body.Close()
		logger.Info("Pinged endpoint", "target", target.URL, "status", resp.Response.StatusCode, "TTFB", resp.TTFB, "Total", resp.Total, "DNS", resp.DNSTime, "Connect", resp.ConnectTime, "TLS", resp.TLSTime)
	}
}
