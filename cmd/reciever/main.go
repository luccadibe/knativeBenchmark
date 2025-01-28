package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
)

type EventLog struct {
	ID        string    `json:"event_id"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	p, err := cehttp.New()
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

	err = c.StartReceiver(context.Background(), func(ctx context.Context, event cloudevents.Event) error {
		// Send event ID to logger service
		eventLog := EventLog{ID: event.ID()}
		// time.Now() has nanosecond precision (1e-9 seconds)
		eventLog.Timestamp = time.Now()
		jsonData, err := json.Marshal(eventLog)
		if err != nil {
			slog.Error("Failed to marshal event", "error", err)
			return err
		}

		resp, err := http.Post("http://event-logger.functions.svc.cluster.local/log",
			"application/json",
			bytes.NewBuffer(jsonData))
		if err != nil {
			slog.Error("Failed to log event", "error", err)
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			slog.Error("Failed to log event", "status", resp.StatusCode)
			return err
		}

		slog.Info("Logged event", "event_id", event.ID())
		return nil
	})

	if err != nil {
		slog.Error("Failed to start receiver", "error", err)
		os.Exit(1)
	}
}
