package connection

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/luccadibe/knativeBenchmark/pkg/config"
)

type Pool interface {
	Get(target *config.Target) (*ResponseMetrics, error)
	Post(target *config.Target, body io.Reader) (*ResponseMetrics, error)
	GenerateCloudEvent(target *config.Target, event *cloudevents.Event) (*ResponseMetrics, error)
	Targets() map[*config.Target]int
}

type pool struct {
	client  *http.Client
	baseURL string
	targets map[*config.Target]int
}

func NewPool(baseURL string, maxIdleConns int, maxIdleConnsPerHost int, idleConnTimeout time.Duration, timeout time.Duration) Pool {
	return &pool{
		baseURL: baseURL,
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        maxIdleConns,
				MaxIdleConnsPerHost: maxIdleConnsPerHost,
				IdleConnTimeout:     idleConnTimeout,
			},
			Timeout: timeout,
		},
	}
}

type ResponseMetrics struct {
	Response    *http.Response
	DNSTime     time.Duration
	ConnectTime time.Duration
	TLSTime     time.Duration
	TTFB        time.Duration
	Total       time.Duration
}

func (p *pool) Get(target *config.Target) (*ResponseMetrics, error) {
	req, err := http.NewRequest("GET", target.URL, nil)
	if err != nil {
		return nil, err
	}

	// Set Host header if specified
	if target.HostHeader != "" {
		req.Host = target.HostHeader
	}

	// Add any additional headers
	for k, v := range target.Headers {
		req.Header.Set(k, v)
	}

	return p.executeWithMetrics(req)
}

func (p *pool) Post(target *config.Target, body io.Reader) (*ResponseMetrics, error) {
	req, err := http.NewRequest("POST", target.URL, body)
	if err != nil {
		return nil, err
	}

	// Set Host header if specified
	if target.HostHeader != "" {
		req.Host = target.HostHeader
	}

	// Add any additional headers
	for k, v := range target.Headers {
		req.Header.Set(k, v)
	}

	// Set content type if not specified
	if _, exists := target.Headers["Content-Type"]; !exists {
		req.Header.Set("Content-Type", "application/json")
	}

	return p.executeWithMetrics(req)
}

func (p *pool) GenerateCloudEvent(target *config.Target, event *cloudevents.Event) (*ResponseMetrics, error) {
	req, err := cehttp.NewHTTPRequestFromEvent(context.Background(), target.URL, *event)
	if err != nil {
		return nil, err
	}
	return p.executeWithMetrics(req)
}

func (p *pool) Targets() map[*config.Target]int {
	return p.targets
}

type poolMock struct {
	targets map[*config.Target]int
	mu      sync.Mutex
}

// GenerateCloudEvent implements Pool.
func (p *poolMock) GenerateCloudEvent(target *config.Target, event *cloudevents.Event) (*ResponseMetrics, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.targets[target]++
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		// This is because we need to call Close() on the body in the generator
		Body: io.NopCloser(strings.NewReader("")),
	}
	return &ResponseMetrics{
		Response: mockResponse,
	}, nil
}

func (p *poolMock) Get(target *config.Target) (*ResponseMetrics, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.targets[target]++
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		// This is because we need to call Close() on the body in the generator
		Body: io.NopCloser(strings.NewReader("")),
	}
	return &ResponseMetrics{
		Response: mockResponse,
	}, nil
}

func (p *poolMock) Post(target *config.Target, body io.Reader) (*ResponseMetrics, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.targets[target]++
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		// This is because we need to call Close() on the body in the generator
		Body: io.NopCloser(strings.NewReader("")),
	}
	return &ResponseMetrics{
		Response: mockResponse,
	}, nil
}

func (p *poolMock) Targets() map[*config.Target]int {
	return p.targets
}

func NewPoolMock(c *config.Config) Pool {
	targets := make(map[*config.Target]int)

	for _, target := range c.Targets {
		targets[target] = 0
	}

	return &poolMock{
		targets: targets,
	}
}

func (p *pool) executeWithMetrics(req *http.Request) (*ResponseMetrics, error) {
	var metrics ResponseMetrics
	var start = time.Now()
	var dnsStart, connectStart, tlsStart time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			metrics.DNSTime = time.Since(dnsStart)
		},

		ConnectStart: func(network, addr string) {
			connectStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			metrics.ConnectTime = time.Since(connectStart)
		},

		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			metrics.TLSTime = time.Since(tlsStart)
		},

		GotFirstResponseByte: func() {
			metrics.TTFB = time.Since(start)
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	metrics.Response = resp
	metrics.Total = time.Since(start)

	return &metrics, nil
}
