package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

type Storage interface {
	StoreNodeMetrics(nodeMetrics NodeMetrics) error
	StorePodMetrics(podMetrics PodMetrics) error
	Close() error
}

type NodeMetrics struct {
	Name             string
	CPUUsage         string
	MemoryUsage      string
	CPUPercentage    float64
	MemoryPercentage float64
	Timestamp        string
}

type PodMetrics struct {
	PodName            string
	NodeName           string
	ContainerName      string
	CPUUsage           string
	MemoryUsage        string
	CPUPercentage      float64
	MemoryPercentage   float64
	Timestamp          string
	ContainerTimestamp string
	ContainerWindow    string
}

var (
	storageType      string
	frequency        int
	allNamespaces    bool
	namespaces       []string
	ignoreNamespaces []string
	collectNodes     bool
	collectPods      bool
	outputDir        string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "k8s-metrics-collector",
		Short: "Collect Kubernetes node and pod metrics",
		Long:  "A tool to collect Kubernetes node and pod metrics and store them in CSV files or SQLite database.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := run(); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().IntVarP(&frequency, "frequency", "f", 5, "Frequency in seconds to pull metrics")
	rootCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", true, "Collect metrics from all namespaces")
	rootCmd.Flags().StringSliceVarP(&namespaces, "namespace", "n", []string{}, "Specific namespaces to collect metrics from")
	rootCmd.Flags().StringSliceVarP(&ignoreNamespaces, "ignore-namespace", "i", []string{}, "Namespaces to ignore")
	rootCmd.Flags().BoolVar(&collectNodes, "nodes", true, "Collect node metrics")
	rootCmd.Flags().BoolVar(&collectPods, "pods", true, "Collect pod metrics")
	rootCmd.Flags().StringVarP(&outputDir, "output-dir", "o", ".", "Output directory for CSV files or SQLite database")
	rootCmd.Flags().StringVarP(&storageType, "storage", "s", "csv", "Storage type (csv or sqlite)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	// Load Kubernetes config
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %v", err)
	}

	// Create Metrics clientset
	metricsClient, err := metricsclientset.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Metrics clientset: %v", err)
	}

	// Create storage based on the storage type
	var storage Storage
	switch storageType {
	case "csv":
		storage = NewCSVStorage(outputDir)
	case "sqlite":
		sqliteStorage, err := NewSQLiteStorage(outputDir)
		if err != nil {
			return fmt.Errorf("failed to create SQLite storage: %v", err)
		}
		storage = sqliteStorage
	default:
		return fmt.Errorf("unsupported storage type: %s", storageType)
	}
	defer storage.Close()

	// Run the metrics collection loop
	ticker := time.NewTicker(time.Duration(frequency) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if collectNodes {
			if err := collectNodeMetrics(clientset, metricsClient, storage); err != nil {
				fmt.Printf("Error collecting node metrics: %v\n", err)
				continue
			}
		}

		if collectPods {
			if err := collectPodMetrics(clientset, metricsClient, storage); err != nil {
				fmt.Printf("Error collecting pod metrics: %v\n", err)
				continue
			}
		}
	}

	return nil
}

func collectNodeMetrics(clientset *kubernetes.Clientset, metricsClient *metricsclientset.Clientset, storage Storage) error {
	// Fetch node metrics
	nodeMetricsList, err := metricsClient.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to fetch node metrics: %v", err)
	}

	// Fetch node allocatable resources
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to fetch nodes: %v", err)
	}

	nodeAllocatable := make(map[string]corev1.ResourceList)
	for _, node := range nodes.Items {
		nodeAllocatable[node.Name] = node.Status.Allocatable
	}

	// Write metrics to storage
	for _, nodeMetrics := range nodeMetricsList.Items {
		cpuUsage := nodeMetrics.Usage[corev1.ResourceCPU]
		memoryUsage := nodeMetrics.Usage[corev1.ResourceMemory]

		allocatable := nodeAllocatable[nodeMetrics.Name]
		cpuAllocatable := allocatable[corev1.ResourceCPU]
		memoryAllocatable := allocatable[corev1.ResourceMemory]

		cpuPercentage := float64(cpuUsage.MilliValue()) / float64(cpuAllocatable.MilliValue())
		memoryPercentage := float64(memoryUsage.Value()) / float64(memoryAllocatable.Value())

		metrics := NodeMetrics{
			Name:             nodeMetrics.Name,
			CPUUsage:         cpuUsage.String(),
			MemoryUsage:      memoryUsage.String(),
			CPUPercentage:    cpuPercentage,
			MemoryPercentage: memoryPercentage,
			Timestamp:        time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		}

		if err := storage.StoreNodeMetrics(metrics); err != nil {
			return fmt.Errorf("failed to store node metrics: %v", err)
		}
	}

	return nil
}

