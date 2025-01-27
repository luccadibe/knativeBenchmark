// main.go
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type config struct {
	dbPath     string
	logDir     string
	timeWindow time.Duration
}

type experimentInfo struct {
	timestamp time.Time
	language  string
	scenario  string
	params    map[string]int
}

type request struct {
	timestamp time.Time
	status    int
	ttfb      float64
	total     float64
	isCold    bool
	dns       float64
	connect   float64
	tls       float64
}

type processingStats struct {
	filesProcessed      int
	experimentsInserted int
	requestsInserted    int
}

var knownLanguages = map[string]bool{
	"go":         true,
	"python":     true,
	"rust":       true,
	"typescript": true,
	"ts":         true,
	"quarkus":    true,
	"springboot": true,
}

var (
	reRequestsPerSecond = regexp.MustCompile(`RequestsPerSecond:(\d+)`)
	reDuration          = regexp.MustCompile(`Duration:([\d\w]+)`)
	reMaxIdleConns      = regexp.MustCompile(`MaxIdleConns:(\d+)`)
	reMaxIdleConnsHost  = regexp.MustCompile(`MaxIdleConnsPerHost:(\d+)`)
	reIdleConnTimeout   = regexp.MustCompile(`IdleConnTimeout:([\d\w]+)`)
	reTimeout           = regexp.MustCompile(`Timeout:([\d\w]+)`)
)

func main() {
	cfg := parseFlags()
	db := initDB(cfg.dbPath)
	defer db.Close()

	stats, err := processLogs(db, cfg)
	if err != nil {
		log.Fatalf("Error processing logs: %v", err)
	}

	log.Printf("Processing complete. Files: %d, Experiments: %d, Requests: %d",
		stats.filesProcessed, stats.experimentsInserted, stats.requestsInserted)
}

func parseFlags() config {
	c := config{}
	flag.StringVar(&c.dbPath, "db", "benchmark.db", "SQLite database path")
	flag.StringVar(&c.logDir, "logs", "./logs", "Log directory path")
	timeWindow := flag.Int("hours", 10, "Processing time window in hours")
	flag.Parse()

	c.timeWindow = time.Duration(*timeWindow) * time.Hour
	return c
}

