package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned"
)

func main() {
	kubeconfig := flag.String("kubeconfig", filepath.Join(os.Getenv("HOME"), ".kube", "config"), "path to kubeconfig file")
	logfile := flag.String("logfile", "metrics.log", "path to log file")
	interval := flag.Duration("interval", 5*time.Second, "collection interval")
	flag.Parse()

	log.Printf("Starting metrics collection with interval %s", *interval)

	// Setup logging
	f, err := os.OpenFile(*logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("failed to open log file", "error", err)
		os.Exit(1)
	}
	defer f.Close()

	logger := slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create metrics client
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		logger.Error("failed to create k8s config", "error", err)
		os.Exit(1)
	}

	metricsClient, err := metricsv1beta1.NewForConfig(config)
	if err != nil {
		logger.Error("failed to create metrics client", "error", err)
		os.Exit(1)
	}

	// Collection loop
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for range ticker.C {
		// Get node metrics
		nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(context.Background(), metav1.ListOptions{
			Limit: 1000,
		})
		if err != nil {
			logger.Error("failed to get node metrics", "error", err)
		} else {
			for _, node := range nodeMetrics.Items {
				logger.Info("node metrics",
					"timestamp", time.Now().UTC(),
					"node", node.Name,
					"cpu", node.Usage.Cpu().MilliValue(),
					"memory_bytes", node.Usage.Memory().Value(),
				)
			}
		}

		// Get pod metrics across all namespaces
		podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses("").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			logger.Error("failed to get pod metrics", "error", err)
			continue
		}

		for _, pod := range podMetrics.Items {
			for _, container := range pod.Containers {
				logger.Info("container metrics",
					"timestamp", time.Now().UTC(),
					"namespace", pod.Namespace,
					"pod", pod.Name,
					"container", container.Name,
					"cpu", container.Usage.Cpu().MilliValue(),
					"memory_bytes", container.Usage.Memory().Value(),
				)
			}
		}
	}
}
