package main

import (
	"context"
	"log/slog"
	"os"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/protocol/http"
)

func main() {
	// Set up logging to file

	var handler slog.Handler

	switch os.Getenv("LOG_TO") {
	case "file":
		logFile, err := os.OpenFile("/logs/receiver.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			slog.Error("Failed to open log file", "error", err)
			os.Exit(1)
		}
		defer logFile.Close()
		handler = slog.NewJSONHandler(logFile, nil)
	case "stdout":
		handler = slog.NewJSONHandler(os.Stdout, nil)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Create CloudEvents client
	p, err := http.New()
	if err != nil {
		slog.Error("Failed to create protocol", "error", err)
		os.Exit(1)
	}

	c, err := cloudevents.NewClient(p)
	if err != nil {
		slog.Error("Failed to create client", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting receiver...")

	// Start receiving events
	err = c.StartReceiver(context.Background(), func(ctx context.Context, event cloudevents.Event) error {
		logger.Info("Received event",
			"event_id", event.ID())
		return nil
	})

	if err != nil {
		slog.Error("Failed to start receiver", "error", err)
		os.Exit(1)
	}
}