func collectPodMetrics(clientset *kubernetes.Clientset, metricsClient *metricsclientset.Clientset, storage Storage) error {
	namespacesToCollect := namespaces
	if allNamespaces {
		namespacesList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to fetch namespaces: %v", err)
		}
		for _, ns := range namespacesList.Items {
			if !contains(ignoreNamespaces, ns.Name) {
				namespacesToCollect = append(namespacesToCollect, ns.Name)
			}
		}
	}

	for _, ns := range namespacesToCollect {
		// Fetch pod metrics
		podMetricsList, err := metricsClient.MetricsV1beta1().PodMetricses(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to fetch pod metrics for namespace %s: %v", ns, err)
		}

		// Write metrics to storage
		for _, podMetrics := range podMetricsList.Items {
			pod, err := clientset.CoreV1().Pods(ns).Get(context.TODO(), podMetrics.Name, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get pod: %v", err)
			}

			nodeName := pod.Spec.NodeName
			if nodeName == "" {
				fmt.Printf("Warning: Pod %s/%s is not scheduled to a node\n", ns, podMetrics.Name)
				continue
			}

			node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to fetch node %s for pod %s: %v", nodeName, podMetrics.Name, err)
			}

			allocatable := node.Status.Allocatable
			cpuAllocatable := allocatable[corev1.ResourceCPU]
			memoryAllocatable := allocatable[corev1.ResourceMemory]

			containerTimestamp := podMetrics.Timestamp
			containerWindow := podMetrics.Window

			for _, container := range podMetrics.Containers {
				cpuUsage := container.Usage[corev1.ResourceCPU]
				memoryUsage := container.Usage[corev1.ResourceMemory]

				cpuPercentage := float64(cpuUsage.MilliValue()) / float64(cpuAllocatable.MilliValue())
				memoryPercentage := float64(memoryUsage.Value()) / float64(memoryAllocatable.Value())

				metrics := PodMetrics{
					PodName:            podMetrics.Name,
					NodeName:           nodeName,
					ContainerName:      container.Name,
					CPUUsage:           cpuUsage.String(),
					MemoryUsage:        memoryUsage.String(),
					CPUPercentage:      cpuPercentage,
					MemoryPercentage:   memoryPercentage,
					Timestamp:          time.Now().Format(time.RFC3339),
					ContainerTimestamp: containerTimestamp.Format(time.RFC3339),
					ContainerWindow:    containerWindow.String(),
				}

				if err := storage.StorePodMetrics(metrics); err != nil {
					return fmt.Errorf("failed to store pod metrics: %v", err)
				}
			}
		}
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

type CSVStorage struct {
	outputDir string
}

func NewCSVStorage(outputDir string) *CSVStorage {
	return &CSVStorage{outputDir: outputDir}
}

func (s *CSVStorage) StoreNodeMetrics(metrics NodeMetrics) error {
	filePath := filepath.Join(s.outputDir, "node_metrics.csv")
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open node_metrics.csv: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if the file is empty
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}
	if fileInfo.Size() == 0 {
		header := []string{"node_name", "cpu_usage", "memory_usage", "cpu_percentage", "memory_percentage", "timestamp"}
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write header to CSV: %v", err)
		}
	}

	record := []string{
		metrics.Name,
		metrics.CPUUsage,
		metrics.MemoryUsage,
		fmt.Sprintf("%.4f", metrics.CPUPercentage),
		fmt.Sprintf("%.4f", metrics.MemoryPercentage),
		metrics.Timestamp,
	}
	if err := writer.Write(record); err != nil {
		return fmt.Errorf("failed to write node metrics to CSV: %v", err)
	}

	return nil
}

