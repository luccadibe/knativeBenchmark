package generator

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"workload-generator/internal/config"
	"workload-generator/internal/connection"
)

type Generator struct {
	cfg     *config.Config
	Pool    connection.Pool
	limiter *rate.Limiter
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	logger  *slog.Logger
}

func New(cfg *config.Config, logger *slog.Logger, pool connection.Pool) *Generator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Generator{
		cfg:     cfg,
		Pool:    pool,
		limiter: rate.NewLimiter(rate.Limit(cfg.Rate.RequestsPerSecond), 1),
		ctx:     ctx,
		cancel:  cancel,
		logger:  logger,
	}
}

func (g *Generator) Start() error {
	err := g.run()
	g.wg.Wait()
	return err
}

func (g *Generator) Stop() {
	g.cancel()
	g.wg.Wait()
}

func (g *Generator) run() error {
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

func (g *Generator) runMaxThroughput() error {
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
