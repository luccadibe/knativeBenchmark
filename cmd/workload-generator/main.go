package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/luccadibe/knativeBenchmark/pkg/config"
	"github.com/luccadibe/knativeBenchmark/pkg/connection"
	"github.com/luccadibe/knativeBenchmark/pkg/generator"
	"github.com/luccadibe/knativeBenchmark/pkg/store"
)

func main() {

	configPath := flag.String("config", "config.yaml", "path to config file")
	pingEndpoints := flag.Bool("ping", false, "ping endpoints")
	devMode := flag.Bool("dev", false, "development mode - use localhost:8080")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("Loading configuration", "configPath", *configPath, "devMode", *devMode)
	cfg, err := config.Load(*configPath, *devMode)
	if err != nil {
		logger.Error("Failed to load config", "error", err)
	}
	logger.Info("Configuration loaded", "baseURL", cfg.BaseURL, "targets", len(cfg.Targets))

	logFile := store.GetLogFileWriter(cfg.Store.LogDirPath)
	defer logFile.Close()

	logger = slog.New(slog.NewTextHandler(logFile, nil))
	logger.Info("Loaded configuration", "config", cfg)
	pool := connection.NewPool(cfg.BaseURL, cfg.Rate.MaxIdleConns, cfg.Rate.MaxIdleConnsPerHost, cfg.Rate.IdleConnTimeout, cfg.Rate.Timeout)

	if *pingEndpoints {
		ping(cfg, logger, pool)
	}

	gen := generator.New(cfg, logger, pool)
	logger.Info("Generator initialized")

	err = gen.Start()
	if err != nil {
		logger.Error("Generator failed", "error", err)
	}

}

func ping(cfg *config.Config, logger *slog.Logger, pool connection.Pool) {
	for _, target := range cfg.Targets {
		resp, err := pool.Get(target)
		if err != nil {
			logger.Error("Failed to ping endpoint", "target", target.URL, "error", err)
			panic(err)
		}
		defer resp.Body.Close()
		logger.Info("Pinged endpoint", "target", target.URL, "status", resp.StatusCode)
	}
}
