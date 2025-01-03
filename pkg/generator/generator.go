package generator

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"golang.org/x/time/rate"

	"github.com/luccadibe/knativeBenchmark/pkg/config"
	"github.com/luccadibe/knativeBenchmark/pkg/connection"
)

type Generator interface {
	Start() error
	Stop()
	GetPool() connection.Pool
}

type generator struct {
	cfg     *config.Config
	Pool    connection.Pool
	limiter *rate.Limiter
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	logger  *slog.Logger
}

func New(cfg *config.Config, logger *slog.Logger, pool connection.Pool) Generator {
	ctx, cancel := context.WithCancel(context.Background())
	return &generator{
		cfg:     cfg,
		Pool:    pool,
		limiter: rate.NewLimiter(rate.Limit(cfg.Rate.RequestsPerSecond), 1),
		ctx:     ctx,
		cancel:  cancel,
		logger:  logger,
	}
}

func (g *generator) Start() error {
	err := g.run()
	g.wg.Wait()
	return err
}

func (g *generator) Stop() {
	g.cancel()
	g.wg.Wait()
}

func (g *generator) GetPool() connection.Pool {
	return g.Pool
}

func (g *generator) run() error {
	g.logger.Info("Starting workload generation", "rate", g.cfg.Rate.RequestsPerSecond)
	if g.cfg.Rate.RequestsPerSecond == 0 {
		g.logger.Info("Rate 0 detected, running for maximum throughput")
		return g.runMaxThroughput()
	}

	interval := time.Second / time.Duration(g.cfg.Rate.RequestsPerSecond)
	g.logger.Info("Calculated ticker interval", "interval", interval)

	ticker := time.NewTicker(interval)
	startTime := time.Now()
	requestCount := 0

	for {
		select {
		case <-g.ctx.Done():
			g.logger.Info("Generator stopped", "totalRequests", requestCount)
			return nil
		case <-ticker.C:
			//g.logger.Info("Tick", "time", time.Now())
			if err := g.limiter.Wait(g.ctx); err != nil {
				g.logger.Error("Rate limiter error", "error", err)
				return err
			}

			for _, target := range g.cfg.Targets {
				g.wg.Add(1)
				go func(target *config.Target) {
					defer g.wg.Done()
					start := time.Now()
					resp, err := g.Pool.Get(target)
					if err != nil {
						g.logger.Error("Request error", "target", target.URL, "error", err)
						return
					}
					defer resp.Body.Close()

					duration := time.Since(start)
					g.logger.Info("Success", "target", target.URL, "latency", duration, "status", resp.StatusCode)
					requestCount++
				}(target)
			}

			if g.cfg.Rate.Duration.Duration > 0 && time.Since(startTime) > g.cfg.Rate.Duration.Duration {
				g.logger.Info("Duration reached", "duration", g.cfg.Rate.Duration.Duration)
				g.Stop()
				return nil
			}
		}
	}
}

func (g *generator) runMaxThroughput() error {
	startTime := time.Now()
	requestCount := 0

	for {
		select {
		case <-g.ctx.Done():
			g.logger.Info("Generator stopped", "totalRequests", requestCount)
			return nil
		default:
			for _, target := range g.cfg.Targets {
				g.wg.Add(1)
				go func(target *config.Target) {
					defer g.wg.Done()
					start := time.Now()
					resp, err := g.Pool.Get(target)
					if err != nil {
						g.logger.Error("Request error", "target", target.URL, "error", err)
						requestCount++
						return
					}
					if resp.StatusCode != http.StatusOK {
						g.logger.Error("Request failed", "target", target.URL, "status", resp.StatusCode)
						body, err := io.ReadAll(resp.Body)
						if err != nil {
							g.logger.Error("Failed to read response body", "target", target.URL, "error", err)
						}
						g.logger.Error("Non-200 response",
							"target", target.URL,
							"status", resp.StatusCode,
							"headers", resp.Header,
							"body", string(body))
						requestCount++
						return
					}
					defer resp.Body.Close()

					duration := time.Since(start)
					g.logger.Info("Success", "target", target.URL, "latency", duration, "status", resp.StatusCode)
					requestCount++
				}(target)
			}

			if g.cfg.Rate.Duration.Duration > 0 && time.Since(startTime) > g.cfg.Rate.Duration.Duration {
				g.logger.Info("Duration reached", "duration", g.cfg.Rate.Duration.Duration)
				g.Stop()
				return nil
			}
		}
	}
}

var _ Generator = &cloudEventGenerator{}

type cloudEventGenerator struct {
	cfg     *config.Config
	event   *cloudevents.Event
	Pool    connection.Pool
	limiter *rate.Limiter
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	logger  *slog.Logger
}

// GetPool implements Generator.
func (c *cloudEventGenerator) GetPool() connection.Pool {
	return c.Pool
}

// Start implements Generator.
func (c *cloudEventGenerator) Start() error {
	err := c.run()
	c.wg.Wait()
	return err
}

// Stop implements Generator.
func (c *cloudEventGenerator) Stop() {
	c.cancel()
	c.wg.Wait()
}

func NewCloudEventGenerator(cfg *config.Config, event *cloudevents.Event, pool connection.Pool, logger *slog.Logger) Generator {
	return &cloudEventGenerator{
		cfg:    cfg,
		event:  event,
		Pool:   pool,
		logger: logger,
	}
}

func (c *cloudEventGenerator) run() error {

	c.logger.Info("Starting workload generation", "rate", c.cfg.Rate.RequestsPerSecond)
	c.logger.Info("Using cloudevent", "event", c.event)

	interval := time.Second / time.Duration(c.cfg.Rate.RequestsPerSecond)
	c.logger.Info("Calculated ticker interval", "interval", interval)

	ticker := time.NewTicker(interval)
	//startTime := time.Now()
	requestCount := 0

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("Generator stopped", "totalRequests", requestCount)
			return nil
		case <-ticker.C:
			c.wg.Add(1)
			go func() {
				defer c.wg.Done()
				//TODO
				c.Pool.Post(c.cfg.Targets[0], bytes.NewReader(c.event.Data()))
			}()
		}
	}
}
