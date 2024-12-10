package generator

import (
	"context"
	"log"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"workload-generator/internal/config"
	"workload-generator/internal/connection"
)

type Generator struct {
	cfg     *config.Config
	pool    *connection.Pool
	limiter *rate.Limiter
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func New(cfg *config.Config) *Generator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Generator{
		cfg:     cfg,
		pool:    connection.NewPool(cfg.BaseURL),
		limiter: rate.NewLimiter(rate.Limit(cfg.Rate.RequestsPerSecond), 1),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (g *Generator) Start() error {
	g.wg.Add(1)
	go g.run()
	g.wg.Wait()
	return nil
}

func (g *Generator) Stop() {
	g.cancel()
	g.wg.Wait()
}

func (g *Generator) run() error {
	defer g.wg.Done()
	log.Printf("Starting workload generation with rate: %d req/s", g.cfg.Rate.RequestsPerSecond)

	interval := time.Second / time.Duration(g.cfg.Rate.RequestsPerSecond)
	log.Printf("Calculated ticker interval: %v", interval)

	ticker := time.NewTicker(interval)
	startTime := time.Now()
	requestCount := 0

	for {
		select {
		case <-g.ctx.Done():
			log.Printf("Generator stopped. Total requests: %d", requestCount)
			return nil
		case <-ticker.C:
			//log.Printf("Tick at %v", time.Now())
			if err := g.limiter.Wait(g.ctx); err != nil {
				log.Printf("Rate limiter error: %v", err)
				return err
			}

			for _, target := range g.cfg.Targets {
				g.wg.Add(1)
				go func(t config.Target) {
					defer g.wg.Done()
					start := time.Now()
					resp, err := g.pool.Get(t)
					if err != nil {
						log.Printf("Request error for %s: %v", t.URL, err)
						return
					}
					defer resp.Body.Close()

					duration := time.Since(start)
					log.Printf("Request to %s completed in %v with status %d",
						t.URL, duration, resp.StatusCode)
					requestCount++
				}(target)
			}

			if g.cfg.Rate.Duration.Duration > 0 && time.Since(startTime) > g.cfg.Rate.Duration.Duration {
				log.Printf("Duration %v reached, stopping generator", g.cfg.Rate.Duration.Duration)
				g.Stop()
				return nil
			}
		}
	}
}
