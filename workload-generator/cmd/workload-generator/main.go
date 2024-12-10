package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"workload-generator/internal/config"
	"workload-generator/internal/generator"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	configPath := flag.String("config", "config.yaml", "path to config file")
	devMode := flag.Bool("dev", false, "development mode - use localhost:8080")
	flag.Parse()

	log.Printf("Loading configuration from: %s", *configPath)
	cfg, err := config.Load(*configPath, *devMode)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Configuration loaded: baseURL=%s, targets=%d", cfg.BaseURL, len(cfg.Targets))

	gen := generator.New(cfg)
	log.Println("Generator initialized")

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the generator
	errChan := make(chan error, 1)
	go func() {
		log.Println("Starting generator...")
		errChan <- gen.Start()
	}()

	// Wait for shutdown signal or error
	select {
	case err := <-errChan:
		if err != nil {
			log.Fatalf("Generator failed: %v", err)
		} else {
			log.Println("Generator stopped")
		}
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
		gen.Stop()
	}
}
