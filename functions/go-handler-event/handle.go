package function

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/cloudevents/sdk-go/v2/event"
)

// Handle an event.
func Handle(ctx context.Context, e event.Event) (*event.Event, error) {

	log.Printf("Received event: ID=%s, Type=%s, Source=%s", e.ID(), e.Type(), e.Source())
	log.Printf("Content Type: %s", e.DataContentType())

	rawData := string(e.Data())
	log.Printf("Raw data (quoted): %q", rawData)      // Will show exact string content
	log.Printf("Raw data bytes: %v", []byte(rawData)) // Will show actual bytes
	log.Printf("Raw data length: %d", len(rawData))   // Will show length

	counter, err := strconv.Atoi(rawData)
	if err != nil {
		log.Printf("Error converting data to integer: %v", err)
		return nil, fmt.Errorf("failed to convert data to integer: %v, raw data: %q, bytes: %v",
			err, rawData, []byte(rawData))
	}

	// Increment counter
	counter++
	// Create response event
	responseEvent := e.Clone()
	if err := responseEvent.SetData(e.DataContentType(), strconv.Itoa(counter)); err != nil {
		return nil, fmt.Errorf("failed to set response data: %v", err)
	}

	return &responseEvent, nil
}

/*
Other supported function signatures:

	Handle()
	Handle() error
	Handle(context.Context)
	Handle(context.Context) error
	Handle(event.Event)
	Handle(event.Event) error
	Handle(context.Context, event.Event)
	Handle(context.Context, event.Event) error
	Handle(event.Event) *event.Event
	Handle(event.Event) (*event.Event, error)
	Handle(context.Context, event.Event) *event.Event
	Handle(context.Context, event.Event) (*event.Event, error)

*/
