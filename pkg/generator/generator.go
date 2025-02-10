package generator

import (
	"context"
	"fmt"
	"hash/maphash"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"golang.org/x/time/rate"

	"github.com/luccadibe/knativeBenchmark/pkg/config"
	"github.com/luccadibe/knativeBenchmark/pkg/connection"
)

type Generator interface {
	Start() error
	StartColdStart() error
	Stop()
	GetPool() connection.Pool
}

type generator struct {
	cfg         *config.Config
	Pool        connection.Pool
	limiter     *rate.Limiter
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	logger      *slog.Logger
	currentRate float64
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

func (g *generator) StartColdStart() error {
	err := g.runColdStart()
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

	// If its less than 1, we are in cold start mode, so we dont need the rampup
	if g.cfg.Rate.RequestsPerSecond > 1 {
		if err := g.rampUp(); err != nil {
			return fmt.Errorf("ramp-up failed: %w", err)
		}
	}

	interval := time.Duration(float64(time.Second) / g.cfg.Rate.RequestsPerSecond)
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
					efficientLogger := g.logger.With("target", target.URL)
					defer g.wg.Done()
					metrics, err := g.Pool.Get(target)
					if err != nil {
						efficientLogger.Error("Request error", "error", err)
						return
					}
					// we expect a boolean response specifying if the function is cold
					body, err := io.ReadAll(metrics.Response.Body)
					if err != nil {
						efficientLogger.Error("Failed to read response body", "error", err)
						return
					}
					defer metrics.Response.Body.Close()
					efficientLogger.Info("Success", "TTFB", metrics.TTFB, "Total", metrics.Total, "isCold", string(body), "status", metrics.Response.StatusCode, "DNS", metrics.DNSTime, "Connect", metrics.ConnectTime, "TLS", metrics.TLSTime)
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

func (g *generator) rampUp() error {
	if g.cfg.Rate.RequestsPerSecond <= 1 {
		return nil
	}

	g.logger.Info("Starting 15-second ramp-up", "targetRate", g.cfg.Rate.RequestsPerSecond)

	steps := 15 // one step per second
	rateIncrement := (g.cfg.Rate.RequestsPerSecond - 1) / float64(steps)
	currentRate := 1.0

	for i := 0; i < steps; i++ {
		select {
		case <-g.ctx.Done():
			return nil
		default:
			currentRate += rateIncrement
			// Calculate appropriate burst size based on rate
			burstSize := int(currentRate)
			if burstSize < 1 {
				burstSize = 1
			}

			localRateLimiter := rate.NewLimiter(rate.Limit(currentRate), burstSize)
			g.currentRate = currentRate

			// Create a ticker for the current rate
			interval := time.Duration(float64(time.Second) / currentRate)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			stepTimer := time.NewTimer(time.Second)
			defer stepTimer.Stop()

		stepLoop:
			for {
				select {
				case <-g.ctx.Done():
					return nil
				case <-stepTimer.C:
					break stepLoop
				case <-ticker.C:
					if err := localRateLimiter.Wait(g.ctx); err != nil {
						return err
					}

					// Launch concurrent requests for each target
					for _, target := range g.cfg.Targets {
						g.wg.Add(1)
						go func(t *config.Target) {
							defer g.wg.Done()
							if err := g.sendRequest(t); err != nil {
								g.logger.Error("Request failed", "error", err)
							}
						}(target)
					}
				}
			}

			g.logger.Info("Ramp-up progress", "currentRate", currentRate)
		}
	}

	g.logger.Info("Ramp-up complete")
	return nil
}

func (g *generator) runColdStart() error {
	interval := time.Duration(float64(time.Second) / g.cfg.Rate.RequestsPerSecond)
	g.logger.Info("Calculated ticker interval", "interval", interval)

	ticker := time.NewTicker(interval)
	startTime := time.Now()

	for {
		select {
		case <-g.ctx.Done():
			g.logger.Info("Generator stopped")
			return nil
		case <-ticker.C:
			for _, target := range g.cfg.Targets {
				// Sequential requests, sleep between each
				if err := g.sendRequest(target); err != nil {
					g.logger.Error("Request failed", "error", err)
				}
				time.Sleep(time.Second * 7)
			}
			if g.cfg.Rate.Duration.Duration > 0 && time.Since(startTime) > g.cfg.Rate.Duration.Duration {
				g.logger.Info("Duration reached", "duration", g.cfg.Rate.Duration.Duration)
				g.Stop()
				return nil
			}
		}
	}
}

func (g *generator) sendRequest(target *config.Target) error {
	efficientLogger := g.logger.With("target", target.URL)
	metrics, err := g.Pool.Get(target)
	if err != nil {
		efficientLogger.Error("Request error", "error", err)
		return err
	}

	body, err := io.ReadAll(metrics.Response.Body)
	if err != nil {
		efficientLogger.Error("Failed to read response body", "error", err)
		return err
	}
	defer metrics.Response.Body.Close()

	efficientLogger.Info("Success",
		"TTFB", metrics.TTFB,
		"Total", metrics.Total,
		"isCold", string(body),
		"status", metrics.Response.StatusCode,
		"DNS", metrics.DNSTime,
		"Connect", metrics.ConnectTime,
		"TLS", metrics.TLSTime)

	return nil
}

func (g *generator) runMaxThroughput() error {
	startTime := time.Now()

	for {
		select {
		case <-g.ctx.Done():
			g.logger.Info("Generator stopped")
			return nil
		default:
			for _, target := range g.cfg.Targets {
				g.wg.Add(1)
				go func(target *config.Target) {
					efficientLogger := g.logger.With("target", target.URL)
					defer g.wg.Done()
					metrics, err := g.Pool.Get(target)
					if err != nil {
						efficientLogger.Error("Request error", "error", err)
						return
					}
					if metrics.Response.StatusCode != http.StatusOK {
						efficientLogger.Error("Failed", "status", metrics.Response.StatusCode, "TTFB", metrics.TTFB, "Total", metrics.Total, "DNS", metrics.DNSTime, "Connect", metrics.ConnectTime, "TLS", metrics.TLSTime)
						return
					}
					body, err := io.ReadAll(metrics.Response.Body)
					if err != nil {
						efficientLogger.Error("Failed to read response body", "error", err)
					}
					efficientLogger.Info("Success", "TTFB", metrics.TTFB, "Total", metrics.Total, "isCold", string(body), "status", metrics.Response.StatusCode, "DNS", metrics.DNSTime, "Connect", metrics.ConnectTime, "TLS", metrics.TLSTime)

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
	cfg         *config.Config
	event       *cloudevents.Event
	Pool        connection.Pool
	limiter     *rate.Limiter
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	logger      *slog.Logger
	currentRate float64
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
	ctx, cancel := context.WithCancel(context.Background())
	return &cloudEventGenerator{
		cfg:    cfg,
		event:  event,
		Pool:   pool,
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}
}

func (c *cloudEventGenerator) run() error {

	c.logger.Info("Starting workload generation", "rate", c.cfg.Rate.RequestsPerSecond)
	c.logger.Info("Using cloudevent", "event", c.event)

	interval := time.Duration(float64(time.Second) / c.cfg.Rate.RequestsPerSecond)
	c.logger.Info("Calculated ticker interval", "interval", interval)

	if c.cfg.Rate.RequestsPerSecond > 1 {
		if err := c.rampUp(); err != nil {
			return fmt.Errorf("ramp-up failed: %w", err)
		}
	}

	ticker := time.NewTicker(interval)
	startTime := time.Now()
	target := c.cfg.Targets[0]
	efficientLogger := c.logger.With("target", target.URL)
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("Generator stopped")
			return nil
		case <-ticker.C:
			c.wg.Add(1)
			go func(target *config.Target) {
				defer c.wg.Done()
				// A unique id allows per request comparison
				// There should be no problem with concurrency here
				id := strconv.Itoa(int(new(maphash.Hash).Sum64()))
				event := c.event.Clone()
				event.SetID(id)

				metrics, err := c.Pool.GenerateCloudEvent(target, &event)
				if err != nil {
					if metrics == nil {
						efficientLogger.Error("Failed with no metrics", "error", err)
						return
					}
					efficientLogger.Error("Failed",
						"id", id,
						"error", err,
						"TTFB", metrics.TTFB,
						"Total", metrics.Total,
						"DNS", metrics.DNSTime,
						"Connect", metrics.ConnectTime,
						"TLS", metrics.TLSTime,
					)
					return
				}
				body, err := io.ReadAll(metrics.Response.Body)
				if err != nil {
					efficientLogger.Error("Failed to read response body", "error", err)
				}
				defer metrics.Response.Body.Close()
				efficientLogger.Info("Success", "id", id, "TTFB", metrics.TTFB, "Total", metrics.Total, "isCold", string(body), "status", metrics.Response.StatusCode, "DNS", metrics.DNSTime, "Connect", metrics.ConnectTime, "TLS", metrics.TLSTime)
			}(target)
		}
		if c.cfg.Rate.Duration.Duration > 0 && time.Since(startTime) > c.cfg.Rate.Duration.Duration {
			c.logger.Info("Duration reached", "duration", c.cfg.Rate.Duration.Duration)
			c.Stop()
			return nil
		}
	}

}
func (c *cloudEventGenerator) rampUp() error {
	if c.cfg.Rate.RequestsPerSecond <= 1 {
		return nil
	}

	c.logger.Info("Starting 15-second ramp-up", "targetRate", c.cfg.Rate.RequestsPerSecond)

	steps := 15 // one step per second
	rateIncrement := (c.cfg.Rate.RequestsPerSecond - 1) / float64(steps)
	currentRate := 1.0

	for i := 0; i < steps; i++ {
		select {
		case <-c.ctx.Done():
			return nil
		default:
			currentRate += rateIncrement
			// Calculate appropriate burst size based on rate
			burstSize := int(currentRate)
			if burstSize < 1 {
				burstSize = 1
			}

			localRateLimiter := rate.NewLimiter(rate.Limit(currentRate), burstSize)
			c.currentRate = currentRate

			// Create a ticker for the current rate
			interval := time.Duration(float64(time.Second) / currentRate)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			stepTimer := time.NewTimer(time.Second)
			defer stepTimer.Stop()

		stepLoop:
			for {
				select {
				case <-c.ctx.Done():
					return nil
				case <-stepTimer.C:
					break stepLoop
				case <-ticker.C:
					if err := localRateLimiter.Wait(c.ctx); err != nil {
						return err
					}

					// Launch concurrent requests for each target
					for _, target := range c.cfg.Targets {
						c.wg.Add(1)
						go func(t *config.Target) {
							defer c.wg.Done()
							if err := c.sendRequest(t); err != nil {
								c.logger.Error("Request failed", "error", err)
							}
						}(target)
					}
				}
			}

			c.logger.Info("Ramp-up progress", "currentRate", currentRate)
		}
	}

	c.logger.Info("Ramp-up complete")
	return nil
}

func (c *cloudEventGenerator) sendRequest(target *config.Target) error {
	id := strconv.Itoa(int(new(maphash.Hash).Sum64()))
	event := c.event.Clone()
	event.SetID(id)
	efficientLogger := c.logger.With("target", target.URL)
	metrics, err := c.Pool.GenerateCloudEvent(target, &event)
	if err != nil {
		efficientLogger.Error("Request error", "error", err)
		return err
	}

	body, err := io.ReadAll(metrics.Response.Body)
	if err != nil {
		efficientLogger.Error("Failed to read response body", "error", err)
		return err
	}
	defer metrics.Response.Body.Close()

	efficientLogger.Info("Success",
		"id", id,
		"TTFB", metrics.TTFB,
		"Total", metrics.Total,
		"isCold", string(body),
		"status", metrics.Response.StatusCode,
		"DNS", metrics.DNSTime,
		"Connect", metrics.ConnectTime,
		"TLS", metrics.TLSTime)

	return nil
}

func (c *cloudEventGenerator) runColdStart() error {
	return nil
}

func (c *cloudEventGenerator) StartColdStart() error {
	return nil
}