func (s *CSVStorage) StorePodMetrics(metrics PodMetrics) error {
	filePath := filepath.Join(s.outputDir, fmt.Sprintf("%s_pod_metrics.csv", metrics.PodName))
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open pod metrics CSV: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if the file is empty
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}
	if fileInfo.Size() == 0 {
		header := []string{"pod_name", "node_name", "container_name", "cpu_usage", "memory_usage", "cpu_percentage", "memory_percentage", "timestamp"}
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write header to CSV: %v", err)
		}
	}

	record := []string{
		metrics.PodName,
		metrics.NodeName,
		metrics.ContainerName,
		metrics.CPUUsage,
		metrics.MemoryUsage,
		fmt.Sprintf("%.4f", metrics.CPUPercentage),
		fmt.Sprintf("%.4f", metrics.MemoryPercentage),
		metrics.Timestamp,
	}
	if err := writer.Write(record); err != nil {
		return fmt.Errorf("failed to write pod metrics to CSV: %v", err)
	}

	return nil
}

func (s *CSVStorage) Close() error {
	return nil
}

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(outputDir string) (*SQLiteStorage, error) {
	// Ensure the output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	// Open or create the SQLite database
	dbPath := filepath.Join(outputDir, "metrics.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %v", err)
	}

	// Create tables if they don't exist
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return &SQLiteStorage{db: db}, nil
}

func createTables(db *sql.DB) error {
	// Create node_metrics table
	nodeMetricsTable := `
    CREATE TABLE IF NOT EXISTS node_metrics (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        node_name TEXT NOT NULL,
        cpu_usage TEXT NOT NULL,
        memory_usage TEXT NOT NULL,
        cpu_percentage REAL NOT NULL,
        memory_percentage REAL NOT NULL,
        timestamp TEXT NOT NULL
    );`
	if _, err := db.Exec(nodeMetricsTable); err != nil {
		return fmt.Errorf("failed to create node_metrics table: %v", err)
	}

	// Create pod_metrics table
	podMetricsTable := `
    CREATE TABLE IF NOT EXISTS pod_metrics (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        pod_name TEXT NOT NULL,
        node_name TEXT NOT NULL,
        container_name TEXT NOT NULL,
        cpu_usage TEXT NOT NULL,
        memory_usage TEXT NOT NULL,
        cpu_percentage REAL NOT NULL,
        memory_percentage REAL NOT NULL,
        timestamp TEXT NOT NULL
    );`
	if _, err := db.Exec(podMetricsTable); err != nil {
		return fmt.Errorf("failed to create pod_metrics table: %v", err)
	}

	return nil
}

func (s *SQLiteStorage) StoreNodeMetrics(metrics NodeMetrics) error {
	query := `
    INSERT INTO node_metrics (node_name, cpu_usage, memory_usage, cpu_percentage, memory_percentage, timestamp)
    VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, metrics.Name, metrics.CPUUsage, metrics.MemoryUsage, metrics.CPUPercentage, metrics.MemoryPercentage, metrics.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to store node metrics: %v", err)
	}
	return nil
}

func (s *SQLiteStorage) StorePodMetrics(metrics PodMetrics) error {
	query := `
    INSERT INTO pod_metrics (pod_name, node_name, container_name, cpu_usage, memory_usage, cpu_percentage, memory_percentage, timestamp)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, metrics.PodName, metrics.NodeName, metrics.ContainerName, metrics.CPUUsage, metrics.MemoryUsage, metrics.CPUPercentage, metrics.MemoryPercentage, metrics.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to store pod metrics: %v", err)
	}
	return nil
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