func initDB(dbPath string) *sql.DB {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS experiments (
			id INTEGER PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			language TEXT NOT NULL,
			scenario TEXT NOT NULL,
			concurrency INTEGER,
			rps INTEGER,
			requests_per_second INTEGER,
			duration TEXT,
			max_idle_conns INTEGER,
			max_idle_conns_per_host INTEGER,
			idle_conn_timeout TEXT,
			timeout TEXT
		);

		CREATE TABLE IF NOT EXISTS requests (
			id INTEGER PRIMARY KEY,
			experiment_id INTEGER NOT NULL,
			timestamp DATETIME NOT NULL,
			status INTEGER NOT NULL,
			ttfb REAL NOT NULL,
			total_time REAL NOT NULL,
			is_cold BOOLEAN NOT NULL,
			dns_time REAL NOT NULL,
			connect_time REAL NOT NULL,
			tls_time REAL NOT NULL,
			FOREIGN KEY(experiment_id) REFERENCES experiments(id)
		);
	`)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func processLogs(db *sql.DB, cfg config) (*processingStats, error) {
	stats := &processingStats{}
	cutoff := time.Now().Add(-cfg.timeWindow)

	entries, err := os.ReadDir(cfg.logDir)
	if err != nil {
		return nil, fmt.Errorf("reading log directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		expInfo, err := parseFilename(entry.Name())
		if err != nil {
			log.Printf("Skipping invalid filename %q: %v", entry.Name(), err)
			continue
		}

		if expInfo.timestamp.Before(cutoff) {
			continue
		}

		filePath := filepath.Join(cfg.logDir, entry.Name())
		config, requests, err := processFile(filePath)
		if err != nil {
			log.Printf("Error processing %q: %v", entry.Name(), err)
			continue
		}

		expID, err := insertExperiment(db, expInfo, config)
		if err != nil {
			log.Printf("Error inserting experiment: %v", err)
			continue
		}

		if len(requests) > 0 {
			err = insertRequests(db, expID, requests)
			if err != nil {
				log.Printf("Error inserting requests: %v", err)
				continue
			}
		}

		stats.filesProcessed++
		stats.experimentsInserted++
		stats.requestsInserted += len(requests)
	}

	return stats, nil
}

func parseFilename(filename string) (*experimentInfo, error) {
	base := strings.TrimSuffix(filename, ".log")
	parts := strings.Split(base, "_")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid filename format")
	}

	timestamp, err := time.Parse(
		"2006-01-02 15-04-05",
		fmt.Sprintf("%s %s", parts[len(parts)-2], parts[len(parts)-1]),
	)
	if err != nil {
		return nil, fmt.Errorf("parsing timestamp: %w", err)
	}

	prefix := strings.Join(parts[:len(parts)-2], "_")
	segments := strings.Split(prefix, "-")[1:] // Skip "serving"

	var language string
	var scenarioParts []string
	params := make(map[string]int)

	for i, seg := range segments {
		if knownLanguages[seg] {
			language = seg
			scenarioParts = segments[:i]
			paramParts := segments[i+1:]
			for j := 0; j < len(paramParts); j += 2 {
				if j+1 >= len(paramParts) {
					break
				}
				val, _ := strconv.Atoi(paramParts[j+1])
				params[paramParts[j]] = val
			}
			break
		}
	}

	if language == "" {
		return nil, fmt.Errorf("language not found in filename")
	}

	return &experimentInfo{
		timestamp: timestamp.UTC(),
		language:  language,
		scenario:  strings.Join(scenarioParts, "-"),
		params:    params,
	}, nil
}

func processFile(path string) (map[string]interface{}, []request, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 {
		return nil, nil, fmt.Errorf("invalid log file format")
	}

	config := parseConfig(lines[1])
	var requests []request

	for _, line := range lines[2:] {
		if strings.Contains(line, `msg=Success`) {
			req, err := parseRequest(line)
			if err == nil {
				requests = append(requests, req)
			}
		}
	}

	return config, requests, nil
}

func parseConfig(line string) map[string]interface{} {
	config := make(map[string]interface{})
	matches := reRequestsPerSecond.FindStringSubmatch(line)
	if len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			config["requests_per_second"] = val
		}
	}

	addConfigValue(line, reDuration, "duration", config)
	addConfigValue(line, reMaxIdleConns, "max_idle_conns", config)
	addConfigValue(line, reMaxIdleConnsHost, "max_idle_conns_per_host", config)
	addConfigValue(line, reIdleConnTimeout, "idle_conn_timeout", config)
	addConfigValue(line, reTimeout, "timeout", config)

	return config
}

func addConfigValue(line string, re *regexp.Regexp, key string, config map[string]interface{}) {
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		config[key] = matches[1]
	}
}

func parseRequest(line string) (request, error) {
	var req request
	pairs := parseKeyValuePairs(line)

	if t, ok := pairs["time"]; ok {
		ts, err := time.Parse(time.RFC3339Nano, t)
		if err == nil {
			req.timestamp = ts.UTC()
		}
	}

	if s, ok := pairs["status"]; ok {
		status, _ := strconv.Atoi(s)
		req.status = status
	}

	req.isCold = pairs["isCold"] == "true"
	req.ttfb = parseDuration(pairs["TTFB"])
	req.total = parseDuration(pairs["Total"])
	req.dns = parseDuration(pairs["DNS"])
	req.connect = parseDuration(pairs["Connect"])
	req.tls = parseDuration(pairs["TLS"])

	return req, nil
}

func parseKeyValuePairs(line string) map[string]string {
	result := make(map[string]string)
	re := regexp.MustCompile(`(\w+)=("[^"]*"|\S+)`)
	matches := re.FindAllStringSubmatch(line, -1)

	for _, m := range matches {
		key := m[1]
		value := strings.Trim(m[2], `"`)
		result[key] = value
	}

	return result
}

func parseDuration(s string) float64 {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return float64(d.Nanoseconds()) / 1e6
}

func insertExperiment(db *sql.DB, exp *experimentInfo, config map[string]interface{}) (int64, error) {
	stmt := `
		INSERT INTO experiments (
			timestamp, language, scenario, concurrency, rps,
			requests_per_second, duration, max_idle_conns,
			max_idle_conns_per_host, idle_conn_timeout, timeout
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	args := []interface{}{
		exp.timestamp.Format(time.RFC3339),
		exp.language,
		exp.scenario,
		exp.params["concurrency"],
		exp.params["rps"],
		config["requests_per_second"],
		config["duration"],
		config["max_idle_conns"],
		config["max_idle_conns_per_host"],
		config["idle_conn_timeout"],
		config["timeout"],
	}

	res, err := db.Exec(stmt, args...)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func insertRequests(db *sql.DB, expID int64, requests []request) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO requests (
			experiment_id, timestamp, status, ttfb, total_time,
			is_cold, dns_time, connect_time, tls_time
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, req := range requests {
		_, err := stmt.Exec(
			expID,
			req.timestamp.Format(time.RFC3339Nano),
			req.status,
			req.ttfb,
			req.total,
			req.isCold,
			req.dns,
			req.connect,
			req.tls,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
