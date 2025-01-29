package main

import (
	"encoding/csv"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

type EventLog struct {
	ID        string    `json:"event_id"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	// Open CSV file for writing
	f, err := os.OpenFile("/data/events.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("Failed to open CSV file", "error", err)
		os.Exit(1)
	}
	defer f.Close()

	mu := sync.Mutex{}

	// Create CSV writer
	writer := csv.NewWriter(f)
	defer writer.Flush()

	// Write header if file is empty
	fi, err := f.Stat()
	if err != nil {
		slog.Error("Failed to stat file", "error", err)
		os.Exit(1)
	}

	if fi.Size() == 0 {
		err = writer.Write([]string{"event_id", "timestamp"})
		if err != nil {
			slog.Error("Failed to write CSV header", "error", err)
			os.Exit(1)
		}
	}

	http.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var event EventLog
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mu.Lock()
		err := writer.Write([]string{
			event.ID,
			// Format with milisecond precision
			event.Timestamp.Format(time.RFC3339Nano),
		})
		if err != nil {
			slog.Error("Failed to write CSV row", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Flush()
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	slog.Info("Starting event logger service on :8080")
	http.ListenAndServe(":8080", nil)
}
