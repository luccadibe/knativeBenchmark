package connection

import (
	"log"
	"net/http"
	"time"

	"workload-generator/internal/config"
)

type Pool struct {
	client  *http.Client
	baseURL string
}

func NewPool(baseURL string) *Pool {
	return &Pool{
		baseURL: baseURL,
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
			Timeout: 30 * time.Second,
		},
	}
}

func (p *Pool) Get(target config.Target) (*http.Response, error) {
	req, err := http.NewRequest("GET", target.URL, nil)
	if err != nil {
		return nil, err
	}

	// Set Host header if specified
	if target.HostHeader != "" {
		req.Host = target.HostHeader
		log.Printf("Setting Host header to: %s", target.HostHeader)
	}

	// Add any additional headers
	for k, v := range target.Headers {
		req.Header.Set(k, v)
		log.Printf("Added header %s: %s", k, v)
	}

	return p.client.Do(req)
}
