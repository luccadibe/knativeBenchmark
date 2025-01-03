package connection

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/luccadibe/knativeBenchmark/pkg/config"
)

type Pool interface {
	Get(target *config.Target) (*http.Response, error)
	Post(target *config.Target, body io.Reader) (*http.Response, error)
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

func (p *pool) Get(target *config.Target) (*http.Response, error) {
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

	return p.client.Do(req)
}

func (p *pool) Post(target *config.Target, body io.Reader) (*http.Response, error) {

	req, err := http.NewRequest("POST", target.URL, body)
	if err != nil {
		return nil, err
	}

	// This is important for cloudevent https://github.com/cloudevents/spec/blob/v1.0.2/cloudevents/bindings/http-protocol-binding.md
	// For now we implement only the binary mode.
	/* POST /someresource HTTP/1.1
	Host: webhook.example.com
	ce-specversion: 1.0
	ce-type: com.example.someevent
	ce-time: 2018-04-05T03:56:24Z
	ce-id: 1234-1234-1234
	ce-source: /mycontext/subcontext
	    .... further attributes ...
	Content-Type: application/json; charset=utf-8
	Content-Length: nnnn
	{
	    ... application data ...
	} */

	for k, v := range target.Headers {
		req.Header.Set(k, v)
	}

	return p.client.Do(req)
}

func (p *pool) Targets() map[*config.Target]int {
	return p.targets
}

type poolMock struct {
	targets map[*config.Target]int
	mu      sync.Mutex
}

func (p *poolMock) Get(target *config.Target) (*http.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.targets[target]++
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		// This is because we need to call Close() on the body in the generator
		Body: io.NopCloser(strings.NewReader("")),
	}
	return mockResponse, nil
}

func (p *poolMock) Post(target *config.Target, body io.Reader) (*http.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.targets[target]++
	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		// This is because we need to call Close() on the body in the generator
		Body: io.NopCloser(strings.NewReader("")),
	}
	return mockResponse, nil
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
